// Package api is the client of the nabu Worker's /api/tasks. Every request
// carries the Access token cloudflared holds for the Worker, so nabu itself
// keeps no secret.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Statuses are the task statuses in workflow order. done is finished work;
// dropped is work decided against, and the task says why.
var Statuses = []string{"inbox", "doing", "done", "dropped"}

// Task is a task as the API returns it. An optional value the API sends as
// null reads as "".
type Task struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Area         string   `json:"area"`
	Project      string   `json:"project"`
	Scheduled    string   `json:"scheduled"`
	Waiting      string   `json:"waiting"`
	Tickets      []string `json:"tickets"`
	Prs          []string `json:"prs"`
	Links        []string `json:"links"`
	Situation    []string `json:"situation"`
	Complication []string `json:"complication"`
	Question     []string `json:"question"`
	Answer       []string `json:"answer"`
	WhyDropped   []string `json:"why_dropped"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
}

// Error is a request the API refused: Status is the HTTP status and Message
// the API's reason.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

// SignedOutError is returned when Access does not let the request through:
// cloudflared holds no token for the Worker, or the one it holds expired.
type SignedOutError struct{ URL string }

func (e *SignedOutError) Error() string {
	return "not signed in: run cloudflared access login " + e.URL
}

// Client talks to one Worker.
type Client struct {
	URL string
	// Token returns the Access token for URL.
	Token func(url string) (string, error)
	HTTP  *http.Client
}

// New returns a client for the Worker at url, signed in through cloudflared.
func New(url string) *Client {
	return &Client{
		URL:   url,
		Token: cloudflared,
		// Access answers a request it does not let through with a redirect to
		// its login page; it must be seen, not followed.
		HTTP: &http.Client{
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// cloudflared prints the token it holds for app; it fails when it holds none.
func cloudflared(app string) (string, error) {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return "", errors.New("cloudflared not found: install it (brew install cloudflared), then run cloudflared access login " + app)
	}
	out, err := exec.Command("cloudflared", "access", "token", "-app="+app).Output()
	token := strings.TrimSpace(string(out))
	if err != nil || token == "" || strings.ContainsAny(token, " \n") {
		return "", &SignedOutError{URL: app}
	}
	return token, nil
}

// List returns every task, or those in status and those containing q
// (either may be ""), ordered by slug.
func (c *Client) List(status, q string) ([]Task, error) {
	v := url.Values{}
	if status != "" {
		v.Set("status", status)
	}
	if q != "" {
		v.Set("q", q)
	}
	path := "/api/tasks"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var out []Task
	return out, c.do(http.MethodGet, path, nil, &out)
}

// Get returns one task.
func (c *Client) Get(slug string) (Task, error) {
	var out Task
	return out, c.do(http.MethodGet, taskPath(slug), nil, &out)
}

// Create files a new task in inbox. fields are the API's names and values.
func (c *Client) Create(fields map[string]any) (Task, error) {
	var out Task
	return out, c.do(http.MethodPost, "/api/tasks", fields, &out)
}

// Patch changes the fields given; a nil value drops an optional one.
func (c *Client) Patch(slug string, fields map[string]any) (Task, error) {
	var out Task
	return out, c.do(http.MethodPatch, taskPath(slug), fields, &out)
}

// Delete removes a task; the API keeps its history.
func (c *Client) Delete(slug string) error {
	return c.do(http.MethodDelete, taskPath(slug), nil, nil)
}

func taskPath(slug string) string { return "/api/tasks/" + url.PathEscape(slug) }

func (c *Client) do(method, path string, in, out any) error {
	token, err := c.Token(c.URL)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.URL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("cf-access-token", token)
	if in != nil {
		req.Header.Set("content-type", "application/json")
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	switch {
	case res.StatusCode >= 300 && res.StatusCode < 400, res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden:
		return &SignedOutError{URL: c.URL}
	case res.StatusCode >= 400:
		var e struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(res.Body).Decode(&e) != nil || e.Error == "" {
			e.Error = res.Status
		}
		return &Error{Status: res.StatusCode, Message: e.Error}
	case out == nil:
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: unreadable response: %w", method, path, err)
	}
	return nil
}
