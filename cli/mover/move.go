package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/andreich/docsync/mover"
)

var (
	configFile = flag.String("config", "$HOME/.docsync/config.json", "The configuration file to read.")
	dryRun     = flag.Bool("dry_run", true, "If true, just print the moves, don't carry them on.")
	interval   = flag.Duration("interval", time.Minute, "How long to sleep between scans.")
)

func main() {
	flag.Parse()
	*configFile = os.ExpandEnv(*configFile)
	log.Printf("Started with config: %s", *configFile)

	cfg := &mover.EmbeddedConfig{}
	if err := cfg.Parse(*configFile); err != nil {
		log.Fatalf("Could not load config from %q: %v", *configFile, err)
	}

	m := mover.New(cfg.Mover)

	for {
		if _, err := m.Scan(*dryRun); err != nil {
			log.Fatal(err)
		}
		log.Printf("Sleeping %v", *interval)
		time.Sleep(*interval)
	}
}
