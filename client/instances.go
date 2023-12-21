package client

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/darkhz/invidtui/resolver"
)

// Instance returns the client's current instance.
func Instance() string {
	return Host()
}

// GetInstances returns a list of instances.
func GetInstances() ([]string, error) {
	var instances [][]interface{}
	var list []string

	host := Instance()

	dataURI := SetHost(InstanceData)

	res, err := Get(Ctx(), fmt.Sprintf("%s?%s", dataURI.Path, dataURI.RawQuery))
	if err != nil {
		return nil, err
	}

	err = resolver.DecodeJSONReader(res.Body, &instances)
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if inst, ok := instance[0].(string); ok {
			if !strings.Contains(inst, ".onion") && !strings.Contains(inst, ".i2p") {
				list = append(list, inst)
			}
		}
	}

	SetHost(host)

	return list, nil
}

// CheckInstance returns if the provided instance is valid.
func CheckInstance(host string) (string, error) {
	if strings.Contains(host, ".onion") || strings.Contains(host, ".i2p") {
		return "", fmt.Errorf("Client: Invalid URL")
	}

	SetHost(host)
	host = Instance()

	res, err := request(Ctx(), http.MethodHead, API+"search", nil)
	if err == nil && res.StatusCode == 200 {
		return host, nil
	}

	return "", fmt.Errorf("Client: Cannot select instance")
}

// GetBestInstance determines and returns the best instance.
func GetBestInstance(custom string) (string, error) {
	var bestInstance string

	if custom != "" {
		if uri, err := url.Parse(custom); err == nil {
			host := uri.Hostname()
			if host != "" {
				custom = host
			}
		}

		return CheckInstance(custom)
	}

	instances, err := GetInstances()
	if err != nil {
		return "", err
	}

	for _, instance := range instances {
		if inst, err := CheckInstance(instance); err == nil {
			bestInstance = inst
			break
		}
	}

	if bestInstance == "" {
		return "", fmt.Errorf("Client: Cannot find an instance")
	}

	return bestInstance, nil
}
