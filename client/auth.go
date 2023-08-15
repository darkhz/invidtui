package client

import (
	"fmt"
	"net/url"
	"sync"
)

// Auth stores information about instances and the user's authentication
// tokens associated with it.
type Auth struct {
	store map[string]string

	mutex sync.Mutex
}

// Credential stores an instance and the user's authentication token.
type Credential struct {
	Instance string `json:"instance"`
	Token    string `json:"token"`
}

// Scopes lists the user token's scopes.
const Scopes = "GET:playlists*,GET:subscriptions*,GET:feed*,GET:notifications*,GET:tokens*"

var auth Auth

// SetAuthCredentials sets the authentication credentials.
func SetAuthCredentials(credentials []Credential) {
	auth.store = make(map[string]string)

	for _, credential := range credentials {
		auth.store[credential.Instance] = credential.Token
	}
}

// GetAuthCredentials returns the authentication credentials.
func GetAuthCredentials() []Credential {
	var creds []Credential

	for instance, token := range auth.store {
		creds = append(creds, Credential{instance, token})
	}

	return creds
}

// AddAuth adds and stores an instance and token credential.
func AddAuth(instance, token string) {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	if instance == "" || token == "" {
		return
	}

	instanceURI, _ := url.Parse(instance)
	if instanceURI.Scheme == "" {
		instanceURI.Scheme = "https"
	}

	auth.store[instanceURI.String()] = token
}

// AddCurrentAuth adds and stores an instance and token credential
// for the selected instance.
func AddCurrentAuth(token string) {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	auth.store[Instance()] = token
}

// Token returns the stored token for the selected instance.
func Token() string {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	return auth.store[Instance()]
}

// AuthLink returns an authorization link.
func AuthLink(instance ...string) string {
	if instance == nil {
		instance = append(instance, Instance())
	}

	return fmt.Sprintf("%s/authorize_token?scopes=%s",
		instance[0], Scopes,
	)
}

// IsTokenValid tests the validity of the given token.
func IsTokenValid(token string) bool {
	_, err := Fetch(Ctx(), "auth/tokens", token)

	return err == nil
}

// CurrentTokenValid tests the validity of the stored token.
func CurrentTokenValid() bool {
	return IsTokenValid(Token())
}

// IsAuthInstance checks whether the selected instance
// has a stored token.
func IsAuthInstance() bool {
	return Token() != ""
}
