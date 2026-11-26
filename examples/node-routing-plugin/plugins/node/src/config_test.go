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

// Shape of the startup_config the router sends, with Kompass subgraph names.
const startupConfig = `{"subgraphs":[
	{"name":"cms-dgs","routing_url":"http://cms-dgs:8080/graphql"},
	{"name":"comments-dgs","routing_url":"http://comments-dgs:8080/graphql"},
	{"name":"poc-dgs","routing_url":"http://poc-dgs:8080/graphql"},
	{"name":"sport-dgs","routing_url":"dns:///sport-dgs:9090"}
]}`

func subgraphURLs(t *testing.T) map[string]string {
	urls, err := SubgraphURLsFromStartupConfig(startupConfig)
	require.NoError(t, err)
	return urls
}

func writeConfig(t *testing.T, content string) string {
	p := filepath.Join(t.TempDir(), "routing.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig("../routing.yaml", subgraphURLs(t))
	require.NoError(t, err)
	assert.Equal(t, map[string]RouteConfig{
		"kmp:cms":      {Subgraph: "cms-dgs", URL: "http://cms-dgs:8080/graphql", Resolver: "nodeCms"},
		"kmp:comments": {Subgraph: "comments-dgs", URL: "http://comments-dgs:8080/graphql", Resolver: "nodeComments"},
		"poc":          {Subgraph: "poc-dgs", URL: "http://poc-dgs:8080/graphql", Resolver: "publicationServicePoc"},
	}, cfg.Routes)
}

func TestLoadConfigURLOverride(t *testing.T) {
	t.Setenv("UGC_URL", "http://ugc.internal/graphql")
	cfg, err := LoadConfig(writeConfig(t, "routes:\n  kmp:ugc:\n    subgraph: ugc-dgs\n    url: ${UGC_URL}\n"), subgraphURLs(t))
	require.NoError(t, err)
	assert.Equal(t, "http://ugc.internal/graphql", cfg.Routes["kmp:ugc"].URL)
	assert.Equal(t, "nodeUgc", cfg.Routes["kmp:ugc"].Resolver)
}

func TestLoadConfigErrors(t *testing.T) {
	for name, content := range map[string]string{
		"no routes":          "routes: {}\n",
		"no subgraph or url": "routes:\n  kmp:cms: {}\n",
		"unknown subgraph":   "routes:\n  kmp:cms:\n    subgraph: cms\n",
		"grpc subgraph":      "routes:\n  kmp:sport:\n    subgraph: sport-dgs\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, content), subgraphURLs(t))
			assert.Error(t, err)
		})
	}
}

func TestSubgraphURLsFromStartupConfig(t *testing.T) {
	urls, err := SubgraphURLsFromStartupConfig("")
	require.NoError(t, err)
	assert.Empty(t, urls)

	_, err = SubgraphURLsFromStartupConfig("{")
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

	r := NewResolver(Config{Routes: map[string]RouteConfig{"kmp:cms": {URL: srv.URL, Resolver: "nodeCms"}}})

	typename, err := r.TypeOf(context.Background(), articleID)
	require.NoError(t, err)
	assert.Equal(t, "Article", typename)

	typename, err = r.TypeOf(context.Background(), unknownCMS)
	require.NoError(t, err)
	assert.Empty(t, typename)

	_, err = r.TypeOf(context.Background(), "kmp:cms:YnJva2Vu")
	assert.ErrorContains(t, err, "boom")
}
