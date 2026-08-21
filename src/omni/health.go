package omni

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Check struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Healthy bool   `json:"healthy"`
	Status  int    `json:"status,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func HTTPCheck(name, url, token string, timeout time.Duration) Check {
	c := Check{Name: name, URL: url}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		c.Detail = err.Error()
		return c
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		c.Detail = err.Error()
		return c
	}
	defer resp.Body.Close()
	c.Status = resp.StatusCode
	c.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	var v any
	if json.NewDecoder(resp.Body).Decode(&v) == nil {
		c.Data = v
	}
	if !c.Healthy {
		c.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return c
}
func LocalURL(bind, port string) string {
	host := strings.TrimSpace(bind)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + host + ":" + port
}
