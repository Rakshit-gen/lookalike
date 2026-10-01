// Package nucla is a small client for NuclaDB's REST API. It covers only
// the calls lookalike makes: creating a tenant, upserting, deleting and
// searching vectors.
package nucla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string // sent as a Bearer token when the server runs with -api-keys
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

type Vector struct {
	ID       string            `json:"id"`
	Values   []float32         `json:"values"`
	Metadata map[string]string `json:"metadata,omitempty"`
	TenantID string            `json:"tenant_id,omitempty"`
}

type Match struct {
	ID       string            `json:"id"`
	Score    float32           `json:"score"` // cosine distance: 0 is identical
	Metadata map[string]string `json:"metadata"`
}

// Error is a non-2xx response from the server.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("nucladb: HTTP %d: %s", e.Status, e.Message)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("nucladb: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error string `json:"error"`
		}
		raw, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(raw, &e) != nil || e.Error == "" {
			e.Error = strings.TrimSpace(string(raw))
		}
		return &Error{Status: resp.StatusCode, Message: e.Error}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// EnsureTenant creates a cosine tenant with the given dimension. An
// existing tenant is not an error.
func (c *Client) EnsureTenant(ctx context.Context, tenant string, dim int) error {
	err := c.do(ctx, http.MethodPost, "/v1/tenants", map[string]any{
		"tenant_id": tenant, "dim": dim, "metric": "cosine",
	}, nil)
	if e, ok := err.(*Error); ok && e.Status == http.StatusConflict {
		return nil
	}
	return err
}

// upsertChunk keeps each request well under the server's 64 MiB body cap.
// A 1024-dim vector is roughly 12 KB of JSON.
const upsertChunk = 500

func (c *Client) Upsert(ctx context.Context, tenant string, vs []Vector) error {
	for i := range vs {
		vs[i].TenantID = tenant
	}
	for start := 0; start < len(vs); start += upsertChunk {
		end := min(start+upsertChunk, len(vs))
		if err := c.do(ctx, http.MethodPost, "/v1/vectors:batch", map[string]any{"vectors": vs[start:end]}, nil); err != nil {
			return err
		}
	}
	return nil
}

// Where is one metadata filter clause. Ops are NuclaDB's: eq, ne, in,
// not_in, gt, gte, lt, lte (numeric) and exists.
type Where struct {
	Key    string   `json:"key"`
	Op     string   `json:"op"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

// Search returns the topK nearest vectors whose metadata matches every
// clause in where.
func (c *Client) Search(ctx context.Context, tenant string, query []float32, topK int, where ...Where) ([]Match, error) {
	var out struct {
		Matches []Match `json:"matches"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/search", map[string]any{
		"tenant_id": tenant, "query": query, "top_k": topK, "where": where,
	}, &out)
	return out.Matches, err
}

// Delete removes one vector. Deleting an id that isn't there is not an
// error.
func (c *Client) Delete(ctx context.Context, tenant, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/vectors/"+url.PathEscape(id)+"?tenant_id="+url.QueryEscape(tenant), nil, nil)
}
