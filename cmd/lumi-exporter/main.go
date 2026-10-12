// Package main provides the evaluation CLI and the thin pull-and-render web service.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AobaIwaki123/lumi-exporter/internal/matching"
	"github.com/AobaIwaki123/lumi-exporter/internal/pull"
	"github.com/AobaIwaki123/lumi-exporter/internal/registry"
	"github.com/AobaIwaki123/lumi-exporter/internal/web"
)

func main() {
	serve := flag.Bool("serve", false, "pull型Webサービスを起動する")
	inputPath := flag.String("input", "testdata/evaluation.json", "評価fixtureのJSONパス")
	pretty := flag.Bool("pretty", true, "JSONをインデントして出力する")
	registryPath := flag.String("registry", envString("LUMI_EXPORTER_REGISTRY", "config/sources.json"), "pull対象registryのJSONパス")
	host := flag.String("host", envString("LUMI_EXPORTER_HOST", "0.0.0.0"), "Webサービスの待受アドレス")
	port := flag.Int("port", envInt("LUMI_EXPORTER_PORT", 8081), "Webサービスの待受ポート")
	timeout := flag.Duration("upstream-timeout", envDuration("LUMI_EXPORTER_UPSTREAM_TIMEOUT", 10*time.Second), "pull先HTTPのタイムアウト")
	flag.Parse()

	var err error
	if *serve {
		err = serveWeb(*registryPath, *host, *port, *timeout)
	} else {
		err = run(*inputPath, *pretty)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveWeb(registryPath, host string, port int, timeout time.Duration) error {
	file, err := os.Open(registryPath)
	if err != nil {
		return fmt.Errorf("open source registry: %w", err)
	}
	defer func() { _ = file.Close() }()
	config, err := registry.Load(file)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	application := web.NewServer(config, pull.NewClient(timeout), logger)
	server := &http.Server{
		Addr:              net.JoinHostPort(host, strconv.Itoa(port)),
		Handler:           application,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(ctx); shutdownErr != nil {
			logger.Error("server shutdown failed", "error", shutdownErr)
		}
	}()
	logger.Info("server listening", "service", "lumi-exporter", "addr", server.Addr, "sources", len(config.Sources))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

func run(inputPath string, pretty bool) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer func() { _ = file.Close() }()

	fixture, err := matching.LoadFixture(file)
	if err != nil {
		return err
	}
	report := matching.Evaluate(fixture, matching.DefaultConfig())

	encoder := json.NewEncoder(os.Stdout)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(report)
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 1 || value > 65535 {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
