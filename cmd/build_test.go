package cmd

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeWithListenerAnnouncesURL(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	var out bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- serveWithListener(
			server,
			listener,
			dist,
			"http://localhost:8100",
			&out,
		)
	}()

	want := servingMessage(dist, "http://localhost:8100")
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(out.String(), want) {
		if time.Now().After(deadline) {
			t.Fatalf("expected %q in output, got %q", want, out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("serveWithListener: %v", err)
	}
}
