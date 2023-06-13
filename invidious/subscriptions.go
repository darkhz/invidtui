package invidious

import (
	"encoding/json"

	"github.com/darkhz/invidtui/client"
)

// SubscriptionData stores information about the user's subscriptions.
type SubscriptionData []struct {
	Author   string `json:"author"`
	AuthorID string `json:"authorId"`
}

// Subscriptions retrieves the user's subscriptions.
func Subscriptions() (SubscriptionData, error) {
	var data SubscriptionData

	res, err := client.Fetch(client.Ctx(), "auth/subscriptions", client.Token())
	if err != nil {
		return SubscriptionData{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return SubscriptionData{}, err
	}

	return data, nil
}

// AddSubscription adds a channel to the user's subscriptions.
func AddSubscription(id string) error {
	_, err := client.Send("auth/subscriptions/"+id, "", client.Token())

	return err
}

// RemoveSubscription removes a user's subscription.
func RemoveSubscription(id string) error {
	_, err := client.Remove("auth/subscriptions/"+id, client.Token())

	return err
}
