package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	tasks map[int]api.Task
	// next is the id the last task created took.
	next int
	// last is the JSON body of the last write, as the CLI sent it.
	last map[string]any
	// items are the checklist lines added, in order.
	items []api.Item
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
	id := 0
	_, _ = fmt.Sscanf(r.URL.Path, "/api/tasks/%d", &id)
	if r.Method == http.MethodPost || r.Method == http.MethodPatch {
		f.last = map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&f.last)
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/schema":
		_, _ = w.Write([]byte(`{"create":{"type":"object"},"patch":{"type":"object"}}`))
		return
	case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
		out := []api.Task{}
		for _, t := range f.tasks {
			out = append(out, t)
		}
		// The API lists tasks in status order, then by id.
		order := []api.Status{api.Todo, api.Done, api.Canceled}
		slices.SortFunc(out, func(a, b api.Task) int {
			return cmp.Or(slices.Index(order, a.Status)-slices.Index(order, b.Status), a.Id-b.Id)
		})
		_ = json.NewEncoder(w).Encode(out)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/items":
		var it api.Item
		b, _ := json.Marshal(f.last)
		_ = json.Unmarshal(b, &it)
		if strings.TrimSpace(it.Text) == "" {
			refuse(422, "text: must not be blank")
			return
		}
		it.Id = len(f.items) + 1
		f.items = append(f.items, it)
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(it)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/api/items":
		out := []api.Item{}
		for _, it := range f.items {
			if fmt.Sprint(it.TaskId) == r.URL.Query().Get("task_id") {
				out = append(out, it)
			}
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/items/"):
		n := 0
		_, _ = fmt.Sscanf(r.URL.Path, "/api/items/%d", &n)
		if n < 1 || n > len(f.items) {
			refuse(404, "no such item")
			return
		}
		if done, ok := f.last["done"].(bool); ok {
			f.items[n-1].Done = done
		}
		_ = json.NewEncoder(w).Encode(f.items[n-1])
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/tasks":
		t := api.Task{Status: api.Todo, Created: "2026-10-06T09:00:00+09:00", Updated: "2026-10-06T09:00:00+09:00"}
		if err := overlay(&t, f.last); err != nil {
			refuse(422, err.Error())
			return
		}
		f.next++
		t.Id = f.next
		f.tasks[t.Id] = t
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(t)
		return
	}
	t, ok := f.tasks[id]
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
		f.tasks[id] = t
		_ = json.NewEncoder(w).Encode(t)
	case http.MethodDelete:
		delete(f.tasks, id)
		w.WriteHeader(204)
	}
}

// overlay writes fields onto t the way the API does, null dropping a value,
// and applies the two rules a CLI test can reach: the line limit and the
// reason only a canceled task carries.
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
	for _, l := range slices.Concat(t.WhyCanceled, t.Memo) {
		if utf8.RuneCountInString(l) > 45 {
			return errors.New("memo.0: must be at most 45 characters")
		}
	}
	if (t.Status == api.Canceled) != (len(t.WhyCanceled) > 0) {
		return errors.New("a canceled task needs why_canceled, and no other task has one")
	}
	return nil
}

// newAPI starts a fake API, points nabu at it with a valid token, and
// returns it with a runner that invokes nabu and fails the test on the wrong
// exit code. The runner returns stdout.
func newAPI(t *testing.T) (*fakeAPI, func(want int, stdin string, args ...string) string) {
	t.Helper()
	f := &fakeAPI{tasks: map[int]api.Task{}}
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

// content is a task body as task new and task replace take it.
func content(title string) string {
	b, _ := json.Marshal(map[string]any{"title": title})
	return string(b)
}

// body is the task content renders as task read prints it.
func body(title string) string {
	return "# " + title + "\n"
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
	if !strings.Contains(out.String(), "nabu task new [flags]") || errb.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errb.String())
	}
}

func TestNoURLExits2(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, stderr := stderrOf([]string{"task", "ls"}, "")
	if code != exitUsage || !strings.Contains(stderr, "no base_url declared") || !strings.Contains(stderr, "nabu login") {
		t.Fatalf("exit %d stderr=%q", code, stderr)
	}
}

