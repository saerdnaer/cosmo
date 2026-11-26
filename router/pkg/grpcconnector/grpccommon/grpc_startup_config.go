package grpccommon

import (
	"time"
)

type GRPCStartupParams struct {
	Telemetry       *GRPCTelemetry       `json:"telemetry,omitempty"`
	IPAnonymization *GRPCIPAnonymization `json:"ip_anonymization,omitempty"`
	// Subgraphs lists the subgraphs of the graph the plugin is part of, with
	// their routing URLs after router overrides. Plugins can use it to call
	// other subgraphs by name instead of hard-coding URLs.
	Subgraphs []GRPCSubgraph `json:"subgraphs,omitempty"`
}

type GRPCSubgraph struct {
	Name       string `json:"name"`
	RoutingURL string `json:"routing_url"`
}
type GRPCTelemetry struct {
	Tracing *GRPCTracing `json:"tracing,omitempty"`
}
type GRPCTracing struct {
	Exporters   []GRPCExporter `json:"exporters,omitempty"`
	Propagators []string       `json:"propagators,omitempty"`
	Sampler     float64        `json:"sampler"`
}
type GRPCExporter struct {
	Endpoint      string            `json:"endpoint"`
	Exporter      string            `json:"exporter"`
	BatchTimeout  time.Duration     `json:"batch_timeout"`
	ExportTimeout time.Duration     `json:"export_timeout"`
	Headers       map[string]string `json:"headers"`
	HTTPPath      string            `json:"http_path"`
}

type GRPCIPAnonymization struct {
	Enabled bool   `json:"enabled"`
	Method  string `json:"method"`
}
