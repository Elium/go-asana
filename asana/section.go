package asana

import (
	"context"
	"fmt"
)

func externalSectionQuery(externalID string) string {
	return fmt.Sprintf("sections/external:%s", externalID)
}

// GetSectionByExternalID gets a section with an external-ID.
//
// https://asana.com/developers/api-reference/sections#get-single
func (c *Client) GetSectionByExternalID(ctx context.Context, externalID string, opt *Filter) (Section, error) {
	section := new(Section)
	err := c.Request(ctx, externalSectionQuery(externalID), opt, section)
	return *section, err
}

// GetSection gets a section.
//
// https://asana.com/developers/api-reference/sections#get-single
func (c *Client) GetSection(ctx context.Context, id int64, opt *Filter) (Section, error) {
	section := new(Section)
	err := c.Request(ctx, fmt.Sprintf("sections/%d", id), opt, section)
	return *section, err
}

// DeleteSectionByExternalID deletes a section.
//
// https://asana.com/developers/api-reference/sections#delete
func (c *Client) DeleteSectionByExternalID(ctx context.Context, externalID string, opt *Filter) error {
	_, err := c.request(ctx, "DELETE", externalSectionQuery(externalID), nil, nil, opt, nil)
	return err
}

// DeleteSection deletes a section.
//
// https://asana.com/developers/api-reference/sections#delete
func (c *Client) DeleteSection(ctx context.Context, id int64, opt *Filter) error {
	_, err := c.request(ctx, "DELETE", fmt.Sprintf("sections/%d", id), nil, nil, opt, nil)
	return err
}

// UpdateSectionByExternalID updates a section.
//
// https://asana.com/developers/api-reference/sections#update
func (c *Client) UpdateSectionByExternalID(ctx context.Context, externalID string, su SectionUpdate, opt *Filter) (Section, error) {
	section := new(Section)
	_, err := c.request(ctx, "PUT", externalSectionQuery(externalID), su, nil, opt, section)
	return *section, err
}

// UpdateSection updates a section.
//
// https://asana.com/developers/api-reference/sections#update
func (c *Client) UpdateSection(ctx context.Context, id int64, su SectionUpdate, opt *Filter) (Section, error) {
	section := new(Section)
	_, err := c.request(ctx, "PUT", fmt.Sprintf("sections/%d", id), su, nil, opt, section)
	return *section, err
}

// CreateSection creates a section.
//
// https://asana.com/developers/api-reference/sections#create
func (c *Client) CreateSection(ctx context.Context, fields map[string]interface{}, opts *Filter) (Section, error) {
	section := new(Section)
	_, err := c.request(ctx, "POST", "sections", fields, nil, opts, section)
	return *section, err
}

// ListProjectSections gets sections in the project.
//
// https://asana.com/developers/api-reference/sections#find-project
func (c *Client) ListProjectSections(ctx context.Context, projectID int64, opt *Filter) ([]Section, error) {
	rets := []Section{}
	if err := c.pagenate(ctx, fmt.Sprintf("projects/%d/sections", projectID), opt, &rets); err != nil {
		return nil, err
	}
	return rets, nil
}
