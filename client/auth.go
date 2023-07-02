package client

import (
	"fmt"
	"os"
	"sync"

	"github.com/darkhz/invidtui/utils"
)

// Auth stores information about instances and the user's authentication
// tokens associated with it.
type Auth struct {
	filename string
	store    map[string]string

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

// LoadAuthFile loads user credentials from the provided file.
func LoadAuthFile(filename string) error {
	var credentials []Credential

	authfile, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer authfile.Close()

	err = utils.JSON().NewDecoder(authfile).Decode(&credentials)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	auth.filename = filename
	auth.store = make(map[string]string)

	for _, instance := range credentials {
		auth.store[instance.Instance] = instance.Token
	}

	return nil
}

// SaveAuthCredentials saves the authentication credentials.
func SaveAuthCredentials() error {
	var credentials []Credential

	if len(auth.store) == 0 {
		return nil
	}

	for instance, token := range auth.store {
		credentials = append(
			credentials,
			Credential{
				Instance: instance,
				Token:    token,
			},
		)
	}

	data, err := utils.JSON().MarshalIndent(credentials, "", " ")
	if err != nil {
		return fmt.Errorf("Credential: Cannot decode auth data: %s", err)
	}

	file, err := os.OpenFile(auth.filename, os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Credential: Cannot open auth data file: %s", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("Credential: Cannot save auth data: %s", err)
	}

	err = file.Sync()
	if err != nil {
		return fmt.Errorf("Credential: Cannot sync auth data: %s", err)
	}

	return nil
}

// AddAuth adds and stores an instance and token credential.
func AddAuth(instance, token string) {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	if instance == "" || token == "" {
		return
	}

	auth.store[instance] = token
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
