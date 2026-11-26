package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node/src/kompassid"
)

// TypeLookup asks the subgraph that owns an ID for its concrete GraphQL type.
// It returns "" if the subgraph does not know the ID.
type TypeLookup func(ctx context.Context, id string) (string, error)

// InvalidIDError marks IDs that cannot be parsed at all.
type InvalidIDError struct{ Err error }

func (e InvalidIDError) Error() string { return e.Err.Error() }
func (e InvalidIDError) Unwrap() error { return e.Err }

// Legacy IDs of the programme service: poc:<tv|radio>:<dvbId>.
var pocID = regexp.MustCompile(`^poc:(tv|radio):[0-9]+$`)

// Resolver works out the GraphQL type behind an ID. The type is read from the
// ID wherever the ID carries it, so most lookups need no network call.
type Resolver struct {
	// CMSLookup resolves kmp:cms IDs. CMS IDs carry no type segment, so the
	// CMS subgraph has to be asked.
	CMSLookup TypeLookup
}

// TypeOf returns the GraphQL type name for id, or "" if no subgraph owns it.
func (r *Resolver) TypeOf(ctx context.Context, id string) (string, error) {
	if m := pocID.FindStringSubmatch(id); m != nil {
		if m[1] == "tv" {
			return "PublicationServiceTV", nil
		}
		return "PublicationServiceRadio", nil
	}

	kid, err := kompassid.Parse(id)
	if err != nil {
		return "", InvalidIDError{err}
	}

	switch kid.Domain {
	case kompassid.Comments:
		// The comments domain only exposes comments; a type segment, if
		// present, must say so.
		if kid.Type == "" || kid.Type == "comment" {
			return "Comment", nil
		}
		return "", nil
	case kompassid.CMS:
		if r.CMSLookup == nil {
			return "", fmt.Errorf("no CMS lookup configured")
		}
		return r.CMSLookup(ctx, id)
	default:
		// Valid Kompass ID, but no subgraph in this graph serves the domain.
		return "", nil
	}
}
