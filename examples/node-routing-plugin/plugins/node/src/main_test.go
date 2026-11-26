package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	service "github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	articleID   = "kmp:cms:mnsoYQFYSOK2EJdaYa7VGQ"
	cmsTVID     = "kmp:cms:jLPeT7neT0OBx1d-5ZlLDA"
	commentID   = "kmp:comments:d6ZfvSNnRgCkwa69rhYUZQ"
	pocTVID     = "poc:tv:28106"
	pocRadioID  = "poc:radio:28402"
	unknownCMS  = "kmp:cms:AAAAAAAAAAAAAAAAAAAAAQ"
	otherDomain = "kmp:event:YWJj"
)

// newService routes like routing.yaml, with in-memory subgraph lookups.
func newService() *NodeService {
	fake := func(types map[string]string) TypeLookup {
		return func(_ context.Context, id string) (string, error) { return types[id], nil }
	}
	return &NodeService{resolver: &Resolver{Routes: map[string]TypeLookup{
		"kmp:cms":      fake(map[string]string{articleID: "Article", cmsTVID: "PublicationServiceCmsTV"}),
		"kmp:comments": fake(map[string]string{commentID: "Comment"}),
		"poc":          fake(map[string]string{pocTVID: "PublicationServiceTV", pocRadioID: "PublicationServiceRadio"}),
	}}}
}

func TestQueryNode(t *testing.T) {
	svc := newService()
	tests := map[string]func(*service.Node) bool{
		articleID:   func(n *service.Node) bool { return n.GetArticle().GetId() == articleID },
		commentID:   func(n *service.Node) bool { return n.GetComment().GetId() == commentID },
		cmsTVID:     func(n *service.Node) bool { return n.GetPublicationServiceCmsTv().GetId() == cmsTVID },
		pocTVID:     func(n *service.Node) bool { return n.GetPublicationServiceTv().GetId() == pocTVID },
		pocRadioID:  func(n *service.Node) bool { return n.GetPublicationServiceRadio().GetId() == pocRadioID },
		unknownCMS:  func(n *service.Node) bool { return n == nil },
		otherDomain: func(n *service.Node) bool { return n == nil },
	}
	for id, ok := range tests {
		t.Run(id, func(t *testing.T) {
			resp, err := svc.QueryNode(context.Background(), &service.QueryNodeRequest{Id: id})
			require.NoError(t, err)
			assert.True(t, ok(resp.GetNode()), "got %v", resp.GetNode())
		})
	}
}

func TestQueryPublicationService(t *testing.T) {
	svc := newService()
	tests := map[string]func(*service.PublicationService) bool{
		cmsTVID:    func(p *service.PublicationService) bool { return p.GetPublicationServiceCmsTv().GetId() == cmsTVID },
		pocTVID:    func(p *service.PublicationService) bool { return p.GetPublicationServiceTv().GetId() == pocTVID },
		pocRadioID: func(p *service.PublicationService) bool { return p.GetPublicationServiceRadio().GetId() == pocRadioID },
		// Valid IDs of other types are not publication services.
		articleID: func(p *service.PublicationService) bool { return p == nil },
		commentID: func(p *service.PublicationService) bool { return p == nil },
	}
	for id, ok := range tests {
		t.Run(id, func(t *testing.T) {
			resp, err := svc.QueryPublicationService(context.Background(), &service.QueryPublicationServiceRequest{Id: id})
			require.NoError(t, err)
			assert.True(t, ok(resp.GetPublicationService()), "got %v", resp.GetPublicationService())
		})
	}
}

func TestInvalidID(t *testing.T) {
	for _, id := range []string{"kmp:cms", "kmp:nope:YWJj", "nope:tv:1", "noprefix", "kmp:comments:Bad:YWJj"} {
		_, err := newService().QueryNode(context.Background(), &service.QueryNodeRequest{Id: id})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), id)
	}
}
