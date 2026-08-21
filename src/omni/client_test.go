package omni

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModels(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatal(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"a"},{"id":"b"}]}`))
	}))
	defer s.Close()
	c := Client{BaseURL: s.URL}
	r, err := c.Models()
	if err != nil {
		t.Fatal(err)
	}
	ids, err := ModelIDs(r)
	if err != nil || len(ids) != 2 {
		t.Fatal(ids, err)
	}
}
