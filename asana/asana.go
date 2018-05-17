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
	"reflect"
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
		ID             int64         `json:"id,omitempty"`
		Assignee       *User         `json:"assignee,omitempty"`
		AssigneeStatus string        `json:"assignee_status,omitempty"`
		CreatedAt      time.Time     `json:"created_at,omitempty"`
		CreatedBy      User          `json:"created_by,omitempty"` // Undocumented field, but it can be included.
		Completed      bool          `json:"completed,omitempty"`
		CompletedAt    time.Time     `json:"completed_at,omitempty"`
		CustomFields   []CustomField `json:"custom_fields,omitempty"`
		Name           string        `json:"name,omitempty"`
		Hearts         []Heart       `json:"hearts,omitempty"`
		Notes          string        `json:"notes,omitempty"`
		ParentTask     *Task         `json:"parent,omitempty"`
		Projects       []Project     `json:"projects,omitempty"`
		DueOn          string        `json:"due_on,omitempty"`
		DueAt          string        `json:"due_at,omitempty"`
		Followers      []User        `json:"followers,omitempty"`
		Liked          bool          `json:"liked,omitempty"`
		NumHearts      int64         `json:"num_hearts,omitempty"`
		Hearted        bool          `json:"hearted,omitempty"`
		ModifiedAt     time.Time     `json:"modified_at,omitempty"`
		NumLikes       int64         `json:"num_likes,omitempty"`
		Tags           []Tag         `json:"tags,omitempty"`
		Memberships    []Membership  `json:"memberships,omitempty"`
		// "workspace":    map[string]interface {}{"id":13218399566047.000000,"name":"wacul.co.jp"},
		External External `json:"external,omitempty"`
	}
	External struct {
		ID   string      `json:"id,omitempty"`
		Data interface{} `json:"data,omitempty"`
	}
	Membership struct {
		Project Project `json:"project,omitempty"`
		Section Section `json:"section,omitempty"`
	}

	// TaskUpdate is used to update a task.
	TaskUpdate struct {
		Assignee     *string               `json:"assignee,omitempty"`
		Name         *string               `json:"name,omitempty"`
		Notes        *string               `json:"notes,omitempty"`
		Hearted      *bool                 `json:"hearted,omitempty"`
		Completed    *bool                 `json:"completed,omitempty"`
		CompletedAt  *time.Time            `json:"completed_at,omitempty"`
		CustomFields map[int64]interface{} `json:"custom_fields,omitempty"`
	}

	MembershipUpdate struct {
		ProjectID    int64  `json:"project,omitempty"`
		InsertAfter  *int64 `json:"insert_after,omitempty"`
		InsertBefore *int64 `json:"insert_before,omitempty"`
		Section      *int64 `json:"section,omitempty"`
	}
	Section struct {
		ID        int64     `json:"id,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		Name      string    `json:"name,omitempty"`
		Project   Project   `json:"project,omitempty"`
		Tags      []Tag     `json:"tags,omitempty"`
		External  External  `json:"external,omitempty"`
	}

	SectionUpdate struct {
		Name *string `json:"name,omitempty"`
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

	CustomField struct {
		ID          int64           `json:"id,omitempty"`
		Name        string          `json:"name,omitempty"`
		Description string          `json:"description,omitempty"`
		Type        string          `json:"type,omitempty"`
		EnumOptions []CFEnumOptions `json:"enum_options,omitempty"`
		Precision   int64           `json:"precision,omitempty"`
		TextValue   string          `json:"text_value,omitempty"`
		NumberValue int64           `json:"number_value,omitempty"`
		EnumValue   CFEnumOptions   `json:"enum_value,omitempty"`
	}

	CFEnumOptions struct {
		ID      int64  `json:"id,omitempty"`
		Name    string `json:"name,omitempty"`
		Color   string `json:"color,omitempty"`
		Enabled bool   `json:"enabled,omitempty"`
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

func remake(v interface{}) (interface{}, error) {
	return remakeValue(reflect.ValueOf(v))
}

func remakeValue(v reflect.Value) (interface{}, error) {
	if v.Kind() != reflect.Ptr {
		return nil, errors.New("not pointer")
	}
	return reflect.New(v.Type().Elem()).Interface(), nil
}

func ve(a interface{}) reflect.Value {
	return reflect.ValueOf(a).Elem()
}

func appendSliceValue(a1, a2 interface{}) {
	na := reflect.AppendSlice(ve(a1), ve(a2))
	ve(a1).Set(na)
}

func (c *Client) pagenate(ctx context.Context, path string, opt *Filter, v interface{}) error {
	for {
		page, err := remake(v)
		if err != nil {
			return err
		}
		next, err := c.request(ctx, "GET", path, nil, nil, opt, page)
		if err != nil {
			return err
		}
		reflect.ValueOf(v).Elem().Set(reflect.AppendSlice(reflect.ValueOf(v).Elem(), reflect.ValueOf(page).Elem()))
		if next == nil {
			break
		} else {
			newOpt := *opt
			opt = &newOpt
			opt.Offset = next.Offset
		}
	}
	return nil
}

func (c *Client) ListWorkspaces(ctx context.Context, opt *Filter) ([]Workspace, error) {
	rets := []Workspace{}
	if err := c.pagenate(ctx, "workspaces", opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}

func (c *Client) ListUsers(ctx context.Context, opt *Filter) ([]User, error) {
	rets := []User{}
	if err := c.pagenate(ctx, "users", opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}

func (c *Client) ListProjects(ctx context.Context, opt *Filter) ([]Project, error) {
	rets := []Project{}
	if err := c.pagenate(ctx, "projects", opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}

func (c *Client) ListTaskStories(ctx context.Context, taskID int64, opt *Filter) ([]Story, error) {
	rets := []Story{}
	if err := c.pagenate(ctx, fmt.Sprintf("tasks/%d/stories", taskID), opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}

func (c *Client) ListTags(ctx context.Context, opt *Filter) ([]Tag, error) {
	rets := []Tag{}
	if err := c.pagenate(ctx, "tags", opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
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
