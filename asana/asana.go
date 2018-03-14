// Package asana is a client for Asana API.
package asana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "0.1"
	userAgent      = "go-asana/" + libraryVersion
	defaultBaseURL = "https://app.asana.com/api/1.0/"
)

var defaultOptFields = map[string][]string{
	"tags":       {"name", "color", "notes"},
	"users":      {"name", "email", "photo"},
	"projects":   {"name", "color", "archived"},
	"workspaces": {"name", "is_organization"},
	"tasks":      {"name", "assignee", "assignee_status", "completed", "parent"},
}

var (
	// ErrUnauthorized can be returned on any call on response status code 401.
	ErrUnauthorized = errors.New("asana: unauthorized")
)

type (
	// Doer interface used for doing http calls.
	// Use it as point of setting Auth header or custom status code error handling.
	Doer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// DoerFunc implements Doer interface.
	// Allow to transform any appropriate function "f" to Doer instance: DoerFunc(f).
	DoerFunc func(req *http.Request) (resp *http.Response, err error)

	Client struct {
		doer      Doer
		BaseURL   *url.URL
		UserAgent string
	}

	Workspace struct {
		ID           int64  `json:"id,omitempty"`
		Name         string `json:"name,omitempty"`
		Organization bool   `json:"is_organization,omitempty"`
	}

	User struct {
		ID         int64             `json:"id,omitempty"`
		Email      string            `json:"email,omitempty"`
		Name       string            `json:"name,omitempty"`
		Photo      map[string]string `json:"photo,omitempty"`
		Workspaces []Workspace       `json:"workspaces,omitempty"`
	}

	Project struct {
		ID       int64  `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Archived bool   `json:"archived,omitempty"`
		Color    string `json:"color,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	Task struct {
		ID             int64     `json:"id,omitempty"`
		Assignee       *User     `json:"assignee,omitempty"`
		AssigneeStatus string    `json:"assignee_status,omitempty"`
		CreatedAt      time.Time `json:"created_at,omitempty"`
		CreatedBy      User      `json:"created_by,omitempty"` // Undocumented field, but it can be included.
		Completed      bool      `json:"completed,omitempty"`
		CompletedAt    time.Time `json:"completed_at,omitempty"`
		Name           string    `json:"name,omitempty"`
		Hearts         []Heart   `json:"hearts,omitempty"`
		Notes          string    `json:"notes,omitempty"`
		ParentTask     *Task     `json:"parent,omitempty"`
		Projects       []Project `json:"projects,omitempty"`
		DueOn          string    `json:"due_on,omitempty"`
		DueAt          string    `json:"due_at,omitempty"`
		Followers      []User    `json:"followers,omitempty"`
		Liked          bool      `json:"liked,omitempty"`
		NumHearts      int64     `json:"num_hearts,omitempty"`
		Hearted        bool      `json:"hearted,omitempty"`
		ModifiedAt     time.Time `json:"modified_at,omitempty"`
		NumLikes       int64     `json:"num_likes,omitempty"`
		Tags           []Tag     `json:"tags,omitempty"`
		// "workspace":    map[string]interface {}{"id":13218399566047.000000,"name":"wacul.co.jp"},
		External External `json:"external,omitempty"`
	}
	External struct {
		ID   string      `json:"id,omitempty"`
		Data interface{} `json:"data,omitempty"`
	}
	// TaskUpdate is used to update a task.
	TaskUpdate struct {
		Assignee    *string    `json:"assignee,omitempty"`
		Name        *string    `json:"name,omitempty"`
		Notes       *string    `json:"notes,omitempty"`
		Hearted     *bool      `json:"hearted,omitempty"`
		Completed   *bool      `json:"completed,omitempty"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	}

	Story struct {
		ID        int64     `json:"id,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		CreatedBy User      `json:"created_by,omitempty"`
		Hearts    []Heart   `json:"hearts,omitempty"`
		Text      string    `json:"text,omitempty"`
		Type      string    `json:"type,omitempty"` // E.g., "comment", "system".
	}

	// Heart represents a ♥ action by a user.
	Heart struct {
		ID   int64 `json:"id,omitempty"`
		User User  `json:"user,omitempty"`
	}

	Tag struct {
		ID    int64  `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color,omitempty"`
		Notes string `json:"notes,omitempty"`
	}

	Filter struct {
		Archived       bool     `url:"archived,omitempty"`
		Assignee       int64    `url:"assignee,omitempty"`
		Project        int64    `url:"project,omitempty"`
		Workspace      int64    `url:"workspace,omitempty"`
		CompletedSince string   `url:"completed_since,omitempty"`
		ModifiedSince  string   `url:"modified_since,omitempty"`
		OptFields      []string `url:"opt_fields,comma,omitempty"`
		OptExpand      []string `url:"opt_expand,comma,omitempty"`
		Offset         string   `url:"offset,omitempty"`
		Limit          uint32   `url:"limit,omitempty"`
	}

	request struct {
		Data interface{} `json:"data,omitempty"`
	}

	Response struct {
		Data     interface{} `json:"data,omitempty"`
		NextPage *NextPage   `json:"next_page,omitempty"`
		Errors   []Error     `json:"errors,omitempty"`
	}

	Error struct {
		Phrase  string `json:"phrase,omitempty"`
		Message string `json:"message,omitempty"`
	}

	Webhook struct {
		ID       int64    `json:"id,omitempty"`
		Resource Resource `json:"resource,omitempty"`
		Target   string   `json:"target,omitempty"`
		Active   bool     `json:"active,omitempty"`
	}

	Resource struct {
		ID   int64  `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	}

	NextPage struct {
		Offset string `json:"offset,omitempty"`
		Path   string `json:"path,omitempty"`
		URI    string `json:"uri,omitempty"`
	}

	// Errors always has at least 1 element when returned.
	Errors struct {
		Errors []Error
		Code   int
	}

	EventSummary struct {
		// User who triggered the event.
		// Read-only.
		// NOTE: The event may be triggered by a different user than the subscriber.
		// For example, if user A subscribes to a task and user B modified it, the event’s user will be user B.
		// NOTE: Some events are generated by the system, and will have null as the user.
		// API consumers should make sure to handle this case.
		UserID int64 `json:"user,omitempty"`
		// Resource the event occurred on.
		// Read-only.
		// NOTE: The resource that triggered the event may be different from the one that the events were requested for.
		// For example, a subscription to a project will contain events for tasks contained within the project.
		ResourceID int64 `json:"resource,omitempty"`
		// Type of the resource that generated the event.
		// Read-only.
		// NOTE: Currently, only tasks, projects and stories generate events.
		Type string `json:"type,omitempty"`
		// Action taken that triggered the event.
		// Read-only.
		Action string `json:"action,omitempty"`
		// Parent that resource was added to or removed from. null for other event types.
		// Read-only.
		ParentID *int64 `json:"parent,omitempty"`
		// Timestamp when the event occurred.
		// Read-only.
		CreatedAt time.Time `json:"created_at,omitempty"`
	}

	Event struct {
		// User who triggered the event.
		// Read-only.
		// NOTE: The event may be triggered by a different user than the subscriber.
		// For example, if user A subscribes to a task and user B modified it, the event’s user will be user B.
		// NOTE: Some events are generated by the system, and will have null as the user.
		// API consumers should make sure to handle this case.
		User User `json:"user,omitempty"`
		// Resource the event occurred on.
		// Read-only.
		// NOTE: The resource that triggered the event may be different from the one that the events were requested for.
		// For example, a subscription to a project will contain events for tasks contained within the project.
		Resource Resource `json:"resource,omitempty"`
		// Type of the resource that generated the event.
		// Read-only.
		// NOTE: Currently, only tasks, projects and stories generate events.
		Type string `json:"type,omitempty"`
		// Action taken that triggered the event.
		// Read-only.
		Action string `json:"action,omitempty"`
		// Parent that resource was added to or removed from. null for other event types.
		// Read-only.
		Parent Resource `json:"parent,omitempty"`
		// Timestamp when the event occurred.
		// Read-only.
		CreatedAt time.Time `json:"created_at,omitempty"`
	}
)

func (f DoerFunc) Do(req *http.Request) (resp *http.Response, err error) {
	return f(req)
}

func (e Error) Error() string {
	return fmt.Sprintf("%v - %v", e.Message, e.Phrase)
}

func (e *Errors) Error() string {
	var sErrs []string
	for _, err := range e.Errors {
		sErrs = append(sErrs, err.Error())
	}
	return fmt.Sprintf("code: %d, %s", e.Code, strings.Join(sErrs, ", "))
}

// NewClient created new asana client with doer.
// If doer is nil then http.DefaultClient used intead.
func NewClient(doer Doer) *Client {
	if doer == nil {
		doer = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	client := &Client{doer: doer, BaseURL: baseURL, UserAgent: userAgent}
	return client
}

func (c *Client) ListWorkspaces(ctx context.Context, opt *Filter) ([]Workspace, error) {
	workspaces := []Workspace{}
	for {
		page := []Workspace{}
		next, err := c.request(ctx, "GET", "workspaces", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return workspaces, nil
}

func (c *Client) ListUsers(ctx context.Context, opt *Filter) ([]User, error) {
	users := []User{}
	for {
		page := []User{}
		next, err := c.request(ctx, "GET", "users", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		users = append(users, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return users, nil
}

func (c *Client) ListProjects(ctx context.Context, opt *Filter) ([]Project, error) {
	projects := []Project{}
	for {
		page := []Project{}
		next, err := c.request(ctx, "GET", "projects", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		projects = append(projects, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return projects, nil
}

func (c *Client) ListTasks(ctx context.Context, opt *Filter) ([]Task, error) {
	tasks := []Task{}
	for {
		page := []Task{}
		next, err := c.request(ctx, "GET", "tasks", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return tasks, nil
}

func externalQuery(externalID string) string {
	return fmt.Sprintf("tasks/external:%s", externalID)
}

func (c *Client) GetTaskByExternalID(ctx context.Context, externalID string, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(ctx, externalQuery(externalID), opt, task)
	return *task, err
}

func (c *Client) GetTask(ctx context.Context, id int64, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(ctx, fmt.Sprintf("tasks/%d", id), opt, task)
	return *task, err
}

// DeleteTaskByExternalID deletes a task.
//
// https://asana.com/developers/api-reference/tasks#delete
func (c *Client) DeleteTaskByExternalID(ctx context.Context, externalID string, opt *Filter) error {
	_, err := c.request(ctx, "DELETE", externalQuery(externalID), nil, nil, opt, nil)
	return err
}

// DeleteTask deletes a task.
//
// https://asana.com/developers/api-reference/tasks#delete
func (c *Client) DeleteTask(ctx context.Context, id int64, opt *Filter) error {
	_, err := c.request(ctx, "DELETE", fmt.Sprintf("tasks/%d", id), nil, nil, opt, nil)
	return err
}

// UpdateTaskByExternalID updates a task.
//
// https://asana.com/developers/api-reference/tasks#update
func (c *Client) UpdateTaskByExternalID(ctx context.Context, externalID string, tu TaskUpdate, opt *Filter) (Task, error) {
	task := new(Task)
	_, err := c.request(ctx, "PUT", externalQuery(externalID), tu, nil, opt, task)
	return *task, err
}

// UpdateTask updates a task.
//
// https://asana.com/developers/api-reference/tasks#update
func (c *Client) UpdateTask(ctx context.Context, id int64, tu TaskUpdate, opt *Filter) (Task, error) {
	task := new(Task)
	_, err := c.request(ctx, "PUT", fmt.Sprintf("tasks/%d", id), tu, nil, opt, task)
	return *task, err
}

// CreateTask creates a task.
//
// https://asana.com/developers/api-reference/tasks#create
func (c *Client) CreateTask(ctx context.Context, fields map[string]interface{}, opts *Filter) (Task, error) {
	task := new(Task)
	_, err := c.request(ctx, "POST", "tasks", fields, nil, opts, task)
	return *task, err
}

// AddTagByExternalID adds a tag to a task.
//
// https://asana.com/developers/api-reference/tasks#tags
func (c *Client) AddTagByExternalID(ctx context.Context, externalID string, tagID int64, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/external:%s/addTag", externalID), map[string]interface{}{"tag": tagID}, nil, opts, nil)
	return err
}

// RemoveTagByExternalID removes a tag to a task.
//
// https://asana.com/developers/api-reference/tasks#tags
func (c *Client) RemoveTagByExternalID(ctx context.Context, externalID string, tagID int64, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/external:%s/removeTag", externalID), map[string]interface{}{"tag": tagID}, nil, opts, nil)
	return err
}

// AddTag adds a tag to a task.
//
// https://asana.com/developers/api-reference/tasks#tags
func (c *Client) AddTag(ctx context.Context, taskID int64, tagID int64, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/%d/addTag", taskID), map[string]interface{}{"tag": tagID}, nil, opts, nil)
	return err
}

// RemoveTag removes a tag to a task.
//
// https://asana.com/developers/api-reference/tasks#tags
func (c *Client) RemoveTag(ctx context.Context, taskID int64, tagID int64, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/%d/removeTag", taskID), map[string]interface{}{"tag": tagID}, nil, opts, nil)
	return err
}

func (c *Client) ListProjectTasks(ctx context.Context, projectID int64, opt *Filter) ([]Task, error) {
	tasks := []Task{}
	for {
		page := []Task{}
		next, err := c.request(ctx, "GET", fmt.Sprintf("projects/%d/tasks", projectID), nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return tasks, nil
}

func (c *Client) ListTaskStories(ctx context.Context, taskID int64, opt *Filter) ([]Story, error) {
	stories := []Story{}
	for {
		page := []Story{}
		next, err := c.request(ctx, "GET", fmt.Sprintf("tasks/%d/stories", taskID), nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		stories = append(stories, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return stories, nil
}

func (c *Client) ListTags(ctx context.Context, opt *Filter) ([]Tag, error) {
	tags := []Tag{}
	for {
		page := []Tag{}
		next, err := c.request(ctx, "GET", "tags", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		tags = append(tags, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return tags, nil
}

func (c *Client) GetAuthenticatedUser(ctx context.Context, opt *Filter) (User, error) {
	user := new(User)
	err := c.Request(ctx, "users/me", opt, user)
	return *user, err
}

func (c *Client) GetUserByID(ctx context.Context, id int64, opt *Filter) (User, error) {
	user := new(User)
	err := c.Request(ctx, fmt.Sprintf("users/%d", id), opt, user)
	return *user, err
}

func (c *Client) GetWebhooks(ctx context.Context, opt *Filter) ([]Webhook, error) {
	webhooks := []Webhook{}
	for {
		page := []Webhook{}
		next, err := c.request(ctx, "GET", "webhooks", nil, nil, opt, &page)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, page...)
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return webhooks, nil
}

func (c *Client) GetWebhook(ctx context.Context, id int64) (Webhook, error) {
	webhook := new(Webhook)
	err := c.Request(ctx, fmt.Sprintf("webhooks/%d", id), nil, &webhook)
	return *webhook, err
}

func (c *Client) CreateWebhook(ctx context.Context, id int64, target string) (Webhook, error) {
	webhook := new(Webhook)
	p := url.Values{
		"resource": []string{fmt.Sprintf("%d", id)},
		"target":   []string{target},
	}
	_, err := c.request(ctx, "POST", "webhooks", nil, p, nil, &webhook)
	return *webhook, err
}

func (c *Client) DeleteWebhook(ctx context.Context, id int64) error {
	var resp interface{} // Empty response
	_, err := c.request(ctx, "DELETE", fmt.Sprintf("webhooks/%d", id), nil, nil, nil, &resp)
	return err
}

func (c *Client) Request(ctx context.Context, path string, opt *Filter, v interface{}) error {
	_, err := c.request(ctx, "GET", path, nil, nil, opt, v)
	return err
}

// request makes a request to Asana API, using method, at path, sending data or form with opt filter.
// Only data or form could be sent at the same time. If both provided form will be omitted.
// Also it's possible to do request with nil data and form.
// The response is populated into v, and any error is returned.
func (c *Client) request(ctx context.Context, method string, path string, data interface{}, form url.Values, opt *Filter, v interface{}) (*NextPage, error) {
	if opt == nil {
		opt = &Filter{}
	}
	if len(opt.OptFields) == 0 {
		// We should not modify opt provided to Request.
		newOpt := *opt
		opt = &newOpt
		opt.OptFields = defaultOptFields[path]
	}
	urlStr, err := addOptions(path, opt)
	if err != nil {
		return nil, err
	}
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(rel)
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(request{Data: data})
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	} else if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	} else if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.doer.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	res := &Response{Data: v}
	err = json.NewDecoder(resp.Body).Decode(res)
	if len(res.Errors) > 0 {
		return nil, &Errors{Errors: res.Errors, Code: resp.StatusCode}
	}
	return res.NextPage, err
}

func addOptions(s string, opt interface{}) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}
	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func toURLValues(m map[string]string) url.Values {
	values := make(url.Values)
	for k, v := range m {
		values[k] = []string{v}
	}
	return values
}
