package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"music-server/server/internal/config"
	"music-server/server/internal/scan"
	"music-server/server/internal/serve"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [serve|scan] [-config music-server.yaml]\n", os.Args[0])
		os.Exit(2)
	}

	subcmd := os.Args[1]
	fs := flag.NewFlagSet(subcmd, flag.ExitOnError)
	configPath := fs.String("config", "music-server.yaml", "config file path")
	_ = fs.Parse(os.Args[2:])

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	switch subcmd {
	case "serve":
		h := serve.NewHandler(cfg)
		addr := fmt.Sprintf(":%d", cfg.Serve.Port)
		log.Printf("serving %s at %s (url base %s)", cfg.Common.Root, addr, cfg.Common.Path)
		if err := http.ListenAndServe(addr, h); err != nil {
			log.Fatal(err)
		}
	case "scan":
		if err := scan.Run(cfg); err != nil {
			log.Fatal(err)
		}
		log.Printf("scan done: %s", cfg.Common.Root)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcmd)
		os.Exit(2)
	}
}
