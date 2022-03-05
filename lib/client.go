package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client stores the host and http client data.
type Client struct {
	host   string
	client *http.Client
}

const api = "/api/v1/"
const instanceApi = "https://api.invidious.io/instances.json?sort_by=health"
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"

var (
	clientCtx     context.Context
	clientCancel  context.CancelFunc
	currentClient *Client
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
	client, err := queryInstances()
	if err != nil {
		return err
	}

	currentClient = client

	return nil
}

// GetClient returns the Current client.
func GetClient() *Client {
	return currentClient
}

// SendRequest sends a request to a url and returns a response.
func SendRequest(ctx context.Context, c *Client, param string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+param, nil)
	req.Header.Set("User-Agent", userAgent)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, clientError(err)
	}

	if res.StatusCode == http.StatusNotFound || res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned %d", res.StatusCode)
	}

	return res, nil
}

// ClientRequest sends a request to the API and returns a response.
func (c *Client) ClientRequest(ctx context.Context, param string) (*http.Response, error) {
	res, err := SendRequest(ctx, c, api+param)

	return res, err
}

// SelectedInstance returns the current client's hostname.
func (c *Client) SelectedInstance() string {
	uri, _ := url.Parse(c.host)

	return uri.Hostname()
}

// queryInstances searches for the best instance and returns a Client.
func queryInstances() (*Client, error) {
	var bestInstance string
	var instances [][]interface{}

	ctx := context.Background()
	cli := NewClient(instanceApi)

	checkInstance := func(inst string) (string, bool) {
		insturl := "https://" + inst

		if strings.Contains(insturl, ".onion") {
			return "", false
		}

		req, err := http.NewRequest("HEAD", insturl+api+"search", nil)
		req.Header.Set("User-Agent", userAgent)
		if err != nil {
			return "", false
		}

		res, err := cli.client.Do(req)
		if err == nil && res.StatusCode == 200 {
			return insturl, true
		}

		return "", false
	}

	if customInstance != "" {
		if uri, err := url.Parse(customInstance); err == nil {
			host := uri.Hostname()

			if host != "" {
				customInstance = host
			}
		}

		if inst, ok := checkInstance(customInstance); ok {
			return NewClient(inst), nil
		}
	}

	res, err := SendRequest(ctx, cli, "")
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(res.Body).Decode(&instances)
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if inst, ok := checkInstance(instance[0].(string)); ok {
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
