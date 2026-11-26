package main

import (
	"fmt"
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
	// URL of the subgraph's GraphQL endpoint, called directly (not via the router).
	URL string `yaml:"url"`
	// Field is the lookup field; defaults to node<Domain>.
	Field string `yaml:"field"`
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

// LoadConfig reads a routing table and expands ${VAR} and ${VAR:-default}.
func LoadConfig(path string) (Config, error) {
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
			return Config{}, fmt.Errorf("route %q has no url", prefix)
		}
		if r.Field == "" {
			r.Field = defaultField(prefix)
			cfg.Routes[prefix] = r
		}
	}
	return cfg, nil
}

// defaultField turns kmp:cms into nodeCms and poc into nodePoc.
func defaultField(prefix string) string {
	parts := strings.Split(prefix, ":")
	last := []rune(parts[len(parts)-1])
	last[0] = unicode.ToUpper(last[0])
	return "node" + string(last)
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
