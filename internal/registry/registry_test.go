package registry

import (
	"strings"
	"testing"
)

func TestLoadAndFind(t *testing.T) {
	config, err := Load(strings.NewReader(`{"sources":[{"id":"source-a","name":"A","service":"lumitlink","kind":"entity","endpoint":"https://example.test/entity","enabled":true}]}`))
	if err != nil {
		t.Fatal(err)
	}
	source, ok := config.Find("source-a")
	if !ok || source.Service != "lumitlink" {
		t.Fatalf("source = %#v, found = %v", source, ok)
	}
}

func TestLoadRejectsInvalidEndpoint(t *testing.T) {
	_, err := Load(strings.NewReader(`{"sources":[{"id":"source-a","name":"A","service":"lumitlink","kind":"entity","endpoint":"file:///tmp/source","enabled":true}]}`))
	if err == nil {
		t.Fatal("expected invalid endpoint error")
	}
}
