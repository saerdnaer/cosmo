package kompassid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IDs taken from the Kompass guide and the Java library's tests.
func TestParse(t *testing.T) {
	tests := []struct {
		raw  string
		want ID
	}{
		// A 16-byte string encodes to 22 characters and would read as a UUID,
		// so StringReference pads it with "=" (same as the Java library).
		{"kmp:cms:c29tZS1leHRlcm5hbC1pZA=", ID{Domain: CMS, Reference: "some-external-id"}},
		{"kmp:cms:ZWZiZWI3YjYtNzA0My00MWQ5LWFhYWYtMThkNzczYjc5ODUy", ID{Domain: CMS, Reference: "efbeb7b6-7043-41d9-aaaf-18d773b79852"}},
		{"kmp:cms:mnsoYQFYSOK2EJdaYa7VGQ", ID{Domain: CMS, Reference: "9a7b2861-0158-48e2-b610-975a61aed519"}},
		{"kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ", ID{Domain: Comments, Reference: "77a65fbd-2367-4600-a4c1-aebdae161465"}},
		{"kmp:comments:myType:YWJj", ID{}}, // type must be lowercase: rejected below
		{"kmp:comments:comment:YWJj", ID{Domain: Comments, Type: "comment", Reference: "abc"}},
		{"kmp:cms:wfCl0p87SneNHitvTHqeEA:answer:AAAAAAAAAAAAAAAAAAAAAQ", ID{
			Domain: CMS, Type: "answer",
			ParentReference: "c1f0a5d2-9f3b-4a77-8d1e-2b6f4c7a9e10",
			Reference:       "00000000-0000-0000-0000-000000000001",
		}},
		{"kmp:event:YWJj", ID{Domain: Event, Reference: "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := Parse(tt.raw)
			if tt.want == (ID{}) {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRejects(t *testing.T) {
	for _, raw := range []string{
		"",
		"kmp:cms",
		"kmp:cms:",
		"kmp:unknown:YWJj",
		"foo:cms:YWJj",
		"kmp:cms:a:b:c:d",
		"kmp:cms:not*base64",
		"poc:tv:28106",
	} {
		_, err := Parse(raw)
		assert.Error(t, err, raw)
	}
}
