SHELL := /bin/sh

IMAGE ?= ghcr.io/aobaiwaki123/lumi-exporter
TAG ?= dev

.PHONY: test verify build run web docker-build docker-push k8s-render k8s-apply

test:
	go test ./...

verify:
	./scripts/verify-all.sh

build:
	go build -trimpath -o /tmp/lumi-exporter ./cmd/lumi-exporter

run:
	go run ./cmd/lumi-exporter

web:
	go run ./cmd/lumi-exporter -serve

docker-build:
	docker build --tag $(IMAGE):$(TAG) .

docker-push:
	docker push $(IMAGE):$(TAG)

k8s-render:
	kubectl kustomize k8s/manifests

k8s-apply:
	kubectl apply -k k8s/manifests
