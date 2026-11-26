// Package kompassid parses Kompass IDs. It mirrors the rules of the Java
// library gg.kompass:id (KompassId, KompassDomain, KompassReference), which
// stays the source of truth for the format.
//
// Accepted forms (all references base64url-encoded, RFC 4648 §5):
//
//	kmp:<domain>:<reference>
//	kmp:<domain>:<type>:<reference>
//	kmp:<domain>:<parentRef>:<childType>:<childRef>
package kompassid

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

const (
	Namespace = "kmp"
	delimiter = ":"
)

// Domain is a Kompass domain, see KompassDomain in the Java library.
type Domain string

const (
	CMS      Domain = "cms"
	Comments Domain = "comments"
	Event    Domain = "event"
	Sport    Domain = "sport"
	UGC      Domain = "ugc"
)

var domains = []Domain{CMS, Comments, Event, Sport, UGC}

var typePattern = regexp.MustCompile(`^[a-z]+$`)

// ID is a parsed Kompass ID. Reference and ParentReference hold the decoded
// values: a UUID string for 22-character references, otherwise the UTF-8 text.
type ID struct {
	Domain          Domain
	Type            string
	ParentReference string
	Reference       string
}

// Parse parses a serialised Kompass ID.
func Parse(raw string) (ID, error) {
	parts := strings.Split(raw, delimiter)
	if len(parts) < 3 || len(parts) > 5 {
		return ID{}, fmt.Errorf("invalid identifier %q: want kmp:<domain>[:<type>]:<ref> or kmp:<domain>:<parentRef>:<childType>:<childRef>", raw)
	}
	domain, err := domainOf(parts[0], parts[1])
	if err != nil {
		return ID{}, fmt.Errorf("invalid identifier %q: %w", raw, err)
	}

	id := ID{Domain: domain}
	var encoded string
	switch len(parts) {
	case 3:
		encoded = parts[2]
	case 4:
		id.Type, encoded = parts[2], parts[3]
	case 5:
		if id.ParentReference, err = decodeReference(parts[2]); err != nil {
			return ID{}, fmt.Errorf("invalid identifier %q: parentRef: %w", raw, err)
		}
		id.Type, encoded = parts[3], parts[4]
		if id.Type == "" {
			return ID{}, fmt.Errorf("invalid identifier %q: childType may not be blank", raw)
		}
	}
	if id.Type != "" && !typePattern.MatchString(id.Type) {
		return ID{}, fmt.Errorf("invalid identifier %q: type must only contain lowercase letters a-z, got %q", raw, id.Type)
	}
	if id.Reference, err = decodeReference(encoded); err != nil {
		return ID{}, fmt.Errorf("invalid identifier %q: %w", raw, err)
	}
	return id, nil
}

func domainOf(namespace, name string) (Domain, error) {
	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("namespace and domain must be non-empty")
	}
	if !strings.EqualFold(namespace, Namespace) {
		return "", fmt.Errorf("no domain for namespace %s and id %s", namespace, name)
	}
	for _, d := range domains {
		if strings.EqualFold(string(d), name) {
			return d, nil
		}
	}
	return "", fmt.Errorf("no domain for namespace %s and id %s", namespace, name)
}

// decodeReference follows KompassReference.parse: 22 characters that decode to
// 16 bytes are a compressed UUID, anything else is base64url-encoded UTF-8.
// String references may carry one "=" so they never look like a UUID.
func decodeReference(encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", fmt.Errorf("reference may not be blank")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(encoded, "="))
	if err != nil {
		return "", fmt.Errorf("reference %q is not base64url: %w", encoded, err)
	}
	if len(encoded) == 22 && len(b) == 16 {
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
	}
	return string(b), nil
}
