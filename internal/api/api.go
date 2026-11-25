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
	"os/exec"
	"strings"
	"time"
)

// Task, Status, Area, and Habit are generated from openapi.json, the document
// the API serves at /api/openapi.json, so the CLI keeps no copy of its own.
//go:generate go tool oapi-codegen -generate types -package api -o openapi.gen.go ../../openapi.json

// Error is a request the API refused: Status is the HTTP status and Message
// the API's reason.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

// SignedOutError is returned when Access does not let the request through:
// cloudflared holds no token for the Worker, or the one it holds expired.
type SignedOutError struct{}

func (e *SignedOutError) Error() string { return "not signed in: run nabu login" }

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

var errNoCloudflared = errors.New("cloudflared not found: install it (brew install cloudflared), then run nabu login")

// Login signs in to the Worker at url through cloudflared, which opens the
// browser and keeps the token. -q keeps the token off the terminal, where an
// agent session would record it; what cloudflared prints goes to out.
func Login(url string, out io.Writer) error {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return errNoCloudflared
	}
	cmd := exec.Command("cloudflared", "access", "login", "-q", url)
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}

// cloudflared prints the token it holds for app; it fails when it holds none.
func cloudflared(app string) (string, error) {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return "", errNoCloudflared
	}
	out, err := exec.Command("cloudflared", "access", "token", "-app="+app).Output()
	token := strings.TrimSpace(string(out))
	if err != nil || token == "" || strings.ContainsAny(token, " \n") {
		return "", &SignedOutError{}
	}
	return token, nil
}

// List returns every task in workflow order, then by id; callers narrow
// them themselves.
func (c *Client) List() ([]Task, error) {
	var out []Task
	return out, c.do(http.MethodGet, "/api/tasks", nil, &out)
}

// Get returns the task with id.
func (c *Client) Get(id int) (Task, error) {
	var out Task
	return out, c.do(http.MethodGet, taskPath(id), nil, &out)
}

// Create files a new todo task. fields are the API's names and values.
func (c *Client) Create(fields map[string]any) (Task, error) {
	var out Task
	return out, c.do(http.MethodPost, "/api/tasks", fields, &out)
}

// Patch changes the fields given; a nil value drops an optional one.
func (c *Client) Patch(id int, fields map[string]any) (Task, error) {
	var out Task
	return out, c.do(http.MethodPatch, taskPath(id), fields, &out)
}

// Delete removes a task; the API keeps its history.
func (c *Client) Delete(id int) error {
	return c.do(http.MethodDelete, taskPath(id), nil, nil)
}

// AddItem adds a line, not done, last in the checklist of the task with id taskID.
func (c *Client) AddItem(taskID int, text string) (Item, error) {
	var out Item
	return out, c.do(http.MethodPost, "/api/items", PostApiItemsJSONBody{TaskId: taskID, Text: text}, &out)
}

// Items returns the checklist of the task with id taskID, in order.
func (c *Client) Items(taskID int) ([]Item, error) {
	var out []Item
	return out, c.do(http.MethodGet, fmt.Sprintf("/api/items?task_id=%d", taskID), nil, &out)
}

// CheckItem marks the checklist line with id done, or not done.
func (c *Client) CheckItem(id int, done bool) (Item, error) {
	var out Item
	return out, c.do(http.MethodPatch, fmt.Sprintf("/api/items/%d", id), PatchApiItemsIdJSONBody{Done: &done}, &out)
}

// Schema returns the task contract as JSON Schema, the rules every write must meet.
func (c *Client) Schema() (json.RawMessage, error) {
	var out json.RawMessage
	return out, c.do(http.MethodGet, "/api/schema", nil, &out)
}

// taskPath is the API's path to the task with id.
func taskPath(id int) string { return fmt.Sprintf("/api/tasks/%d", id) }

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
		return &SignedOutError{}
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
