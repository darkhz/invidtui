package lib

import (
	"encoding/json"
)

// SubResult stores the subscription data.
type SubResult []struct {
	Author   string `json:"author"`
	AuthorID string `json:"authorId"`
}

// Subscriptions gets the user's subscriptions.
func (c *Client) Subscriptions() (SubResult, error) {
	var result SubResult

	res, err := c.ClientRequest(ClientCtx(), "auth/subscriptions/", GetToken())
	if err != nil {
		return SubResult{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return SubResult{}, err
	}

	return result, nil
}

// AddSubscription adds a subscription.
func (c *Client) AddSubscription(id string) error {
	_, err := c.ClientSend("auth/subscriptions/"+id, "", GetToken())

	return err
}

// DeleteSubscription deletes a subscription.
func (c *Client) DeleteSubscription(id string) error {
	_, err := c.ClientDelete("auth/subscriptions/"+id, GetToken())

	return err
}
