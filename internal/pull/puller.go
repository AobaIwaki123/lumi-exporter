// Package pull fetches registered lumi services and normalizes their payloads.
package pull

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AobaIwaki123/lumi-exporter/internal/registry"
	"github.com/AobaIwaki123/lumi-spec/entity"
	"github.com/AobaIwaki123/lumi-spec/event"
)

const MaxResponseBytes = 5 * 1024 * 1024

type Result struct {
	Source     registry.Source   `json:"source"`
	PulledAt   string            `json:"pulled_at"`
	StatusCode int               `json:"status_code"`
	DurationMS int64             `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
	Entity     *entity.Record    `json:"entity,omitempty"`
	Entities   []entity.Record   `json:"entities,omitempty"`
	Events     []event.Candidate `json:"events,omitempty"`
	Raw        json.RawMessage   `json:"raw,omitempty"`
}

type Client struct {
	HTTPClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{HTTPClient: &http.Client{Timeout: timeout}}
}

func (c *Client) Pull(ctx context.Context, source registry.Source) Result {
	started := time.Now()
	result := Result{Source: source, PulledAt: started.UTC().Format(time.RFC3339Nano)}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.Endpoint, nil)
	if err != nil {
		return result.withError(err, started)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return result.withError(err, started)
	}
	defer func() { _ = response.Body.Close() }()
	result.StatusCode = response.StatusCode
	body, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBytes+1))
	if err != nil {
		return result.withError(err, started)
	}
	if len(body) > MaxResponseBytes {
		return result.withError(fmt.Errorf("response exceeded %d bytes", MaxResponseBytes), started)
	}
	result.Raw = rawJSON(body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return result.withError(fmt.Errorf("upstream returned HTTP %d", response.StatusCode), started)
	}
	if err := decode(source, body, &result); err != nil {
		return result.withError(err, started)
	}
	result.DurationMS = time.Since(started).Milliseconds()
	return result
}

func (r Result) withError(err error, started time.Time) Result {
	r.Error = err.Error()
	r.DurationMS = time.Since(started).Milliseconds()
	return r
}

func PullAll(ctx context.Context, client *Client, config registry.Config) []Result {
	results := make([]Result, len(config.Sources))
	var waitGroup sync.WaitGroup
	for index, source := range config.Sources {
		if !source.Enabled {
			results[index] = Result{Source: source, PulledAt: time.Now().UTC().Format(time.RFC3339Nano), Error: "source is disabled"}
			continue
		}
		waitGroup.Add(1)
		go func(index int, source registry.Source) {
			defer waitGroup.Done()
			results[index] = client.Pull(ctx, source)
		}(index, source)
	}
	waitGroup.Wait()
	return results
}

func decode(source registry.Source, body []byte, result *Result) error {
	switch source.Kind {
	case registry.KindEntity:
		var value entity.Record
		if err := json.Unmarshal(body, &value); err != nil {
			return fmt.Errorf("decode entity: %w", err)
		}
		if value.SchemaVersion != entity.SchemaVersion {
			return fmt.Errorf("unsupported entity schema version %q", value.SchemaVersion)
		}
		result.Entity = &value
	case registry.KindEntityFeed:
		var value entity.Feed
		if err := json.Unmarshal(body, &value); err != nil {
			return fmt.Errorf("decode entity feed: %w", err)
		}
		if value.SchemaVersion != entity.FeedSchemaVersion {
			return fmt.Errorf("unsupported entity feed schema version %q", value.SchemaVersion)
		}
		result.Entities = value.Entities
	case registry.KindEventFeed:
		var value event.Feed
		if err := json.Unmarshal(body, &value); err != nil {
			return fmt.Errorf("decode event feed: %w", err)
		}
		if value.SchemaVersion != event.FeedSchemaVersion {
			return fmt.Errorf("unsupported event feed schema version %q", value.SchemaVersion)
		}
		result.Events = value.Events
	case registry.KindLumitreeCalendar:
		return decodeLumitree(source, body, result)
	case registry.KindLumidiveEvent:
		return decodeLumidive(source, body, result)
	}
	return nil
}

type lumitreeCalendarResponse struct {
	Calendar struct {
		AliasCode string `json:"aliasCode"`
		ID        string `json:"id"`
		Title     string `json:"title"`
	} `json:"calendar"`
	Events []lumitreeEvent `json:"events"`
}

type lumitreeEvent struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	StartAt  string         `json:"startAt"`
	EndAt    string         `json:"endAt"`
	Timezone string         `json:"timezone"`
	Location *string        `json:"location"`
	URL      *string        `json:"url"`
	Links    []lumitreeLink `json:"links"`
}

type lumitreeLink struct {
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Label  string `json:"label"`
	Origin string `json:"origin"`
}

func decodeLumitree(source registry.Source, body []byte, result *Result) error {
	var value lumitreeCalendarResponse
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode lumitree response: %w", err)
	}
	collectionID := value.Calendar.AliasCode
	if collectionID == "" {
		collectionID = value.Calendar.ID
	}
	result.Events = make([]event.Candidate, 0, len(value.Events))
	for index, item := range value.Events {
		id := item.ID
		if id == "" {
			id = fmt.Sprintf("%s-%d", source.ID, index)
		}
		links := make([]event.Link, 0, len(item.Links))
		for _, link := range item.Links {
			if strings.TrimSpace(link.URL) == "" {
				continue
			}
			links = append(links, event.Link{Kind: link.Kind, URL: link.URL, Label: link.Label, Origin: link.Origin})
		}
		result.Events = append(result.Events, event.Candidate{
			SchemaVersion: event.CandidateSchemaVersion,
			ID:            source.Service + ":" + id,
			SourceRef: event.SourceRef{
				Source:       source.Service,
				RecordID:     id,
				URL:          stringPointer(item.URL),
				CollectionID: collectionID,
			},
			Title:     item.Title,
			LocalDate: localDate(item.StartAt),
			StartsAt:  item.StartAt,
			EndsAt:    item.EndAt,
			Timezone:  item.Timezone,
			Venue:     stringPointer(item.Location),
			Links:     links,
		})
	}
	return nil
}

type lumidiveEventResponse struct {
	Data struct {
		ID       string           `json:"id"`
		Name     string           `json:"name"`
		URL      string           `json:"url"`
		ShortURL string           `json:"shortUrl"`
		Stages   []lumidiveStage  `json:"stages"`
		Artists  []lumidiveArtist `json:"artists"`
	} `json:"data"`
}

type lumidiveStage struct {
	StartAt   *string `json:"startAt"`
	EndAt     *string `json:"endAt"`
	StageName string  `json:"stageName"`
	Venue     *struct {
		Name *string `json:"name"`
	} `json:"venue"`
	Artists []lumidiveArtist `json:"artists"`
}

type lumidiveArtist struct {
	Name string `json:"name"`
}

func decodeLumidive(source registry.Source, body []byte, result *Result) error {
	var value lumidiveEventResponse
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("decode lumidive response: %w", err)
	}
	if value.Data.ID == "" && value.Data.Name == "" {
		return fmt.Errorf("lumidive response has no event data")
	}
	startAt, endAt, venue, performers := "", "", "", value.Data.Artists
	if len(value.Data.Stages) > 0 {
		stage := value.Data.Stages[0]
		startAt = stringPointer(stage.StartAt)
		endAt = stringPointer(stage.EndAt)
		if stage.Venue != nil {
			venue = stringPointer(stage.Venue.Name)
		}
		if len(stage.Artists) > 0 {
			performers = stage.Artists
		}
	}
	if value.Data.URL == "" {
		value.Data.URL = value.Data.ShortURL
	}
	id := value.Data.ID
	if id == "" {
		id = source.ID
	}
	performerNames := make([]string, 0, len(performers))
	for _, performer := range performers {
		if strings.TrimSpace(performer.Name) != "" {
			performerNames = append(performerNames, performer.Name)
		}
	}
	links := make([]event.Link, 0, 2)
	seenURLs := make(map[string]struct{}, 2)
	for _, ticketURL := range []string{value.Data.URL, value.Data.ShortURL} {
		ticketURL = strings.TrimSpace(ticketURL)
		if ticketURL == "" {
			continue
		}
		if _, seen := seenURLs[ticketURL]; seen {
			continue
		}
		seenURLs[ticketURL] = struct{}{}
		links = append(links, event.Link{Kind: event.LinkKindTicket, URL: ticketURL, Label: "TicketDive", Origin: "structured_url"})
	}
	result.Events = []event.Candidate{{
		SchemaVersion: event.CandidateSchemaVersion,
		ID:            source.Service + ":" + id,
		SourceRef: event.SourceRef{
			Source:   source.Service,
			RecordID: id,
			URL:      value.Data.URL,
		},
		Title:      value.Data.Name,
		LocalDate:  localDate(startAt),
		StartsAt:   startAt,
		EndsAt:     endAt,
		Venue:      venue,
		Performers: performerNames,
		Links:      links,
	}}
	return nil
}

func localDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func stringPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func rawJSON(body []byte) json.RawMessage {
	if json.Valid(body) {
		return append(json.RawMessage(nil), body...)
	}
	encoded, _ := json.Marshal(string(body))
	return encoded
}
