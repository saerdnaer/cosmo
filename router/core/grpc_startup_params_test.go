package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wundergraph/cosmo/router/pkg/grpcconnector/grpccommon"
	rtrace "github.com/wundergraph/cosmo/router/pkg/trace"
)

func TestNewGRPCStartupParamsSubgraphs(t *testing.T) {
	t.Parallel()

	params := newGRPCStartupParams(&rtrace.Config{}, nil, []Subgraph{
		{Name: "cms-dgs", UrlString: "http://cms-dgs:8080/graphql"},
		{Name: "sport-dgs", UrlString: "dns:///sport-dgs:9090"},
		{Name: "node"}, // plugins have no routing URL
	})

	require.Equal(t, []grpccommon.GRPCSubgraph{
		{Name: "cms-dgs", RoutingURL: "http://cms-dgs:8080/graphql"},
		{Name: "sport-dgs", RoutingURL: "dns:///sport-dgs:9090"},
	}, params.Subgraphs)
}

func TestNewGRPCStartupParamsWithoutSubgraphs(t *testing.T) {
	t.Parallel()

	params := newGRPCStartupParams(&rtrace.Config{}, nil, nil)
	require.Nil(t, params.Subgraphs)
}
