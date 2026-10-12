package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AobaIwaki123/lumi-exporter/internal/pull"
	"github.com/AobaIwaki123/lumi-exporter/internal/registry"
	"github.com/AobaIwaki123/lumi-spec/entity"
	"github.com/AobaIwaki123/lumi-spec/event"
)

func TestServerRendersCatalogFromPullResult(t *testing.T) {
	body, _ := json.Marshal(entity.Record{
		SchemaVersion: entity.SchemaVersion,
		ID:            "idol-1",
		Kind:          entity.KindIdol,
		Name:          "雨宮サク",
		Assets:        []entity.Asset{{Kind: entity.AssetIcon, URL: "https://example.test/icon.jpg"}},
		SourceRefs:    []event.SourceRef{{Source: "fixture", RecordID: "idol-1"}},
	})
	client := &pull.Client{HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
	})}}

	server := NewServer(registry.Config{Sources: []registry.Source{{
		ID: "fixture", Name: "fixture source", Service: "fixture", Kind: registry.KindEntity, Endpoint: "https://fixture.test/entity", Enabled: true,
	}}}, client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var catalog Catalog
	if err := json.Unmarshal(recorder.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Results) != 1 || catalog.Results[0].Entity == nil || catalog.Results[0].Entity.Name != "雨宮サク" {
		t.Fatalf("catalog = %#v", catalog)
	}

	uiRecorder := httptest.NewRecorder()
	server.ServeHTTP(uiRecorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if uiRecorder.Code != http.StatusOK || !strings.Contains(uiRecorder.Body.String(), "lumi-exporter catalog") {
		t.Fatalf("ui status = %d", uiRecorder.Code)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
