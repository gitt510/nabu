package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gitt510/nabu/internal/api"
)

// fakeAPI stands in for the Worker: it keeps tasks in memory and refuses
// what the real API refuses that nabu passes through, so the tests cover the
// CLI, not the API's rules.
type fakeAPI struct {
	tasks map[string]api.Task
	// last is the JSON body of the last write, as the CLI sent it.
	last map[string]any
	// query is the query string of the last list.
	query string
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("cf-access-token") != "token" {
		http.Redirect(w, r, "https://team.cloudflareaccess.com/cdn-cgi/access/login", http.StatusFound)
		return
	}
	refuse := func(status int, msg string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}
	slug := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		f.last = map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&f.last)
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
		f.query = r.URL.RawQuery
		out := []api.Task{}
		for _, t := range f.tasks {
			b, _ := json.Marshal(t)
			if (r.URL.Query().Get("status") == "" || t.Status == r.URL.Query().Get("status")) &&
				strings.Contains(strings.ToLower(string(b)), strings.ToLower(r.URL.Query().Get("q"))) {
				out = append(out, t)
			}
		}
		slices.SortFunc(out, func(a, b api.Task) int { return strings.Compare(a.Slug, b.Slug) })
		_ = json.NewEncoder(w).Encode(out)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/tasks":
		t := api.Task{Status: "inbox", Created: "2026-10-06T09:00:00+09:00", Updated: "2026-10-06T09:00:00+09:00"}
		if err := overlay(&t, f.last); err != nil {
			refuse(422, err.Error())
			return
		}
		if _, ok := f.tasks[t.Slug]; ok {
			refuse(409, t.Slug+" already exists")
			return
		}
		f.tasks[t.Slug] = t
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(t)
		return
	}
	t, ok := f.tasks[slug]
	if !ok {
		refuse(404, "no such task")
		return
	}
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(t)
	case http.MethodPatch:
		if err := overlay(&t, f.last); err != nil {
			refuse(422, err.Error())
			return
		}
		if _, taken := f.tasks[t.Slug]; taken && t.Slug != slug {
			refuse(409, t.Slug+" already exists")
			return
		}
		delete(f.tasks, slug)
		f.tasks[t.Slug] = t
		_ = json.NewEncoder(w).Encode(t)
	case http.MethodDelete:
		delete(f.tasks, slug)
		w.WriteHeader(204)
	}
}

// overlay writes fields onto t the way the API does, null dropping a value,
// and applies the two rules a CLI test can reach: the line limit and the
// reason a dropped task needs.
func overlay(t *api.Task, fields map[string]any) error {
	b, _ := json.Marshal(t)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for k, v := range fields {
		m[k] = v
	}
	b, _ = json.Marshal(m)
	*t = api.Task{}
	_ = json.Unmarshal(b, t)
	for _, l := range slices.Concat(t.Situation, t.Complication, t.Question, t.Answer, t.WhyDropped) {
		if utf8.RuneCountInString(l) > 30 {
			return errors.New("situation.0: must be at most 30 characters")
		}
	}
	if t.Status == "dropped" && len(t.WhyDropped) == 0 {
		return errors.New("a dropped task needs why_dropped")
	}
	return nil
}

// newAPI starts a fake API, points nabu at it with a valid token, and
// returns it with a runner that invokes nabu and fails the test on the wrong
// exit code. The runner returns stdout.
func newAPI(t *testing.T) (*fakeAPI, func(want int, stdin string, args ...string) string) {
	t.Helper()
	f := &fakeAPI{tasks: map[string]api.Task{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	useClient(t, srv.URL, func(string) (string, error) { return "token", nil })
	return f, runner(t)
}

func useClient(t *testing.T, url string, token func(string) (string, error)) {
	t.Helper()
	orig := client
	client = func() (*api.Client, error) {
		c := api.New(url)
		c.Token = token
		return c, nil
	}
	t.Cleanup(func() { client = orig })
}

func runner(t *testing.T) func(want int, stdin string, args ...string) string {
	return func(want int, stdin string, args ...string) string {
		t.Helper()
		var out, errb bytes.Buffer
		if code := run(args, strings.NewReader(stdin), &out, &errb); code != want {
			t.Fatalf("%v: exit %d want %d: %s%s", args, code, want, out.String(), errb.String())
		}
		return out.String()
	}
}

// stderrOf runs nabu and returns its exit code and stderr.
func stderrOf(args []string, stdin string) (int, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, errb.String()
}

// body is a task body in the shape, with answer under ## Answer.
func body(title, answer string) string {
	return "# " + title + "\n\n## Situation\n\n- 本番と検証が旧 runtime\n\n## Complication\n\n- 旧 runtime はサポートが切れる\n\n## Question\n\n- 期限前に止めずに移れる？\n\n## Answer\n\n" + answer
}

func TestTaskHelpGroupsCommands(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"task", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got := out.String()
	at := func(s string) int {
		i := strings.Index(got, s)
		if i < 0 {
			t.Fatalf("task -h lacks %q:\n%s", s, got)
		}
		return i
	}
	if !(at("Browse:") < at("  ls ") && at("  ls ") < at("Read content:") &&
		at("Read content:") < at("  read ") && at("  read ") < at("Create and update:") && at("Create and update:") < at("  new ")) {
		t.Fatalf("commands are not under their groups:\n%s", got)
	}
}

func TestGroupAloneExits2(t *testing.T) {
	for _, args := range [][]string{{}, {"task"}, {"task", "bogus"}, {"task", "ls", "--nope"}, {"task", "read"}, {"bogus"}} {
		var out, errb bytes.Buffer
		if code := run(args, strings.NewReader(""), &out, &errb); code != exitUsage || out.Len() != 0 || !strings.Contains(errb.String(), "Usage:") {
			t.Fatalf("%v: exit %d stdout=%q stderr=%q", args, code, out.String(), errb.String())
		}
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"task", "new", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "nabu task new <slug> [flags]") || errb.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errb.String())
	}
}

