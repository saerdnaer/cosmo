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

func TestQueryNode(t *testing.T) {
	svc := &NodeService{}

	t.Run("article", func(t *testing.T) {
		resp, err := svc.QueryNode(context.Background(), &service.QueryNodeRequest{Id: "kmp:cms:article:1"})
		require.NoError(t, err)
		assert.Equal(t, "kmp:cms:article:1", resp.GetNode().GetArticle().GetId())
	})

	t.Run("comment", func(t *testing.T) {
		resp, err := svc.QueryNode(context.Background(), &service.QueryNodeRequest{Id: "kmp:comments:comment:7"})
		require.NoError(t, err)
		assert.Equal(t, "kmp:comments:comment:7", resp.GetNode().GetComment().GetId())
	})

	t.Run("unknown type resolves to null", func(t *testing.T) {
		resp, err := svc.QueryNode(context.Background(), &service.QueryNodeRequest{Id: "kmp:epg:broadcast:1"})
		require.NoError(t, err)
		assert.Nil(t, resp.GetNode())
	})

	t.Run("malformed id is rejected", func(t *testing.T) {
		_, err := svc.QueryNode(context.Background(), &service.QueryNodeRequest{Id: "kmp:cms"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
