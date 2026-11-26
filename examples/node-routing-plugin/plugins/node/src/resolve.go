package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node/src/kompassid"
)

// TypeLookup asks the subgraph that owns an ID for its concrete GraphQL type.
// It returns "" if the subgraph does not know the ID.
type TypeLookup func(ctx context.Context, id string) (string, error)

// InvalidIDError marks IDs that cannot be routed because they are malformed.
type InvalidIDError struct{ Err error }

func (e InvalidIDError) Error() string { return e.Err.Error() }
func (e InvalidIDError) Unwrap() error { return e.Err }

// Resolver routes an ID to the subgraph that owns it, by prefix, and lets
// that subgraph tell the concrete type.
type Resolver struct {
	// Routes maps an ID prefix (kmp:<domain>, or the first segment of other
	// IDs) to the lookup of the owning subgraph.
	Routes map[string]TypeLookup
}

// NewResolver builds a Resolver that calls each subgraph's node<Domain> field.
func NewResolver(cfg Config) *Resolver {
	r := &Resolver{Routes: map[string]TypeLookup{}}
	for prefix, route := range cfg.Routes {
		r.Routes[prefix] = subgraphTypeLookup(route.URL, route.Resolver)
	}
	return r
}

// TypeOf returns the GraphQL type name for id, or "" if nothing owns it.
func (r *Resolver) TypeOf(ctx context.Context, id string) (string, error) {
	prefix, err := routingPrefix(id)
	if err != nil {
		return "", InvalidIDError{err}
	}
	lookup, ok := r.Routes[prefix]
	if !ok {
		if strings.HasPrefix(prefix, kompassid.Namespace+":") {
			// Valid Kompass ID of a domain this graph does not serve.
			return "", nil
		}
		return "", InvalidIDError{fmt.Errorf("unrecognised ID %q", id)}
	}
	return lookup(ctx, id)
}

// routingPrefix is kmp:<domain> for Kompass IDs (validated against the ID
// format) and the first segment for any other ID.
func routingPrefix(id string) (string, error) {
	first, _, found := strings.Cut(id, ":")
	if !found || first == "" {
		return "", fmt.Errorf("unrecognised ID %q", id)
	}
	if first != kompassid.Namespace {
		return first, nil
	}
	kid, err := kompassid.Parse(id)
	if err != nil {
		return "", err
	}
	return kompassid.Namespace + ":" + string(kid.Domain), nil
}
