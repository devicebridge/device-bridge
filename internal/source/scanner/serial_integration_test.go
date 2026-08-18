//go:build linux && integration

package scanner

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSerialAdapterWithSocatPTY(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "app")
	injectorPath := filepath.Join(dir, "injector")

	cmd := exec.Command("socat", "-d", "-d",
		"pty,raw,echo=0,link="+appPath,
		"pty,raw,echo=0,link="+injectorPath,
	)
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start socat: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	if err := waitForPTY(appPath, injectorPath); err != nil {
		t.Fatal(err)
	}

	port, err := os.OpenFile(appPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewSerialAdapter(port, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- adapter.Run(ctx) }()

	injector, err := os.OpenFile(injectorPath, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer injector.Close()
	if _, err := injector.WriteString("first\r\nsecond\npartial\n"); err != nil {
		t.Fatal(err)
	}

	for _, expected := range []string{"first", "second", "partial"} {
		select {
		case input := <-adapter.Input():
			if input.Value != expected {
				t.Fatalf("expected %q, got %q", expected, input.Value)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %q", expected)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serial adapter did not stop")
	}
}

func waitForPTY(appPath, injectorPath string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(appPath); err == nil {
			if _, err := os.Stat(injectorPath); err == nil {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return os.ErrDeadlineExceeded
}
