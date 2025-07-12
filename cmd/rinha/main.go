package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avalonbits/rinha/cmd/setup"
	"github.com/avalonbits/rinha/config"
)

var (
	port = flag.String("port", "", "If set, overrides port from config.")
)

func main() {
	flag.Parse()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	cfg := config.Get(context.Background())
	server := setup.Echo(cfg)
	if *port != "" {
		cfg.Port = *port
	}

	go func() {
		if err := server.Start(fmt.Sprintf(":%s", cfg.Port)); err != nil && err != http.ErrServerClosed {
			server.Logger.Fatal(err)
		}
	}()

	<-sigc
	fmt.Println("Shutting down server.")
	server.MarkUnavailable()

	if err := server.Shutdown(context.Background()); err != nil {
		server.Logger.Fatal(err)
	}

	fmt.Println("Cleaning up server.")
	server.Cleanup()

	fmt.Println("Wating for server to perform cleanup")
	time.Sleep(5 * time.Second)
	fmt.Println("Done. bye!")
}
