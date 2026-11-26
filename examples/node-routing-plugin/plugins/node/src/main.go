package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	service "github.com/wundergraph/cosmo/examples/node-routing-plugin/plugins/node/generated"
	routerplugin "github.com/wundergraph/cosmo/router-plugin"
	"github.com/wundergraph/cosmo/router-plugin/httpclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	path, err := configPath()
	if err != nil {
		log.Fatalf("locate routing config: %v", err)
	}
	subgraphURLs, err := SubgraphURLsFromStartupConfig(os.Getenv("startup_config"))
	if err != nil {
		log.Fatalf("%v", err)
	}
	cfg, err := LoadConfig(path, subgraphURLs)
	if err != nil {
		log.Fatalf("%v", err)
	}
	svc := &NodeService{resolver: NewResolver(cfg)}

	pl, err := routerplugin.NewRouterPlugin(func(s *grpc.Server) {
		s.RegisterService(&service.NodeService_ServiceDesc, svc)
	}, routerplugin.WithTracing())
	if err != nil {
		log.Fatalf("failed to create router plugin: %v", err)
	}

	pl.Serve()
}

// NodeService serves every root field that returns an object by ID from
// whichever subgraph owns it. Each field only accepts the types of its return
// interface; IDs of other types resolve to null.
type NodeService struct {
	service.UnimplementedNodeServiceServer
	resolver *Resolver
}

func (s *NodeService) QueryNode(ctx context.Context, req *service.QueryNodeRequest) (*service.QueryNodeResponse, error) {
	typename, err := s.typeOf(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &service.QueryNodeResponse{Node: nodeStub(typename, req.GetId())}, nil
}

func (s *NodeService) QueryPublicationService(ctx context.Context, req *service.QueryPublicationServiceRequest) (*service.QueryPublicationServiceResponse, error) {
	typename, err := s.typeOf(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &service.QueryPublicationServiceResponse{PublicationService: publicationServiceStub(typename, req.GetId())}, nil
}

func (s *NodeService) typeOf(ctx context.Context, id string) (string, error) {
	typename, err := s.resolver.TypeOf(ctx, id)
	var invalid InvalidIDError
	switch {
	case errors.As(err, &invalid):
		return "", status.Error(codes.InvalidArgument, err.Error())
	case err != nil:
		return "", status.Error(codes.Unavailable, err.Error())
	}
	return typename, nil
}

func nodeStub(typename, id string) *service.Node {
	switch typename {
	case "Article":
		return &service.Node{Instance: &service.Node_Article{Article: &service.Article{Id: id}}}
	case "Comment":
		return &service.Node{Instance: &service.Node_Comment{Comment: &service.Comment{Id: id}}}
	case "PublicationServiceCmsTV":
		return &service.Node{Instance: &service.Node_PublicationServiceCmsTv{PublicationServiceCmsTv: &service.PublicationServiceCmsTV{Id: id}}}
	case "PublicationServiceTV":
		return &service.Node{Instance: &service.Node_PublicationServiceTv{PublicationServiceTv: &service.PublicationServiceTV{Id: id}}}
	case "PublicationServiceRadio":
		return &service.Node{Instance: &service.Node_PublicationServiceRadio{PublicationServiceRadio: &service.PublicationServiceRadio{Id: id}}}
	}
	return nil
}

func publicationServiceStub(typename, id string) *service.PublicationService {
	switch typename {
	case "PublicationServiceCmsTV":
		return &service.PublicationService{Instance: &service.PublicationService_PublicationServiceCmsTv{PublicationServiceCmsTv: &service.PublicationServiceCmsTV{Id: id}}}
	case "PublicationServiceTV":
		return &service.PublicationService{Instance: &service.PublicationService_PublicationServiceTv{PublicationServiceTv: &service.PublicationServiceTV{Id: id}}}
	case "PublicationServiceRadio":
		return &service.PublicationService{Instance: &service.PublicationService_PublicationServiceRadio{PublicationServiceRadio: &service.PublicationServiceRadio{Id: id}}}
	}
	return nil
}

// subgraphTypeLookup asks a subgraph directly (not through the router) for
// the __typename behind an ID, using a domain-specific lookup field such as
// `nodeCms(id: ID!): Node`.
func subgraphTypeLookup(url, field string) TypeLookup {
	client := httpclient.New(httpclient.WithBaseURL(url), httpclient.WithTimeout(5*time.Second))
	query := "query($id: ID!) { " + field + "(id: $id) { __typename } }"

	return func(ctx context.Context, id string) (string, error) {
		resp, err := client.Post(ctx, "", map[string]any{"query": query, "variables": map[string]any{"id": id}})
		if err != nil {
			return "", err
		}
		if !resp.IsSuccess() {
			return "", fmt.Errorf("type lookup failed with HTTP %d", resp.StatusCode)
		}
		body, err := httpclient.UnmarshalTo[struct {
			Data map[string]*struct {
				Typename string `json:"__typename"`
			} `json:"data"`
			Errors []struct{ Message string } `json:"errors"`
		}](resp)
		if err != nil {
			return "", err
		}
		if len(body.Errors) > 0 {
			return "", errors.New("type lookup failed: " + body.Errors[0].Message)
		}
		if node := body.Data[field]; node != nil {
			return node.Typename, nil
		}
		return "", nil
	}
}
