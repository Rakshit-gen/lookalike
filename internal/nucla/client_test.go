package nucla

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureTenantIgnoresConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"tenant already exists"}`))
	}))
	defer srv.Close()

	if err := New(srv.URL).EnsureTenant(context.Background(), "t", 8); err != nil {
		t.Fatalf("conflict should be fine, got %v", err)
	}
}

func TestErrorCarriesServerMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"dimension mismatch"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Search(context.Background(), "t", []float32{1}, 3)
	var e *Error
	if !errors.As(err, &e) || e.Status != 400 || e.Message != "dimension mismatch" {
		t.Fatalf("got %v", err)
	}
}

func TestSearchSendsTenantAndDecodesMatches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TenantID string `json:"tenant_id"`
			TopK     int    `json:"top_k"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/v1/search" || body.TenantID != "repo" || body.TopK != 2 {
			t.Errorf("unexpected request %s %+v", r.URL.Path, body)
		}
		// NuclaDB omits score when it is exactly 0.
		w.Write([]byte(`{"matches":[{"id":"7","metadata":{"title":"x"}},{"id":"9","score":0.25}]}`))
	}))
	defer srv.Close()

	ms, err := New(srv.URL).Search(context.Background(), "repo", []float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].ID != "7" || ms[0].Score != 0 || ms[1].Score != 0.25 || ms[0].Metadata["title"] != "x" {
		t.Fatalf("got %+v", ms)
	}
}

func TestUpsertSplitsLargeBatches(t *testing.T) {
	var sizes []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Vectors []Vector `json:"vectors"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		sizes = append(sizes, len(body.Vectors))
		for _, v := range body.Vectors {
			if v.TenantID != "repo" {
				t.Errorf("vector %s missing tenant", v.ID)
			}
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	vs := make([]Vector, upsertChunk*2+3)
	if err := New(srv.URL).Upsert(context.Background(), "repo", vs); err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 3 || sizes[0] != upsertChunk || sizes[2] != 3 {
		t.Fatalf("chunk sizes %v", sizes)
	}
}

func TestSearchSendsWhereClauses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Where []Where `json:"where"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Where) != 1 || body.Where[0].Key != "state" || body.Where[0].Op != "eq" || body.Where[0].Value != "open" {
			t.Errorf("where = %+v", body.Where)
		}
		w.Write([]byte(`{"matches":[]}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Search(context.Background(), "repo", []float32{1}, 5, Where{Key: "state", Op: "eq", Value: "open"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIKeySentAsBearer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key-1234567" {
			t.Errorf("Authorization = %q", got)
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.APIKey = "secret-key-1234567"
	if err := c.EnsureTenant(context.Background(), "t", 4); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSendsTenant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/vectors/42" || r.URL.Query().Get("tenant_id") != "lib" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		w.Write([]byte(`{"deleted":true}`))
	}))
	defer srv.Close()

	if err := New(srv.URL).Delete(context.Background(), "lib", "42"); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteTenantIgnoresMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/tenants/gone" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"tenant not found"}`))
	}))
	defer srv.Close()

	if err := New(srv.URL).DeleteTenant(context.Background(), "gone"); err != nil {
		t.Fatal(err)
	}
}
