// Package registry defines the pull sources registered with lumi-exporter.
package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

type Kind string

const (
	KindEntity           Kind = "entity"
	KindEntityFeed       Kind = "entity_feed"
	KindEventFeed        Kind = "event_feed"
	KindLumitreeCalendar Kind = "lumitree_calendar"
	KindLumidiveEvent    Kind = "lumidive_event"
)

// Source is an intentionally small registration record. Secrets and
// provider-specific options stay outside the shared lumi-spec contract.
type Source struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Service  string `json:"service"`
	Kind     Kind   `json:"kind"`
	Endpoint string `json:"endpoint"`
	Enabled  bool   `json:"enabled"`
}

type Config struct {
	Sources []Source `json:"sources"`
}

func Load(reader io.Reader) (Config, error) {
	var config Config
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode source registry: %w", err)
	}
	if len(config.Sources) == 0 {
		return Config{}, fmt.Errorf("source registry has no sources")
	}
	seen := make(map[string]struct{}, len(config.Sources))
	for index, source := range config.Sources {
		if err := validate(source); err != nil {
			return Config{}, fmt.Errorf("source %d: %w", index, err)
		}
		if _, ok := seen[source.ID]; ok {
			return Config{}, fmt.Errorf("source %q is duplicated", source.ID)
		}
		seen[source.ID] = struct{}{}
	}
	return config, nil
}

func validate(source Source) error {
	if strings.TrimSpace(source.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(source.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(source.Service) == "" {
		return fmt.Errorf("service is required")
	}
	switch source.Kind {
	case KindEntity, KindEntityFeed, KindEventFeed, KindLumitreeCalendar, KindLumidiveEvent:
	default:
		return fmt.Errorf("unsupported kind %q", source.Kind)
	}
	parsed, err := url.Parse(source.Endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("endpoint must be an absolute http(s) URL")
	}
	return nil
}

func (c Config) Find(id string) (Source, bool) {
	for _, source := range c.Sources {
		if source.ID == id {
			return source, true
		}
	}
	return Source{}, false
}
