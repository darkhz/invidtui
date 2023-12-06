package invidious

import (
	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/resolver"
)

const subFields = "?fields=author,authorId,error"

// SubscriptionData stores information about the user's subscriptions.
type SubscriptionData []struct {
	Author   string `json:"author"`
	AuthorID string `json:"authorId"`
}

// Subscriptions retrieves the user's subscriptions.
func Subscriptions() (SubscriptionData, error) {
	var data SubscriptionData

	res, err := client.Fetch(client.Ctx(), "auth/subscriptions"+subFields, client.Token())
	if err != nil {
		return SubscriptionData{}, err
	}
	defer res.Body.Close()

	err = resolver.DecodeJSONReader(res.Body, &data)
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