func TestNoURLExits2(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, stderr := stderrOf([]string{"task", "ls"}, "")
	if code != exitUsage || !strings.Contains(stderr, "no url declared") || !strings.Contains(stderr, "cloudflared access login") {
		t.Fatalf("exit %d stderr=%q", code, stderr)
	}
}

func TestConfigURLReachesTheClient(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "nabu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nabu", "config.json"), []byte(`{"url": "https://nabu.example.dev/"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := client()
	if err != nil || c.URL != "https://nabu.example.dev" {
		t.Fatalf("client %+v, %v", c, err)
	}
}

func TestSignedOutExits2(t *testing.T) {
	f := &fakeAPI{tasks: map[string]api.Task{}}
	srv := httptest.NewServer(f)
	defer srv.Close()

	// Access redirects a request whose token it does not accept
	useClient(t, srv.URL, func(string) (string, error) { return "expired", nil })
	code, stderr := stderrOf([]string{"task", "ls"}, "")
	if code != exitUsage || stderr != "not signed in: run cloudflared access login "+srv.URL+"\n" {
		t.Fatalf("redirect: exit %d stderr=%q", code, stderr)
	}

	// cloudflared holds no token
	useClient(t, srv.URL, func(u string) (string, error) { return "", &api.SignedOutError{URL: u} })
	if code, stderr := stderrOf([]string{"task", "read", "x"}, ""); code != exitUsage || !strings.Contains(stderr, "cloudflared access login") {
		t.Fatalf("no token: exit %d stderr=%q", code, stderr)
	}
}

func TestTaskNewViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	ok := body("Ok", "- 未定\n")
	for _, args := range [][]string{
		{"task", "new", "Bad Slug", "--area", "work", "--content", ok},
		{"task", "new", "a/b", "--area", "work", "--content", ok},
		{"task", "new", "ok", "--content", ok},
		{"task", "new", "ok", "--area", "life", "--content", ok},
		{"task", "new", "ok", "--area", "work", "--content", ok, "--scheduled", "2026-09-15T18:00"},
		{"task", "new", "ok", "--area", "work", "--content", ok, "--ticket", "http://example.com/1"},
		{"task", "new", "ok", "--area", "work", "--content", ok, "--project", "Not Kebab"},
		{"task", "new", "ok", "--area", "work", "--content", "---\na: b\n---\n" + ok},
		{"task", "new", "ok", "--area", "work", "--content", "# Ok\n\n- 見出しの前\n"},
		{"task", "new", "ok", "--area", "work", "--content", body("Ok", "- x\n") + "\n## AI memo\n\nnotes\n"},
		{"task", "new", "ok", "--area", "work", "--content", body("Ok", "- "+strings.Repeat("あ", 29)+"\n")},
	} {
		nabu(exitUsage, "", args...)
	}
	if len(f.tasks) != 0 {
		t.Fatalf("a refused task was filed: %v", f.tasks)
	}

	got := nabu(exitOK, ok, "task", "new", "ok", "--area", "work", "--ticket", "https://example.com/1", "--ticket", "https://example.com/1")
	want := `{
  "action": "task new",
  "slug": "ok",
  "status": "inbox",
  "frontmatter": {
    "created": "2026-10-06T09:00:00+09:00",
    "area": "work",
    "tickets": [
      "https://example.com/1"
    ]
  }
}
`
	if got != want {
		t.Fatalf("new:\n%s", got)
	}
	// the body reaches the API as sections; what the task lacks is left out
	if f.last["title"] != "Ok" || !slices.Equal(toStrings(f.last["answer"]), []string{"- 未定"}) || !slices.Equal(toStrings(f.last["why_dropped"]), []string{}) {
		t.Fatalf("sent %v", f.last)
	}
	for _, k := range []string{"project", "scheduled", "waiting", "prs", "status", "created"} {
		if _, ok := f.last[k]; ok {
			t.Fatalf("new sent %s: %v", k, f.last)
		}
	}
	nabu(exitFail, ok, "task", "new", "ok", "--area", "work")
}

func toStrings(v any) []string {
	out := []string{}
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestTaskReadPrintsWhatNewTakes(t *testing.T) {
	_, nabu := newAPI(t)
	b := body("Ok", "- 検証 → 本番の順で切り替える\n")
	nabu(exitOK, b, "task", "new", "ok", "--area", "work", "--project", "nabu")
	want := "---\ncreated: \"2026-10-06T09:00:00+09:00\"\narea: \"work\"\nproject: \"nabu\"\n---\n" + b
	if got := nabu(exitOK, "", "task", "read", "ok"); got != want {
		t.Fatalf("read:\n%s", got)
	}
	nabu(exitFail, "", "task", "read", "missing")
	nabu(exitUsage, "", "task", "read", "Bad")
}

func TestTaskReplaceAndSetViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, body("Ok", "- 未定\n"), "task", "new", "ok", "--area", "work", "--project", "nabu", "--link", "https://example.com/a")

	if got := nabu(exitOK, body("Renamed", "- 決めた\n"), "task", "replace", "ok"); strings.Contains(got, "frontmatter") {
		t.Fatalf("replace prints frontmatter: %s", got)
	}
	if tk := f.tasks["ok"]; tk.Title != "Renamed" || tk.Answer[0] != "- 決めた" || tk.Project != "nabu" {
		t.Fatalf("after replace: %+v", tk)
	}
	for _, k := range []string{"area", "project", "status"} {
		if _, ok := f.last[k]; ok {
			t.Fatalf("replace sent %s", k)
		}
	}

	nabu(exitUsage, "", "task", "set", "ok")
	nabu(exitUsage, "", "task", "set", "ok", "--project", "x", "--clear-project")
	got := nabu(exitOK, "", "task", "set", "ok", "--clear-project", "--waiting", "review", "--link", "https://example.com/a", "--link", "https://example.com/b")
	if !strings.Contains(got, `"waiting": "review"`) || strings.Contains(got, `"project"`) {
		t.Fatalf("set: %s", got)
	}
	if v, ok := f.last["project"]; !ok || v != nil {
		t.Fatalf("set did not drop project: %v", f.last)
	}
	if tk := f.tasks["ok"]; !slices.Equal(tk.Links, []string{"https://example.com/a", "https://example.com/b"}) || tk.Title != "Renamed" {
		t.Fatalf("after set: %+v", tk)
	}
	nabu(exitFail, "", "task", "set", "missing", "--waiting", "x")
}

func TestTaskMvRenameRmViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, body("Ok", "- 未定\n"), "task", "new", "ok", "--area", "work")

	nabu(exitUsage, "", "task", "mv", "ok", "later")
	nabu(exitFail, "", "task", "mv", "ok", "inbox")
	if got := nabu(exitOK, "", "task", "mv", "ok", "doing"); !strings.Contains(got, `"from": "inbox"`) || !strings.Contains(got, `"to": "doing"`) {
		t.Fatalf("mv: %s", got)
	}

	// dropped needs the reason first
	code, stderr := stderrOf([]string{"task", "mv", "ok", "dropped"}, "")
	if code != exitUsage || !strings.Contains(stderr, "needs why_dropped") || !strings.Contains(stderr, "task replace first") {
		t.Fatalf("dropped: exit %d stderr=%q", code, stderr)
	}
	nabu(exitOK, body("Ok", "- やらない\n")+"\n## Why dropped\n\n- 今は使い道がない\n", "task", "replace", "ok")
	nabu(exitOK, "", "task", "mv", "ok", "dropped")
	if f.tasks["ok"].Status != "dropped" {
		t.Fatalf("not dropped: %+v", f.tasks["ok"])
	}

	nabu(exitUsage, "", "task", "rename", "ok", "ok")
	nabu(exitOK, body("Other", "")+"\n", "task", "new", "other", "--area", "work")
	nabu(exitFail, "", "task", "rename", "ok", "other")
	if got := nabu(exitOK, "", "task", "rename", "ok", "renamed"); !strings.Contains(got, `"to": "renamed"`) {
		t.Fatalf("rename: %s", got)
	}
	nabu(exitOK, "", "task", "rm", "renamed")
	nabu(exitFail, "", "task", "rm", "renamed")
	if _, ok := f.tasks["renamed"]; ok {
		t.Fatal("rm left the task")
	}
}

func TestTaskLsAndGrepViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, body("B doing", "- 未定\n"), "task", "new", "b", "--area", "work")
	nabu(exitOK, body("A inbox", "- PING them\n"), "task", "new", "a", "--area", "personal")
	nabu(exitOK, "", "task", "set", "a", "--waiting", "their reply")
	nabu(exitOK, body("C inbox", "- 未定\n"), "task", "new", "c", "--area", "work")
	nabu(exitOK, "", "task", "mv", "b", "doing")

	if got := nabu(exitOK, "", "task", "ls"); got != "inbox\tA inbox\t2026-10-06\ninbox\tC inbox\t2026-10-06\ndoing\tB doing\t2026-10-06\n" {
		t.Fatalf("ls: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "doing"); got != "doing\tB doing\t2026-10-06\n" {
		t.Fatalf("ls doing: %q", got)
	}
	nabu(exitUsage, "", "task", "ls", "later")

	var rows []taskEntry
	if err := json.Unmarshal([]byte(nabu(exitOK, "", "task", "ls", "--json")), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Slug != "a" || rows[0].Frontmatter.Waiting != "their reply" || rows[0].Body != body("A inbox", "- PING them\n") {
		t.Fatalf("ls --json: %+v", rows)
	}

	// grep searches the task as read prints it, frontmatter included
	if got := nabu(exitOK, "", "task", "grep", "ping"); got != "a:22: - PING them\n" {
		t.Fatalf("grep: %q", got)
	}
	if f.query != "q=ping" {
		t.Fatalf("grep query: %q", f.query)
	}
	if got := nabu(exitOK, "", "task", "grep", "their reply"); got != "a:4: waiting: \"their reply\"\n" {
		t.Fatalf("grep frontmatter: %q", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "nothing-here", "--json"); strings.TrimSpace(got) != "[]" {
		t.Fatalf("grep --json: %q", got)
	}
}

func TestParseBody(t *testing.T) {
	title, fields, err := parseBody([]byte("\n# T  \n\n## Situation\n- a\n\n- b\n## Complication\n## Question\n- q\n## Answer\n\n## Why dropped\n- w\n"))
	if err != nil || title != "T" {
		t.Fatalf("%q %v", title, err)
	}
	want := map[string][]string{"situation": {"- a", "- b"}, "complication": {}, "question": {"- q"}, "answer": {}, "why_dropped": {"- w"}}
	for k, v := range want {
		if !slices.Equal(fields[k].([]string), v) {
			t.Fatalf("%s: %v", k, fields[k])
		}
	}
	for _, bad := range []string{
		"",
		"## Situation\n",
		"# T\n\n## Complication\n## Situation\n## Question\n## Answer\n",
		"# T\n## Situation\n## Complication\n## Question\n",
		"# T\n## Situation\n## Complication\n## Question\n## Answer\n## Answer\n",
		"# T\n## Situation\n## Why dropped\n## Complication\n## Question\n## Answer\n",
		"# T\n## Situation\n## Complication\n## Question\n## Answer\n## Why dropped\n## Why dropped\n",
		"# T\nloose line\n## Situation\n## Complication\n## Question\n## Answer\n",
	} {
		if _, _, err := parseBody([]byte(bad)); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestCreatedDateIsLocal(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("JST", 9*60*60)
	defer func() { time.Local = orig }()
	for in, want := range map[string]string{
		"2026-10-18T23:30:00.000Z":  "2026-10-19",
		"2026-10-19T08:00:00+09:00": "2026-10-19",
		"someday":                   "-",
	} {
		if got := createdDate(in); got != want {
			t.Fatalf("%s: %s want %s", in, got, want)
		}
	}
}

func TestRenderBodyRoundTrips(t *testing.T) {
	tk := api.Task{Title: "T", Situation: []string{"- a"}, Answer: []string{"- x", "- y"}, WhyDropped: []string{"- w"}}
	title, fields, err := parseBody([]byte(renderBody(tk)))
	if err != nil || title != "T" || !slices.Equal(fields["answer"].([]string), tk.Answer) || !slices.Equal(fields["why_dropped"].([]string), tk.WhyDropped) {
		t.Fatalf("%q %v %v", title, fields, err)
	}
	if strings.Contains(renderBody(api.Task{Title: "T"}), "Why dropped") {
		t.Fatal("an empty reason is rendered")
	}
}
