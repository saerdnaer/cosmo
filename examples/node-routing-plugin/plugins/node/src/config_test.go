package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("CMS_SUBGRAPH_URL", "http://cms.internal/graphql")
	t.Setenv("COMMENTS_SUBGRAPH_URL", "")

	cfg, err := LoadConfig("../routing.yaml")
	require.NoError(t, err)
	assert.Equal(t, map[string]RouteConfig{
		"kmp:cms":      {URL: "http://cms.internal/graphql", Field: "nodeCms"},
		"kmp:comments": {URL: "http://localhost:4002/graphql", Field: "nodeComments"},
		"poc":          {URL: "http://localhost:4003/graphql", Field: "nodePoc"},
	}, cfg.Routes)
}

func TestLoadConfigFieldOverrideAndErrors(t *testing.T) {
	write := func(content string) string {
		p := filepath.Join(t.TempDir(), "routing.yaml")
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
		return p
	}

	cfg, err := LoadConfig(write("routes:\n  kmp:ugc:\n    url: http://ugc\n    field: lookupUgc\n"))
	require.NoError(t, err)
	assert.Equal(t, "lookupUgc", cfg.Routes["kmp:ugc"].Field)

	_, err = LoadConfig(write("routes: {}\n"))
	assert.Error(t, err)
	_, err = LoadConfig(write("routes:\n  kmp:cms: {}\n"))
	assert.Error(t, err)
}

func TestSubgraphTypeLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query     string
			Variables map[string]string
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "query($id: ID!) { nodeCms(id: $id) { __typename } }", req.Query)
		switch req.Variables["id"] {
		case articleID:
			_, _ = w.Write([]byte(`{"data":{"nodeCms":{"__typename":"Article"}}}`))
		case "kmp:cms:YnJva2Vu":
			_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}],"data":{"nodeCms":null}}`))
		default:
			_, _ = w.Write([]byte(`{"data":{"nodeCms":null}}`))
		}
	}))
	defer srv.Close()

	r := NewResolver(Config{Routes: map[string]RouteConfig{"kmp:cms": {URL: srv.URL, Field: "nodeCms"}}})

	typename, err := r.TypeOf(context.Background(), articleID)
	require.NoError(t, err)
	assert.Equal(t, "Article", typename)

	typename, err = r.TypeOf(context.Background(), unknownCMS)
	require.NoError(t, err)
	assert.Empty(t, typename)

	_, err = r.TypeOf(context.Background(), "kmp:cms:YnJva2Vu")
	assert.ErrorContains(t, err, "boom")
}
