package asana

import (
	"context"
	"fmt"
	"net/url"
)

// GetWebhooks gets webhooks.
//TODO: rename to ListWebhooks
//
// https://asana.com/developers/api-reference/webhooks#get
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

// GetWebhook gets a webhook.
//
// https://asana.com/developers/api-reference/webhooks#get-single
func (c *Client) GetWebhook(ctx context.Context, id int64) (Webhook, error) {
	webhook := new(Webhook)
	err := c.Request(ctx, fmt.Sprintf("webhooks/%d", id), nil, &webhook)
	return *webhook, err
}

// CreateWebhook creates a webhook.
//
// https://asana.com/developers/api-reference/webhooks#create
func (c *Client) CreateWebhook(ctx context.Context, id int64, target string) (Webhook, error) {
	webhook := new(Webhook)
	p := url.Values{
		"resource": []string{fmt.Sprintf("%d", id)},
		"target":   []string{target},
	}
	_, err := c.request(ctx, "POST", "webhooks", nil, p, nil, &webhook)
	return *webhook, err
}

// DeleteWebhook deletes a webhook.
//
// https://asana.com/developers/api-reference/webhooks#delete
func (c *Client) DeleteWebhook(ctx context.Context, id int64) error {
	var resp interface{} // Empty response
	_, err := c.request(ctx, "DELETE", fmt.Sprintf("webhooks/%d", id), nil, nil, nil, &resp)
	return err
}
