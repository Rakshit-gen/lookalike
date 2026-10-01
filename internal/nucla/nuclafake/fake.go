// Package nuclafake is an in-memory stand-in for NuclaDB's REST API, for
// tests. Search is exact brute force, so tests check lookalike's logic and
// not HNSW recall. Only the routes lookalike calls are implemented.
package nuclafake

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"

	"github.com/Rakshit-gen/lookalike/internal/nucla"
)

type Server struct {
	*httptest.Server
	mu      sync.Mutex
	tenants map[string]map[string]nucla.Vector
}

func New() *Server {
	s := &Server{tenants: map[string]map[string]nucla.Vector{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/tenants", s.createTenant)
	mux.HandleFunc("POST /v1/vectors:batch", s.batch)
	mux.HandleFunc("POST /v1/search", s.search)
	mux.HandleFunc("DELETE /v1/vectors/{id}", s.delete)
	s.Server = httptest.NewServer(mux)
	return s
}

// Count is how many vectors a tenant holds.
func (s *Server) Count(tenant string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tenants[tenant])
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID string `json:"tenant_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[body.TenantID]; ok {
		http.Error(w, `{"error":"tenant already exists"}`, http.StatusConflict)
		return
	}
	s.tenants[body.TenantID] = map[string]nucla.Vector{}
	w.Write([]byte(`{}`))
}

func (s *Server) batch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Vectors []nucla.Vector `json:"vectors"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range body.Vectors {
		t, ok := s.tenants[v.TenantID]
		if !ok {
			http.Error(w, `{"error":"tenant not found"}`, http.StatusNotFound)
			return
		}
		t[v.ID] = v
	}
	w.Write([]byte(`{}`))
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID string        `json:"tenant_id"`
		Query    []float32     `json:"query"`
		TopK     int           `json:"top_k"`
		Where    []nucla.Where `json:"where"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []nucla.Match
	for _, v := range s.tenants[body.TenantID] {
		if matches(v.Metadata, body.Where) {
			out = append(out, nucla.Match{ID: v.ID, Score: cosineDistance(body.Query, v.Values), Metadata: v.Metadata})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score < out[j].Score })
	if len(out) > body.TopK {
		out = out[:body.TopK]
	}
	json.NewEncoder(w).Encode(map[string]any{"matches": out})
}

func matches(meta map[string]string, where []nucla.Where) bool {
	for _, c := range where {
		v, ok := meta[c.Key]
		switch c.Op {
		case "eq":
			if v != c.Value {
				return false
			}
		case "lt", "gt":
			a, err1 := strconv.ParseFloat(v, 64)
			b, err2 := strconv.ParseFloat(c.Value, 64)
			if !ok || err1 != nil || err2 != nil || (c.Op == "lt" && a >= b) || (c.Op == "gt" && a <= b) {
				return false
			}
		default:
			panic("nuclafake: op not implemented: " + c.Op)
		}
	}
	return true
}

func cosineDistance(a, b []float32) float32 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 1
	}
	return float32(1 - dot/math.Sqrt(na*nb))
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants[r.URL.Query().Get("tenant_id")], r.PathValue("id"))
	w.Write([]byte(`{"deleted":true}`))
}
