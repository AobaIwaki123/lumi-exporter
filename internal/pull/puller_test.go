package pull

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AobaIwaki123/lumi-exporter/internal/registry"
	"github.com/AobaIwaki123/lumi-spec/entity"
	"github.com/AobaIwaki123/lumi-spec/event"
)

func TestPullNormalizesRegisteredFormats(t *testing.T) {
	entityBody, _ := json.Marshal(entity.Record{
		SchemaVersion: entity.SchemaVersion,
		ID:            "idol-1",
		Kind:          entity.KindIdol,
		Name:          "雨宮サク",
		SourceRefs:    []event.SourceRef{{Source: "lumitlink", RecordID: "amamiyasaku"}},
	})
	feedBody, _ := json.Marshal(event.Feed{
		SchemaVersion: event.FeedSchemaVersion,
		Source:        "fixture",
		Events: []event.Candidate{{
			SchemaVersion: event.CandidateSchemaVersion,
			ID:            "event-1",
			SourceRef:     event.SourceRef{Source: "fixture", RecordID: "event-1"},
			Title:         "テストライブ",
			LocalDate:     "2026-10-12",
		}},
	})
	responses := map[string]string{
		"/entity":   string(entityBody),
		"/feed":     string(feedBody),
		"/lumitree": `{"calendar":{"aliasCode":"idol-calendar","title":"アイドル","id":"calendar-1"},"events":[{"id":"tt-1","title":"秋ライブ","startAt":"2026-10-12T18:00:00+09:00","endAt":"2026-10-12T20:00:00+09:00","timezone":"Asia/Tokyo","location":"渋谷","url":"https://example.test/tt-1"}]}`,
		"/lumidive": `{"success":true,"data":{"id":"td-1","name":"冬ライブ","url":"https://ticketdive.com/event/td-1","stages":[{"startAt":"2026-12-01T18:00:00+09:00","endAt":"2026-12-01T20:00:00+09:00","venue":{"name":"新宿"},"artists":[{"name":"A"},{"name":"B"}]}]}}`,
	}
	client := &Client{HTTPClient: &http.Client{Timeout: 2 * time.Second, Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, ok := responses[request.URL.Path]
		status := http.StatusOK
		if !ok {
			status = http.StatusNotFound
			body = `{"error":"not found"}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})}}

	cases := []struct {
		name  string
		path  string
		kind  registry.Kind
		check func(*testing.T, Result)
	}{
		{name: "entity", path: "/entity", kind: registry.KindEntity, check: func(t *testing.T, result Result) {
			if result.Error != "" || result.Entity == nil || result.Entity.Name != "雨宮サク" {
				t.Fatalf("result = %#v", result)
			}
		}},
		{name: "feed", path: "/feed", kind: registry.KindEventFeed, check: func(t *testing.T, result Result) {
			if result.Error != "" || len(result.Events) != 1 || result.Events[0].Title != "テストライブ" {
				t.Fatalf("result = %#v", result)
			}
		}},
		{name: "lumitree", path: "/lumitree", kind: registry.KindLumitreeCalendar, check: func(t *testing.T, result Result) {
			if result.Error != "" || len(result.Events) != 1 || result.Events[0].LocalDate != "2026-10-12" || result.Events[0].Venue != "渋谷" {
				t.Fatalf("result = %#v", result)
			}
		}},
		{name: "lumidive", path: "/lumidive", kind: registry.KindLumidiveEvent, check: func(t *testing.T, result Result) {
			if result.Error != "" || len(result.Events) != 1 || len(result.Events[0].Performers) != 2 || result.Events[0].Venue != "新宿" {
				t.Fatalf("result = %#v", result)
			}
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := client.Pull(context.Background(), registry.Source{
				ID:       testCase.name,
				Name:     testCase.name,
				Service:  testCase.name,
				Kind:     testCase.kind,
				Endpoint: "https://fixture.test" + testCase.path,
				Enabled:  true,
			})
			testCase.check(t, result)
			if len(result.Raw) == 0 {
				t.Fatal("raw response is empty")
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
