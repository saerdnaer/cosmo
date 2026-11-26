package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Config is the plugin's routing table, see routing.yaml.
type Config struct {
	Routes map[string]RouteConfig `yaml:"routes"`
}

type RouteConfig struct {
	// Subgraph is the name of the owning subgraph in the graph.
	Subgraph string `yaml:"subgraph"`
	// URL overrides the subgraph's routing URL.
	URL string `yaml:"url"`
	// Resolver is the lookup field; defaults to node<Domain>.
	Resolver string `yaml:"resolver"`
}

// configPath returns $NODE_ROUTING_CONFIG, or routing.yaml in the plugin
// directory (the binary lives in <plugin>/bin/).
func configPath() (string, error) {
	if p := os.Getenv("NODE_ROUTING_CONFIG"); p != "" {
		return p, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "..", "routing.yaml"), nil
}

// LoadConfig reads a routing table, expands ${VAR} and ${VAR:-default}, and
// resolves each route's URL: its own `url`, else the routing URL the router
// reported for its `subgraph`.
func LoadConfig(path string, subgraphURLs map[string]string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read routing config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte(expandEnv(string(raw))), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse routing config %s: %w", path, err)
	}
	if len(cfg.Routes) == 0 {
		return Config{}, fmt.Errorf("routing config %s has no routes", path)
	}
	for prefix, r := range cfg.Routes {
		if r.URL == "" {
			if r.Subgraph == "" {
				return Config{}, fmt.Errorf("route %q needs a subgraph or a url", prefix)
			}
			u, ok := subgraphURLs[r.Subgraph]
			if !ok {
				return Config{}, fmt.Errorf("route %q: subgraph %q is not part of the graph", prefix, r.Subgraph)
			}
			r.URL = u
		}
		if parsed, err := url.Parse(r.URL); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return Config{}, fmt.Errorf("route %q: %q is not an HTTP GraphQL endpoint", prefix, r.URL)
		}
		if r.Resolver == "" {
			r.Resolver = defaultResolver(prefix)
		}
		cfg.Routes[prefix] = r
	}
	return cfg, nil
}

// defaultResolver turns kmp:cms into nodeCms and poc into nodePoc.
func defaultResolver(prefix string) string {
	parts := strings.Split(prefix, ":")
	last := []rune(parts[len(parts)-1])
	last[0] = unicode.ToUpper(last[0])
	return "node" + string(last)
}

// SubgraphURLsFromStartupConfig reads the subgraph routing URLs the router
// passes to plugins in the startup_config environment variable.
func SubgraphURLsFromStartupConfig(raw string) (map[string]string, error) {
	urls := map[string]string{}
	if raw == "" {
		return urls, nil
	}
	var startup struct {
		Subgraphs []struct {
			Name       string `json:"name"`
			RoutingURL string `json:"routing_url"`
		} `json:"subgraphs"`
	}
	if err := json.Unmarshal([]byte(raw), &startup); err != nil {
		return nil, fmt.Errorf("parse startup_config: %w", err)
	}
	for _, sg := range startup.Subgraphs {
		urls[sg.Name] = sg.RoutingURL
	}
	return urls, nil
}

var envVar = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

func expandEnv(s string) string {
	return envVar.ReplaceAllStringFunc(s, func(m string) string {
		g := envVar.FindStringSubmatch(m)
		if v, ok := os.LookupEnv(g[1]); ok && v != "" {
			return v
		}
		return g[2]
	})
}
