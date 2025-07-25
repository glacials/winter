package cmd // import "twos.dev/winter/cmd"

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

const (
	port            = 8100
	serveFlag       = "serve"
	sourceDir       = "src"
	staticAssetsDir = "public"
)

var (
	ignoreDirectories = map[string]struct{}{
		".git":         {},
		".github":      {},
		dist:           {},
		"node_modules": {},
	}
	serve *bool
)

// Builder is a function that builds a source file src into a destination
// directory dst.
type Builder func(src, dst string, cfg document.Config) error

func newBuildCmd() *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Winter project",
		Long: cliutils.Sprintf(`
			Build the Winter project in the current directory into ` + "`./dist`" + `.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Reading config.")
			cfg, err := document.NewConfig()
			if err != nil {
				return err
			}

			slog.Debug("Building substructure.")
			s, err := document.NewSubstructure(cfg)
			if err != nil {
				return err
			}

			slog.Debug("Executing templates.")
			if err := s.ExecuteAll(dist); err != nil {
				return err
			}

			if !*serve {
				cliutils.Printf(
					"Build complete! Generated %d pages and %d images. Stay frosty!\n",
					s.DocumentCount(),
					s.ImageCount(),
				)
			}

			if *serve {
				baseURL := url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("%s:%d", "localhost", port),
				}
				stop := make(chan struct{})
				reloader := Reloader{
					Ignore:       ignoreDirectories,
					Substructure: s,
					closeSockets: make(chan struct{}),
				}

				var mux http.ServeMux
				mux.Handle("/", http.FileServer(http.Dir(dist)))
				mux.Handle("/ws", reloader.Handler())

				server := http.Server{
					Addr:    fmt.Sprintf(":%d", port),
					Handler: &mux,
				}
				server.RegisterOnShutdown(reloader.Shutdown)

				go listenForCtrlC(stop, &server, &reloader)
				go startFileServer(&server)

				if err := reloader.Watch(append(cfg.Src, ".")); err != nil {
					return err
				}

				log.Printf("Serving %s on %s", dist, baseURL.String())

				<-stop
				return nil
			}

			return nil
		},
	}

	f := buildCmd.PersistentFlags()
	serve = f.BoolP(
		serveFlag,
		"s",
		false,
		"start a webserver and rebuild on file changes",
	)

	return buildCmd
}

func listenForCtrlC(stop chan struct{}, srvr *http.Server, reloader *Reloader) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	slog.Debug("Stopping")
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	if err := srvr.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	stop <- struct{}{}
}

func startFileServer(server *http.Server) {
	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			// Expected when we call server.Shutdown()
			return
		}
		log.Fatal(fmt.Errorf("can't listen and serve: %w", err))
	}
}
