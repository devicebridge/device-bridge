package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("DEVICE_BRIDGE_HTTP_HOST")
	os.Unsetenv("DEVICE_BRIDGE_HTTP_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPHost != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %q", cfg.HTTPHost)
	}

	if cfg.HTTPPort != 8080 {
		t.Fatalf("expected port 8080, got %d", cfg.HTTPPort)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("DEVICE_BRIDGE_HTTP_HOST", "127.0.0.1")
	defer os.Unsetenv("DEVICE_BRIDGE_HTTP_HOST")

	os.Setenv("DEVICE_BRIDGE_HTTP_PORT", "9090")
	defer os.Unsetenv("DEVICE_BRIDGE_HTTP_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPHost != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %q", cfg.HTTPHost)
	}

	if cfg.HTTPPort != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.HTTPPort)
	}
}

func TestLoadSourcesFromEnv(t *testing.T) {
	t.Setenv("DEVICE_BRIDGE_SOURCES", "scanner-main, scanner-secondary")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Sources) != 2 || cfg.Sources[0] != "scanner-main" || cfg.Sources[1] != "scanner-secondary" {
		t.Fatalf("unexpected sources: %#v", cfg.Sources)
	}
}

func TestLoadScannerPathFromEnv(t *testing.T) {
	t.Setenv("DEVICE_BRIDGE_SCANNER_PATH", "/tmp/scanner")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScannerPath != "/tmp/scanner" {
		t.Fatalf("expected scanner path, got %q", cfg.ScannerPath)
	}
}

func TestLoadSerialSettings(t *testing.T) {
	t.Setenv("DEVICE_BRIDGE_SCANNER_BAUD", "115200")
	t.Setenv("DEVICE_BRIDGE_SCANNER_RECONNECT_SECONDS", "3")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScannerBaud != 115200 || cfg.ScannerReconnectDelay != 3 {
		t.Fatalf("unexpected serial settings: baud=%d reconnect=%d", cfg.ScannerBaud, cfg.ScannerReconnectDelay)
	}
}

func TestLoadRejectsEmptySourceName(t *testing.T) {
	t.Setenv("DEVICE_BRIDGE_SOURCES", "scanner-main,,scanner-secondary")

	if _, err := Load(); err == nil {
		t.Fatal("expected empty source name error")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	testCases := []string{
		"abc",
		"0",
		"65536",
		"-1",
		"99999",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			os.Setenv("DEVICE_BRIDGE_HTTP_HOST", "0.0.0.0")
			defer os.Unsetenv("DEVICE_BRIDGE_HTTP_HOST")

			os.Setenv("DEVICE_BRIDGE_HTTP_PORT", tc)
			defer os.Unsetenv("DEVICE_BRIDGE_HTTP_PORT")

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for port %q, got nil", tc)
			}
		})
	}
}

func TestListenAddr(t *testing.T) {
	cfg := &Config{
		HTTPHost: "0.0.0.0",
		HTTPPort: 8080,
	}

	if addr := cfg.ListenAddr(); addr != "0.0.0.0:8080" {
		t.Fatalf("expected 0.0.0.0:8080, got %q", addr)
	}
}

func TestListenAddrCustom(t *testing.T) {
	cfg := &Config{
		HTTPHost: "127.0.0.1",
		HTTPPort: 9090,
	}

	if addr := cfg.ListenAddr(); addr != "127.0.0.1:9090" {
		t.Fatalf("expected 127.0.0.1:9090, got %q", addr)
	}
}
