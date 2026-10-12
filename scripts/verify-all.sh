#!/usr/bin/env bash
set -euo pipefail

project_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$project_dir"
go_cache="${GOCACHE:-/tmp/lumi-exporter-go-cache}"

unformatted="$(gofmt -l cmd internal)"
if [[ -n "$unformatted" ]]; then
	printf 'gofmt required:\n%s\n' "$unformatted" >&2
	exit 1
fi

GOCACHE="$go_cache" go vet ./...
GOCACHE="$go_cache" go test -race -cover ./...
GOCACHE="$go_cache" go build -trimpath -o /tmp/lumi-exporter-verify ./cmd/lumi-exporter
GOCACHE="$go_cache" go run ./cmd/lumi-exporter -input testdata/evaluation.json >/tmp/lumi-exporter-analysis.json
test -s /tmp/lumi-exporter-analysis.json
kubectl kustomize k8s/manifests >/tmp/lumi-exporter-k8s.yaml
test -s /tmp/lumi-exporter-k8s.yaml
ruby -ryaml -e 'YAML.load_file("internal/web/assets/openapi.yaml"); YAML.load_file("k8s/manifests/kustomization.yaml")'

printf 'lumi-exporter verification passed\n'
