package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

// AuthInstance stores an authentication credential.
type AuthInstance struct {
	Instance string `json:"instance"`
	Token    string `json:"token"`
}

var (
	authInstance []AuthInstance
	authMap      map[string]string
	authMutex    sync.Mutex
)

const scopes = "GET:playlists*,GET:subscriptions*,GET:feed*,GET:notifications*,GET:tokens*"

// LoadAuth loads the authentication credentials.
func LoadAuth() error {
	authMap = make(map[string]string)

	auth, err := ConfigPath("auth.json")
	if err != nil {
		return err
	}

	authfile, err := os.Open(auth)
	if err != nil {
		return err
	}

	err = json.NewDecoder(authfile).Decode(&authInstance)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	for _, instance := range authInstance {
		authMap[instance.Instance] = instance.Token
	}

	return nil
}

// SaveAuth saves the authentication credentials.
func SaveAuth() error {
	if len(authMap) == 0 {
		return nil
	}

	auth, err := ConfigPath("auth.json")
	if err != nil {
		return err
	}

	authInstance = nil
	for instance, token := range authMap {
		authInstance = append(
			authInstance,
			AuthInstance{
				Instance: instance,
				Token:    token,
			},
		)
	}

	data, err := json.MarshalIndent(authInstance, "", " ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(auth, data, 0664)
	if err != nil {
		return err
	}

	return nil
}

// AddAuth adds and stores an instance and token credential.
func AddAuth(instance, token string) {
	authMutex.Lock()
	defer authMutex.Unlock()

	if instance == "" || token == "" {
		return
	}

	authMap[instance] = token
}

// AddCurrentAuth adds and stores an instance and token credential
// for the selected instance.
func AddCurrentAuth(token string) {
	authMutex.Lock()
	defer authMutex.Unlock()

	authMap[GetClient().SelectedInstance()] = token
}

// GetToken returns the stored token for the selected instance.
func GetToken() string {
	authMutex.Lock()
	defer authMutex.Unlock()

	return authMap[GetClient().SelectedInstance()]
}

// GetAuthLink returns an authorization link.
func GetAuthLink(instance ...string) string {
	var selectedInstance string

	if instance != nil {
		selectedInstance = instance[0]
	} else {
		selectedInstance = GetClient().SelectedInstance()
	}

	return fmt.Sprintf("https://%s/authorize_token?scopes=%s",
		selectedInstance, scopes,
	)
}

// TokenValid tests the validity of the given token.
func TokenValid(token string) bool {
	_, err := GetClient().ClientRequest(context.Background(), "auth/tokens/", token)

	return err == nil
}

// AuthTokenValid tests the validity of the stored token.
func AuthTokenValid() bool {
	return TokenValid(GetToken())
}

// IsAuthInstance checks whether the selected instance
// has a stored token.
func IsAuthInstance() bool {
	return GetToken() != ""
}
