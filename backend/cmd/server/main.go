package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"amnezia_go/backend/internal/api"
	"amnezia_go/backend/internal/auth"
	appconfig "amnezia_go/backend/internal/config"
	"amnezia_go/backend/internal/ipam"
	"amnezia_go/backend/internal/service"
	"amnezia_go/backend/internal/store"
	"amnezia_go/backend/internal/wg"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	interfaceName := getEnv("AWG_INTERFACE", "awg0")
	runtimeMode := getEnv("AWG_RUNTIME_MODE", "strict")
	fallbackServerPublicKey := getEnv("AWG_SERVER_PUBLIC_KEY", "UNAVAILABLE_SERVER_PUBLIC_KEY")
	addr := getEnv("HTTP_ADDR", ":8080")
	dataDir := getEnv("DATA_DIR", "./data")
	dbPath := getEnv("SQLITE_PATH", filepath.Join(dataDir, "awg.db"))
	templatePath := getEnv("AWG_TEMPLATE_PATH", "./config/awg-template.env")
	serverHost := getEnv("AWG_SERVER_HOST", "127.0.0.1")
	serverPort := getEnvAsInt("AWG_SERVER_PORT", 51820)
	dns := getEnv("AWG_CLIENT_DNS", "1.1.1.1")
	adminUser := getEnv("ADMIN_USERNAME", "admin")
	adminPass := getEnv("ADMIN_PASSWORD", "admin")

	var wgService wg.Service
	wgService, err := initWGService(logger, interfaceName, runtimeMode, fallbackServerPublicKey)
	if err != nil {
		logger.Error("failed to initialize wg service", "error", err, "mode", runtimeMode)
		os.Exit(1)
	}
	defer wgService.Close()

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		logger.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	repo, err := store.NewSQLitePeerRepository(dbPath)
	if err != nil {
		logger.Error("failed to initialize sqlite repository", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	if err := repo.Init(context.Background()); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	tmpl, err := appconfig.LoadAWGTemplate(templatePath)
	if err != nil {
		logger.Error("failed to load awg template", "error", err)
		os.Exit(1)
	}

	authService, err := auth.NewService(adminUser, adminPass, 12*time.Hour)
	if err != nil {
		logger.Error("failed to initialize auth service", "error", err)
		os.Exit(1)
	}

	peerService := service.NewPeerService(
		repo,
		wgService,
		ipam.NewAllocator("10.8.0.0/24"),
		tmpl,
		serverHost,
		serverPort,
		dns,
	)

	router := api.NewRouter(authService, peerService).Build()

	logger.Info("starting api server", "addr", addr, "interface", interfaceName)
	if err := router.Run(addr); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func initWGService(logger *slog.Logger, interfaceName, runtimeMode, fallbackServerPublicKey string) (wg.Service, error) {
	probe := func(service wg.Service) error {
		_, err := service.ServerPublicKey(context.Background())
		return err
	}

	switch runtimeMode {
	case "disabled":
		logger.Warn("wg runtime is disabled")
		return wg.NewNoopService(fallbackServerPublicKey), nil
	case "optional":
		realService, err := wg.NewWGCtrlService(interfaceName)
		if err != nil {
			logger.Warn("wg runtime unavailable, using noop mode", "error", err)
			return wg.NewNoopService(fallbackServerPublicKey), nil
		}
		if err := probe(realService); err != nil {
			logger.Warn("awg interface unavailable, using noop mode", "interface", interfaceName, "error", err)
			_ = realService.Close()
			return wg.NewNoopService(fallbackServerPublicKey), nil
		}
		return realService, nil
	case "strict":
		fallthrough
	default:
		realService, err := wg.NewWGCtrlService(interfaceName)
		if err != nil {
			return nil, err
		}
		if err := probe(realService); err != nil {
			_ = realService.Close()
			return nil, fmt.Errorf("awg interface %q is not available: %w", interfaceName, err)
		}
		return realService, nil
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid integer env %s=%q", key, value))
	}
	return n
}
