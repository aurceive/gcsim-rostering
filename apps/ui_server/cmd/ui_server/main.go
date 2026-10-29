package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/engine"
	"github.com/aurceive/gcsim-rostering/apps/ui_server/internal/web"
)

func findRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".gitmodules")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s", start)
		}
		dir = parent
	}
}

func run() error {
	rootFlag := flag.String("root", "", "repository root (auto-detected if omitted)")
	port := flag.Int("port", 54320, "local UI port")
	workers := flag.Int("workers", 10, "workers per engine")
	timeout := flag.Int("timeout", 300, "simulation timeout in seconds")
	flag.Parse()
	if *port < 1 || *port > 65535 || *workers < 1 || *timeout < 1 {
		return errors.New("port must be 1-65535; workers and timeout must be positive")
	}
	root := *rootFlag
	if root == "" {
		var err error
		root, err = findRoot(".")
		if err != nil {
			exe, exeErr := os.Executable()
			if exeErr != nil {
				return exeErr
			}
			root, err = findRoot(filepath.Dir(exe))
			if err != nil {
				return err
			}
		}
	}
	logger := log.New(os.Stdout, "ui_server: ", log.LstdFlags)
	manager, err := engine.New(root, *workers, *timeout, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := manager.Close(); err != nil {
			logger.Printf("close engines: %v", err)
		}
	}()
	handler, err := web.New(manager, filepath.Join(root, "apps", "ui_server", "web", "dist"), logger)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	defer func() {
		if err := server.Close(); err != nil {
			logger.Printf("close HTTP server: %v", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			logger.Printf("HTTP shutdown: %v", err)
			if err := server.Close(); err != nil {
				logger.Printf("force HTTP shutdown: %v", err)
			}
		}
	}()
	logger.Printf("open http://127.0.0.1:%d", *port)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
