package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	service "github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node/generated"
	routerplugin "github.com/wundergraph/cosmo/router-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	pl, err := routerplugin.NewRouterPlugin(func(s *grpc.Server) {
		s.RegisterService(&service.NodeService_ServiceDesc, &NodeService{})
	}, routerplugin.WithTracing())
	if err != nil {
		log.Fatalf("failed to create router plugin: %v", err)
	}

	pl.Serve()
}

// stubFactories maps "<domain>:<type>" of a global ID to a constructor for the
// matching Node implementation. The plugin only returns the ID; the router
// fetches all other selected fields from the subgraph that owns the type.
var stubFactories = map[string]func(id string) *service.Node{
	"cms:article": func(id string) *service.Node {
		return &service.Node{Instance: &service.Node_Article{Article: &service.Article{Id: id}}}
	},
	"comments:comment": func(id string) *service.Node {
		return &service.Node{Instance: &service.Node_Comment{Comment: &service.Comment{Id: id}}}
	},
}

// NodeService resolves Query.node by routing on the global ID.
type NodeService struct {
	service.UnimplementedNodeServiceServer
}

func (s *NodeService) QueryNode(_ context.Context, req *service.QueryNodeRequest) (*service.QueryNodeResponse, error) {
	node, err := stubFor(req.GetId())
	if err != nil {
		return nil, err
	}
	return &service.QueryNodeResponse{Node: node}, nil
}

// stubFor parses an ID of the form "<prefix>:<domain>:<type>:<reference>".
// Unknown domains/types resolve to null, as the Relay spec allows for IDs
// that do not identify an object.
func stubFor(id string) (*service.Node, error) {
	parts := strings.SplitN(id, ":", 4)
	if len(parts) != 4 || parts[3] == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid global ID %q: want <prefix>:<domain>:<type>:<reference>", id))
	}
	factory, ok := stubFactories[parts[1]+":"+parts[2]]
	if !ok {
		return nil, nil
	}
	return factory(id), nil
}
