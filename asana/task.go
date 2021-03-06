package asana

import (
	"context"
	"fmt"
)

// ListTasks gets tasks.
//
// https://asana.com/developers/api-reference/tasks#query
func (c *Client) ListTasks(ctx context.Context, opt *Filter) ([]Task, error) {
	rets := []Task{}
	if err := c.pagenate(ctx, "tasks", opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}

func externalTaskQuery(externalID string) string {
	return fmt.Sprintf("tasks/external:%s", externalID)
}

// GetTaskByExternalID gets a task with an external-ID.
//
// https://asana.com/developers/api-reference/tasks#get
func (c *Client) GetTaskByExternalID(ctx context.Context, externalID string, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(ctx, externalTaskQuery(externalID), opt, task)
	return *task, err
}

// GetTask gets a task.
//
// https://asana.com/developers/api-reference/tasks#get
func (c *Client) GetTask(ctx context.Context, id int64, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(ctx, fmt.Sprintf("tasks/%d", id), opt, task)
	return *task, err
}

// DeleteTaskByExternalID deletes a task.
//
// https://asana.com/developers/api-reference/tasks#delete
func (c *Client) DeleteTaskByExternalID(ctx context.Context, externalID string, opt *Filter) error {
	_, err := c.request(ctx, "DELETE", externalTaskQuery(externalID), nil, nil, opt, nil)
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
	_, err := c.request(ctx, "PUT", externalTaskQuery(externalID), tu, nil, opt, task)
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

// ListProjectTasks gets tasks in the project.
//
// https://asana.com/developers/api-reference/tasks#query
func (c *Client) ListProjectTasks(ctx context.Context, projectID int64, opt *Filter) ([]Task, error) {
	rets := []Task{}
	if err := c.pagenate(ctx, fmt.Sprintf("projects/%d/tasks", projectID), opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
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

// AddProjectByExternalID adds a project to a task.
//
// https://asana.com/developers/api-reference/tasks#projects
func (c *Client) AddProjectByExternalID(ctx context.Context, externalID string, mu MembershipUpdate, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/external:%s/addProject", externalID), mu, nil, opts, nil)
	return err
}

// RemoveProjectByExternalID removes a project to a task.
//
// https://asana.com/developers/api-reference/tasks#projects
func (c *Client) RemoveProjectByExternalID(ctx context.Context, externalID string, mu MembershipUpdate, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/external:%s/removeProject", externalID), mu, nil, opts, nil)
	return err
}

// AddProject adds a project to a task.
//
// https://asana.com/developers/api-reference/tasks#projects
func (c *Client) AddProject(ctx context.Context, taskID int64, mu MembershipUpdate, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/%d/addProject", taskID), mu, nil, opts, nil)
	return err
}

// RemoveProject removes a project to a task.
//
// https://asana.com/developers/api-reference/tasks#projects
func (c *Client) RemoveProject(ctx context.Context, taskID int64, mu MembershipUpdate, opts *Filter) error {
	_, err := c.request(ctx, "POST", fmt.Sprintf("tasks/%d/removeProject", taskID), mu, nil, opts, nil)
	return err
}

// GetCustomFieldValue Get a custom_field value from a task
func (t *Task) GetCustomFieldValue(name string) (string, error) {
	for _, cf := range t.CustomFields {
		if cf.Name == name {
			switch cf.Type {
			case "enum":
				return cf.EnumValue.Name, nil
			case "text":
				return cf.TextValue, nil
			case "number":
				return string(cf.NumberValue), nil
			}
		}
	}
	return "", fmt.Errorf("Custom field '%s' not found", name)
}