func TestConfigURLReachesTheClient(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "nabu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nabu", "config.json"), []byte(`{"base_url": "https://nabu.example.dev/"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := client()
	if err != nil || c.URL != "https://nabu.example.dev" {
		t.Fatalf("client %+v, %v", c, err)
	}
}

func TestSignedOutExits2(t *testing.T) {
	f := &fakeAPI{tasks: map[int]api.Task{}}
	srv := httptest.NewServer(f)
	defer srv.Close()

	// Access redirects a request whose token it does not accept
	useClient(t, srv.URL, func(string) (string, error) { return "expired", nil })
	code, stderr := stderrOf([]string{"task", "ls"}, "")
	if code != exitUsage || stderr != "not signed in: run nabu login\n" {
		t.Fatalf("redirect: exit %d stderr=%q", code, stderr)
	}

	// cloudflared holds no token
	useClient(t, srv.URL, func(string) (string, error) { return "", &api.SignedOutError{} })
	if code, stderr := stderrOf([]string{"task", "read", "1"}, ""); code != exitUsage || !strings.Contains(stderr, "nabu login") {
		t.Fatalf("no token: exit %d stderr=%q", code, stderr)
	}
}

func TestTaskNewViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	ok := content("Ok")
	// the CLI refuses a body that is not a JSON object; every value it passes to the API, whose 422 it relays
	for _, args := range [][]string{
		{"task", "new", "--area", "work", "--content", body("Ok")},
		{"task", "new", "--area", "work", "--content", "[]"},
		{"task", "new", "--area", "work", "--content", `{"title": "Ok", "memo": ["` + strings.Repeat("あ", 46) + `"]}`},
	} {
		nabu(exitUsage, "", args...)
	}
	if len(f.tasks) != 0 {
		t.Fatalf("a refused task was filed: %v", f.tasks)
	}

	got := nabu(exitOK, ok, "task", "new", "--area", "work", "--ticket", "https://example.com/1", "--ticket", "https://example.com/1")
	want := `{
  "action": "task new",
  "id": 1,
  "status": "todo",
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
	// the body reaches the API as it is; what the task lacks is left out
	if f.last["title"] != "Ok" {
		t.Fatalf("sent %v", f.last)
	}
	for _, k := range []string{"project", "scheduled", "waiting", "prs", "status", "created", "why_canceled"} {
		if _, ok := f.last[k]; ok {
			t.Fatalf("new sent %s: %v", k, f.last)
		}
	}
	nabu(exitUsage, ok, "task", "new", "ok", "--area", "work")
}

func toStrings(v any) []string {
	out := []string{}
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestTaskReadPrintsMarkdown(t *testing.T) {
	_, nabu := newAPI(t)
	nabu(exitOK, content("Ok"), "task", "new", "--area", "work", "--project", "nabu")
	want := "---\ncreated: \"2026-10-06T09:00:00+09:00\"\narea: \"work\"\nproject: \"nabu\"\n---\n" + body("Ok")
	// a task is named by its id, with or without #
	nabu(exitUsage, "", "task", "read", "ok")
	for _, ref := range []string{"1", "#1"} {
		if got := nabu(exitOK, "", "task", "read", ref); got != want {
			t.Fatalf("read %s:\n%s", ref, got)
		}
	}
	nabu(exitFail, "", "task", "read", "2")
}

func TestTaskReplaceAndSetViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, content("Ok"), "task", "new", "--area", "work", "--project", "nabu", "--link", "https://example.com/a")

	if got := nabu(exitOK, `{"title": "Renamed"}`, "task", "replace", "1"); strings.Contains(got, "frontmatter") {
		t.Fatalf("replace prints frontmatter: %s", got)
	}
	if tk := f.tasks[1]; tk.Title != "Renamed" || orEmpty(tk.Project) != "nabu" {
		t.Fatalf("after replace: %+v", tk)
	}
	for _, k := range []string{"area", "project", "status"} {
		if _, ok := f.last[k]; ok {
			t.Fatalf("replace sent %s", k)
		}
	}

	nabu(exitUsage, "", "task", "set", "1")
	nabu(exitUsage, "", "task", "set", "1", "--project", "x", "--clear-project")
	got := nabu(exitOK, "", "task", "set", "1", "--clear-project", "--waiting", "review", "--link", "https://example.com/a", "--link", "https://example.com/b")
	if !strings.Contains(got, `"waiting": "review"`) || strings.Contains(got, `"project"`) {
		t.Fatalf("set: %s", got)
	}
	if v, ok := f.last["project"]; !ok || v != nil {
		t.Fatalf("set did not drop project: %v", f.last)
	}
	if tk := f.tasks[1]; !slices.Equal(tk.Links, []string{"https://example.com/a", "https://example.com/b"}) || tk.Title != "Renamed" {
		t.Fatalf("after set: %+v", tk)
	}
	nabu(exitFail, "", "task", "set", "9", "--waiting", "x")
}

func TestTaskDoneCancelReopenRmViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, content("Ok"), "task", "new", "--area", "work")

	nabu(exitFail, "", "task", "reopen", "1")
	if got := nabu(exitOK, "", "task", "done", "1"); !strings.Contains(got, `"status": "done"`) {
		t.Fatalf("done: %s", got)
	}
	nabu(exitFail, "", "task", "done", "1")

	// cancel needs a reason, and keeps it as why_canceled
	nabu(exitUsage, "", "task", "cancel", "1")
	if got := nabu(exitOK, "", "task", "cancel", "1", "今は使い道がない"); !strings.Contains(got, `"status": "canceled"`) {
		t.Fatalf("cancel: %s", got)
	}
	if tk := f.tasks[1]; tk.Status != api.Canceled || !slices.Equal(tk.WhyCanceled, []string{"今は使い道がない"}) {
		t.Fatalf("not canceled: %+v", tk)
	}
	nabu(exitFail, "", "task", "cancel", "1", "今は使い道がない")

	// reopening forgets the reason
	if got := nabu(exitOK, "", "task", "reopen", "1"); !strings.Contains(got, `"status": "todo"`) {
		t.Fatalf("reopen: %s", got)
	}
	if tk := f.tasks[1]; tk.Status != api.Todo || len(tk.WhyCanceled) != 0 {
		t.Fatalf("not reopened: %+v", tk)
	}

	if got := nabu(exitOK, "", "task", "rm", "1"); !strings.Contains(got, `"id": 1`) {
		t.Fatalf("rm: %s", got)
	}
	nabu(exitFail, "", "task", "rm", "1")
	if got := nabu(exitOK, "", "task", "schema"); !strings.Contains(got, `"create"`) {
		t.Fatalf("schema: %s", got)
	}
	if _, ok := f.tasks[1]; ok {
		t.Fatal("rm left the task")
	}
}

func TestTaskItemAddViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, content("Ok"), "task", "new", "--area", "work")
	nabu(exitUsage, "", "task", "item", "add", "1")
	nabu(exitUsage, "", "task", "item", "add", "ok", "x")
	if got := nabu(exitOK, "", "task", "item", "add", "1", "first", "second"); !strings.Contains(got, `"id": 1`) {
		t.Fatalf("item add: %s", got)
	}
	id := f.tasks[1].Id
	if len(f.items) != 2 || f.items[0] != (api.Item{Id: 1, TaskId: id, Text: "first"}) || f.items[1] != (api.Item{Id: 2, TaskId: id, Text: "second"}) {
		t.Fatalf("items: %+v", f.items)
	}
	nabu(exitUsage, "", "task", "item", "add", "1", " ")
	nabu(exitFail, "", "task", "item", "add", "9", "x")
}

