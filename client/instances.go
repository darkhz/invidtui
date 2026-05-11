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

	host := FullHost()

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
	host = FullHost()

	res, err := request(Ctx(), http.MethodHead, API+"search", nil)
	if err == nil && res.StatusCode == 200 {
		return host, nil
	}

	return "", fmt.Errorf("Client: Cannot select instance")
}

func NormalizeInstance(raw string) string {
	if raw == "" {
		return raw
	}
	if !strings.Contains(raw, "://") {
		raw = "//" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u.String()
}

func GetBestInstance(custom string) (string, error) {
	var bestInstance string

	if custom != "" {
		hasScheme := strings.Contains(custom, "://")
		normalized := NormalizeInstance(custom)
		if inst, err := CheckInstance(normalized); err == nil || hasScheme {
			return inst, err
		}
		u, _ := url.Parse(normalized)
		u.Scheme = "http"
		return CheckInstance(u.String())
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
