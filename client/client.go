package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/darkhz/invidtui/resolver"
	"github.com/darkhz/invidtui/utils"
)

const (
	// API is the api endpoint.
	API = "/api/v1/"

	// InstanceData is the URL to retrieve available Invidious instances.
	InstanceData = "https://api.invidious.io/instances.json?sort_by=api,health"

	// UserAgent is the user agent for the client.
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"
)

// Client stores information about a client.
type Client struct {
	uri *url.URL

	rctx, sctx       context.Context
	rcancel, scancel context.CancelFunc

	mutex sync.Mutex

	*http.Client
}

var client Client

// Init intitializes the client.
func Init() {
	client = Client{}
	client.Client = &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
		},
	}

	client.rctx, client.rcancel = context.WithCancel(context.Background())
	client.sctx, client.scancel = context.WithCancel(context.Background())
}

// Host returns the client's host.
func Host() string {
	if client.uri == nil {
		return ""
	}

	return client.uri.Scheme + "://" + client.uri.Hostname()
}

// SetHost sets the client's host.
func SetHost(host string) *url.URL {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.uri, _ = url.Parse(host)
	if client.uri.Scheme == "" {
		client.uri.Scheme = "https"
		client.uri, _ = url.Parse(client.uri.String())
	}

	return client.uri
}

// Get send a GET request to the host and returns a response
func Get(ctx context.Context, param string, token ...string) (*http.Response, error) {
	res, err := request(ctx, http.MethodGet, param, nil, token...)
	if err != nil {
		return nil, err
	}

	return checkStatusCode(res, http.StatusOK)
}

// Post send a POST request to the host and returns a response.
func Post(ctx context.Context, param, body string, token ...string) (*http.Response, error) {
	res, err := request(ctx, http.MethodPost, param, bytes.NewBuffer([]byte(body)), token...)
	if err != nil {
		return nil, err
	}

	return checkStatusCode(res, 201, 204)
}

// Delete send a DELETE request to the host and returns a response.
func Delete(ctx context.Context, param string, token ...string) (*http.Response, error) {
	res, err := request(ctx, http.MethodDelete, param, nil, token...)
	if err != nil {
		return nil, err
	}

	return checkStatusCode(res, 204)
}

// Patch send a PATCH request to the host and returns a response.
func Patch(ctx context.Context, param, body string, token ...string) (*http.Response, error) {
	res, err := request(ctx, http.MethodPatch, param, bytes.NewBuffer([]byte(body)), token...)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 204 {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, err
}

// Fetch sends a GET request to the API endpoint and returns a response.
func Fetch(ctx context.Context, param string, token ...string) (*http.Response, error) {
	return Get(ctx, API+param, token...)
}

// Send sends a POST request to the API endpoint and returns a response.
func Send(param, body string, token ...string) (*http.Response, error) {
	SendCancel()

	return Post(SendCtx(), API+param, body, token...)
}

// Remove sends a DELETE request to the API endpoint and returns a response.
func Remove(param string, token ...string) (*http.Response, error) {
	SendCancel()

	return Delete(SendCtx(), API+param, token...)
}

// Modify sends a PATCH request to the API endpoint and returns a response.
func Modify(param, body string, token ...string) (*http.Response, error) {
	SendCancel()

	return Patch(SendCtx(), API+param, body, token...)
}

// Ctx returns the client's current context.
func Ctx() context.Context {
	return client.rctx
}

// Cancel cancels the client's context.
func Cancel() {
	if client.rctx != nil {
		client.rcancel()
	}

	client.rctx, client.rcancel = context.WithCancel(context.Background())
}

// SendCtx returns the client's send context.
func SendCtx() context.Context {
	return client.sctx
}

// SendCancel cancels the client's send context.
func SendCancel() {
	if client.sctx != nil {
		client.scancel()
	}

	client.sctx, client.scancel = context.WithCancel(context.Background())
}

// request sends a HTTP request to the URL and returns a response.
func request(ctx context.Context, method, param string, body io.Reader, token ...string) (*http.Response, error) {
	if client.uri == nil {
		return nil, fmt.Errorf("Client: Not initialized")
	}

	req, err := http.NewRequestWithContext(ctx, method, Host()+param, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	if method == http.MethodPost || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != nil {
		if utils.IsValidJSON(token[0]) {
			req.Header.Set("Authorization", "Bearer "+token[0])
		} else {
			req.AddCookie(
				&http.Cookie{
					Name:  "SID",
					Value: utils.SanitizeCookie(token[0]),
				},
			)
		}
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, netError(err)
	}

	return res, nil
}

// checkStatusCode checks and returns an error if the codes don't match the response's status code.
func checkStatusCode(res *http.Response, codes ...int) (*http.Response, error) {
	var checked int

	for _, code := range codes {
		if res.StatusCode != code {
			checked++
		}
	}

	if checked == len(codes) {
		var responseError struct {
			Error string `json:"error"`
		}

		message := "API request returned %d"

		if err := resolver.DecodeJSONReader(res.Body, &responseError); err == nil {
			message += ": " + responseError.Error
		}

		return nil, fmt.Errorf(message, res.StatusCode)
	}

	return res, nil
}

// netError returns messages for common network errors.
func netError(err error) error {
	if err, ok := err.(net.Error); ok {
		switch {
		case err.Timeout():
			return fmt.Errorf("Client: Connection has timed out")
		}
	}

	return err
}
