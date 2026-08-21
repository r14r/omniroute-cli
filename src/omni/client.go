package omni

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL, ManagementToken, InferenceToken string
	Timeout                                  time.Duration
}

type Response struct {
	Status int    `json:"status"`
	Body   any    `json:"body,omitempty"`
	Raw    string `json:"raw,omitempty"`
}

func (c Client) do(method, path, token string, body []byte) (Response, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	hc := http.Client{Timeout: timeout}
	resp, err := hc.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Response{}, err
	}
	r := Response{Status: resp.StatusCode, Raw: string(b)}
	if len(bytes.TrimSpace(b)) > 0 {
		var v any
		if json.Unmarshal(b, &v) == nil {
			r.Body = v
			r.Raw = ""
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return r, fmt.Errorf("HTTP %d from %s", resp.StatusCode, path)
	}
	return r, nil
}
func (c Client) Management(method, path string) (Response, error) {
	return c.do(method, path, c.ManagementToken, nil)
}
func (c Client) Inference(method, path string) (Response, error) {
	return c.do(method, path, c.InferenceToken, nil)
}
func (c Client) Health() (Response, error) {
	return c.Management(http.MethodGet, "/api/monitoring/health")
}
func (c Client) Models() (Response, error) {
	return c.Inference(http.MethodGet, "/v1/models?prefix=alias")
}
func (c Client) Providers() (Response, error) { return c.Management(http.MethodGet, "/api/providers") }
func (c Client) Sessions() (Response, error)  { return c.Management(http.MethodGet, "/api/sessions") }
func (c Client) Usage() (Response, error)     { return c.Management(http.MethodGet, "/api/usage/history") }
func (c Client) CacheStats() (Response, error) {
	return c.Management(http.MethodGet, "/api/cache/stats")
}
func (c Client) ClearCache() (Response, error) {
	return c.Management(http.MethodDelete, "/api/cache/stats")
}

func ModelIDs(r Response) ([]string, error) {
	m, ok := r.Body.(map[string]any)
	if !ok {
		return nil, errors.New("unexpected models response")
	}
	data, ok := m["data"].([]any)
	if !ok {
		return nil, errors.New("models response missing data")
	}
	ids := make([]string, 0, len(data))
	for _, x := range data {
		if o, ok := x.(map[string]any); ok {
			if id, ok := o["id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}