func TestTaskItemCheckViaCLI(t *testing.T) {
	f, nabu := newAPI(t)
	nabu(exitOK, content("Other"), "task", "new", "--area", "work")
	nabu(exitOK, content("Ok"), "task", "new", "--area", "work")
	nabu(exitOK, "", "task", "item", "add", "1", "theirs")
	nabu(exitOK, "", "task", "item", "add", "2", "first", "second", "third")

	if got := nabu(exitOK, "", "task", "item", "check", "2", "3", "1"); !strings.Contains(got, `"third"`) || !strings.Contains(got, `"first"`) || strings.Contains(got, `"second"`) {
		t.Fatalf("item check: %s", got)
	}
	if got := nabu(exitOK, "", "task", "item", "ls", "2"); got != "1\t[x]\tfirst\n2\t[ ]\tsecond\n3\t[x]\tthird\n" {
		t.Fatalf("item ls: %q", got)
	}
	// a line is numbered within its own task
	if f.items[0].Done {
		t.Fatalf("checked a line of another task: %+v", f.items)
	}
	nabu(exitOK, "", "task", "item", "uncheck", "2", "1")
	if got := nabu(exitOK, "", "task", "item", "ls", "2"); !strings.HasPrefix(got, "1\t[ ]\tfirst\n") {
		t.Fatalf("item uncheck: %q", got)
	}

	// one bad number changes no line
	nabu(exitFail, "", "task", "item", "check", "2", "2", "4")
	nabu(exitUsage, "", "task", "item", "check", "2", "2", "second")
	if f.items[2].Done {
		t.Fatalf("checked a line beside a bad number: %+v", f.items)
	}
	nabu(exitUsage, "", "task", "item", "check", "2")
	nabu(exitFail, "", "task", "item", "check", "9", "1")
	nabu(exitFail, "", "task", "item", "ls", "9")
}

func TestTaskLsAndGrepViaCLI(t *testing.T) {
	_, nabu := newAPI(t)
	nabu(exitOK, content("B done"), "task", "new", "--area", "work")
	nabu(exitOK, content("A todo"), "task", "new", "--area", "personal")
	nabu(exitOK, "", "task", "set", "2", "--waiting", "their reply")
	nabu(exitOK, content("C canceled"), "task", "new", "--area", "work")
	nabu(exitOK, "", "task", "done", "1")
	nabu(exitOK, "", "task", "cancel", "3", "no use")

	if got := nabu(exitOK, "", "task", "ls"); got != "todo\tA todo\t2026-10-06\ndone\tB done\t2026-10-06\ncanceled\tC canceled\t2026-10-06\n" {
		t.Fatalf("ls: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "canceled"); got != "canceled\tC canceled\t2026-10-06\n" {
		t.Fatalf("ls canceled: %q", got)
	}
	nabu(exitUsage, "", "task", "ls", "doing")

	var rows []taskEntry
	if err := json.Unmarshal([]byte(nabu(exitOK, "", "task", "ls", "--json")), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].ID != 2 || rows[0].Frontmatter.Waiting != "their reply" || rows[0].Body != body("A todo") {
		t.Fatalf("ls --json: %+v", rows)
	}

	// grep searches the task as read prints it, frontmatter included
	if got := nabu(exitOK, "", "task", "grep", "a TODO"); got != "2:6: # A todo\n" {
		t.Fatalf("grep: %q", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "their reply"); got != "2:4: waiting: \"their reply\"\n" {
		t.Fatalf("grep frontmatter: %q", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "nothing-here", "--json"); strings.TrimSpace(got) != "[]" {
		t.Fatalf("grep --json: %q", got)
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

func TestRenderBodyLeavesOutAnEmptyReason(t *testing.T) {
	if got := renderBody(api.Task{Title: "T", WhyCanceled: []string{"w"}}); got != "# T\n\n## Why canceled\n\n- w\n" {
		t.Fatalf("%q", got)
	}
	if strings.Contains(renderBody(api.Task{Title: "T"}), "Why canceled") {
		t.Fatal("an empty reason is rendered")
	}
}

func TestLogin(t *testing.T) {
	useClient(t, "https://nabu.example.dev", func(string) (string, error) { return "", nil })
	orig := signIn
	t.Cleanup(func() { signIn = orig })

	var got string
	signIn = func(url string, out io.Writer) error {
		got = url
		fmt.Fprintln(out, "browser opened")
		return nil
	}
	var out, errb bytes.Buffer
	if code := run([]string{"login"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d stderr=%q", code, errb.String())
	}
	// cloudflared's own output goes to stderr, so stdout holds only the result
	if got != "https://nabu.example.dev" || errb.String() != "browser opened\n" ||
		out.String() != "{\n  \"action\": \"login\",\n  \"url\": \"https://nabu.example.dev\"\n}\n" {
		t.Fatalf("url=%q stdout=%q stderr=%q", got, out.String(), errb.String())
	}

	signIn = func(string, io.Writer) error { return errors.New("login failed") }
	if code, stderr := stderrOf([]string{"login"}, ""); code != exitFail || stderr != "login failed\n" {
		t.Fatalf("exit %d stderr=%q", code, stderr)
	}
}
