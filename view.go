package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// md renders task bodies; GFM so tables and task lists come through.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

//go:embed web/index.html.tmpl web/app.css web/app.js
var web embed.FS

// viewTask is one task as the page's script sees it: the ls row with the
// body rendered to HTML, plus a lowercase blob the search box matches on.
type viewTask struct {
	Path        string   `json:"path"`
	Slug        string   `json:"slug"`
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Frontmatter taskMeta `json:"frontmatter"`
	HTML        string   `json:"html"`
	Text        string   `json:"text"`
}

func runTaskView(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task view", "nabu task view [--out <dir>] [--no-open]\n\nrenders every task into one self-contained <dir>/index.html and opens it in the browser.\n<dir> defaults to the user cache directory (nabu/view); --no-open only writes the file")
	root := bindRoot(fs)
	out := fs.String("out", "", "directory to write index.html into")
	noOpen := fs.Bool("no-open", false, "write the file without opening it")
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rows, err := listTasks(s, "")
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	page, err := renderView(rows)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	dir := *out
	if dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return fail(stderr, err, exitFail)
		}
		dir = filepath.Join(cache, "nabu", "view")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(stderr, err, exitFail)
	}
	file := filepath.Join(dir, "index.html")
	if err := os.WriteFile(file, page, 0o644); err != nil {
		return fail(stderr, err, exitFail)
	}
	fmt.Fprintln(stdout, file)
	if *noOpen {
		return exitOK
	}
	if err := openBrowser(file); err != nil {
		return fail(stderr, err, exitFail)
	}
	return exitOK
}

// renderView fills the embedded page with the tasks: doing first, then
// inbox, then done; inside a status, scheduled tasks by time, then the rest
// by slug.
func renderView(rows []taskEntry) ([]byte, error) {
	order := map[string]int{"doing": 0, "inbox": 1, "done": 2, "stray": 3}
	tasks := make([]viewTask, 0, len(rows))
	for _, r := range rows {
		body := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Body), "# "+r.Title))
		var html bytes.Buffer
		if err := md.Convert([]byte(body), &html); err != nil {
			return nil, fmt.Errorf("%s: %w", r.Path, err)
		}
		tasks = append(tasks, viewTask{
			Path: r.Path, Slug: r.Slug, Status: r.Status, Title: r.Title, Frontmatter: r.Frontmatter,
			HTML: html.String(),
			Text: strings.ToLower(strings.Join([]string{r.Title, r.Slug, r.Frontmatter.Waiting, body}, "\n")),
		})
	}
	key := func(t viewTask) string {
		if t.Frontmatter.Scheduled != "" {
			return "0" + t.Frontmatter.Scheduled
		}
		return "1" + t.Slug
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if order[tasks[i].Status] != order[tasks[j].Status] {
			return order[tasks[i].Status] < order[tasks[j].Status]
		}
		return key(tasks[i]) < key(tasks[j])
	})
	var data bytes.Buffer
	enc := json.NewEncoder(&data) // escapes < as <, so the JSON is safe inside <script>
	if err := enc.Encode(tasks); err != nil {
		return nil, err
	}
	read := func(name string) string {
		b, err := web.ReadFile("web/" + name)
		if err != nil {
			panic(err) // embedded; cannot fail
		}
		return string(b)
	}
	tmpl, err := template.New("page").Parse(read("index.html.tmpl"))
	if err != nil {
		return nil, err
	}
	loc, _ := time.LoadLocation("Asia/Tokyo")
	var page bytes.Buffer
	err = tmpl.Execute(&page, map[string]string{
		"CSS":   read("app.css"),
		"JS":    read("app.js"),
		"Data":  strings.TrimSpace(data.String()),
		"Built": time.Now().In(loc).Format("2006/01/02 15:04"),
	})
	return page.Bytes(), err
}

func openBrowser(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
