package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client stores the host and http client data.
type Client struct {
	host   string
	client *http.Client
}

const api = "/api/v1/"
const instanceApi = "https://api.invidious.io/instances.json?sort_by=api,health"
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"

var (
	clientCtx     context.Context
	clientCancel  context.CancelFunc
	cliSendCtx    context.Context
	cliSendCancel context.CancelFunc

	currentClient *Client

	clientLock sync.Mutex
)

// NewClient creates a new client.
func NewClient(host string) *Client {
	return &Client{
		host: host,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UpdateClient queries available instances and updates the client.
func UpdateClient() error {
	if currentClient != nil {
		return nil
	}

	client, err := queryInstances()
	if err != nil {
		return err
	}

	currentClient = client

	return nil
}

// GetClient returns the Current client.
func GetClient() *Client {
	clientLock.Lock()
	defer clientLock.Unlock()

	return currentClient
}

// SetClient sets the current client.
func SetClient(instance string) {
	clientLock.Lock()
	defer clientLock.Unlock()

	currentClient = NewClient(instance)
}

// SetRequest sets the request type, sends a request and returns a response.
func (c *Client) SetRequest(ctx context.Context, method, param string, body io.Reader, token ...string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.host+param, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	if method == http.MethodPost || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != nil {
		if IsValidJSON(token[0]) {
			req.Header.Set("Authorization", "Bearer "+token[0])
		} else {
			req.AddCookie(
				&http.Cookie{
					Name:  "SID",
					Value: token[0],
				},
			)
		}
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, clientError(err)
	}

	return res, nil
}

// GetRequest sends a GET request to a url and returns a response.
func (c *Client) GetRequest(ctx context.Context, param string, token ...string) (*http.Response, error) {
	res, err := c.SetRequest(ctx, http.MethodGet, param, nil, token...)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound || res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, err
}

// PostRequest sends a POST request to a url and returns a response.
func (c *Client) PostRequest(ctx context.Context, param, body string, token ...string) (*http.Response, error) {
	res, err := c.SetRequest(ctx, http.MethodPost, param, bytes.NewBuffer([]byte(body)), token...)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 204 {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, err
}

// DeleteRequest sends a DELETE request to a url and returns a response.
func (c *Client) DeleteRequest(ctx context.Context, param string, token ...string) (*http.Response, error) {
	res, err := c.SetRequest(ctx, http.MethodDelete, param, nil, token...)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 204 {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, err
}

// PatchRequest sends a PATCH request to a url and returns a response.
func (c *Client) PatchRequest(ctx context.Context, param, body string, token ...string) (*http.Response, error) {
	res, err := c.SetRequest(ctx, http.MethodPatch, param, bytes.NewBuffer([]byte(body)), token...)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 204 {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, err
}

// ClientRequest sends a GET request to the API and returns a response.
func (c *Client) ClientRequest(ctx context.Context, param string, token ...string) (*http.Response, error) {
	return c.GetRequest(ctx, api+param, token...)
}

// ClientSend sends a POST request to the API and returns a response.
func (c *Client) ClientSend(param, body string, token ...string) (*http.Response, error) {
	ClientSendCancel()

	return c.PostRequest(ClientSendCtx(), api+param, body, token...)
}

// ClientDelete sends a DELETE request to the API and returns a response.
func (c *Client) ClientDelete(param string, token ...string) (*http.Response, error) {
	ClientSendCancel()

	return c.DeleteRequest(ClientSendCtx(), api+param, token...)
}

// ClientPatch sends a PATCH request to the API and returns a response.
func (c *Client) ClientPatch(param, body string, token ...string) (*http.Response, error) {
	ClientSendCancel()

	return c.PatchRequest(ClientSendCtx(), api+param, body, token...)
}

// SelectedInstance returns the current client's hostname.
func (c *Client) SelectedInstance() string {
	return GetHostname(c.host)
}

// ClientCtx returns the client's context.
func ClientCtx() context.Context {
	ClientCancel()

	return clientCtx
}

// ClientCancel cancels and renews the client context.
func ClientCancel() {
	if clientCtx != nil {
		clientCancel()
	}

	clientCtx, clientCancel = context.WithCancel(context.Background())
}

// ClientSendCtx returns the client's send context.
func ClientSendCtx() context.Context {
	return cliSendCtx
}

// ClientSendCancel cancels and renews the client send context.
func ClientSendCancel() {
	if cliSendCtx != nil {
		cliSendCancel()
	}

	cliSendCtx, cliSendCancel = context.WithCancel(context.Background())
}

// CheckInstance checks if an instance is functional.
func CheckInstance(cli *Client, inst string) (string, error) {
	insturl := "https://" + inst

	if strings.Contains(insturl, ".onion") {
		return "", fmt.Errorf("Invalid URL")
	}

	req, err := http.NewRequestWithContext(ClientCtx(), "HEAD", insturl+api+"search", nil)
	req.Header.Set("User-Agent", userAgent)
	if err != nil {
		return "", err
	}

	res, err := cli.client.Do(req)
	if err == nil && res.StatusCode == 200 {
		return insturl, nil
	}

	return "", err
}

// GetInstanceList returns a list of instances.
func GetInstanceList() ([]string, error) {
	var instances [][]interface{}
	var list []string

	cli := NewClient(instanceApi)

	res, err := cli.GetRequest(ClientCtx(), "")
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&instances)
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if inst, ok := instance[0].(string); ok {
			if !strings.Contains(inst, ".onion") {
				list = append(list, inst)
			}
		}
	}

	return list, nil
}

// queryInstances searches for the best instance and returns a Client.
func queryInstances() (*Client, error) {
	var bestInstance string

	cli := NewClient(instanceApi)

	if customInstance != "" {
		if uri, err := url.Parse(customInstance); err == nil {
			host := uri.Hostname()

			if host != "" {
				customInstance = host
			}
		}

		inst, err := CheckInstance(cli, customInstance)
		if err != nil {
			return nil, err
		}

		return NewClient(inst), nil
	}

	instances, err := GetInstanceList()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if inst, err := CheckInstance(cli, instance); err == nil {
			bestInstance = inst
			break
		}
	}

	if bestInstance == "" {
		return nil, fmt.Errorf("Cannot find an instance")
	}

	return NewClient(bestInstance), nil
}

// clientError returns a suitable error message for common http errors.
func clientError(err error) error {
	if err, ok := err.(net.Error); ok {
		e := err.(net.Error)
		switch {
		case e.Timeout():
			return fmt.Errorf("Connection has timed out")
		case e.Temporary():
			return fmt.Errorf("Temporary failure in name resolution")
		}
	}

	return err
}
