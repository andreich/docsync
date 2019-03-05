package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andreich/docsync/config"
	"github.com/andreich/docsync/crypt"
	"github.com/andreich/docsync/manifest"
	"github.com/andreich/docsync/mover"
	"github.com/andreich/docsync/storage"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	configFile = flag.String("config", "$HOME/.docsync/config.json", "The configuration file to read.")
	dryRun     = flag.Bool("dry_run", true, "Simulate running but don't write anything to storage.")
	port       = flag.Int("port", 9871, "Port on which to expose metrics about the run.")
)

func uploadContent(ctx context.Context, s storage.Storage, enc crypt.Encryption, dst string, data []byte) error {
	data, err := enc.Encrypt(data)
	if err != nil {
		return err
	}
	if *dryRun {
		log.Printf("dry run: uploading to %s (%d bytes)", dst, len(data))
		return nil
	}
	return s.Upload(ctx, dst, data)
}

func upload(ctx context.Context, s storage.Storage, enc crypt.Encryption, srcfilename, dstfilename string) error {
	data, err := ioutil.ReadFile(srcfilename)
	if err != nil {
		return err
	}
	return uploadContent(ctx, s, enc, dstfilename, data)
}

func main() {
	flag.Parse()
	ctx := context.Background()

	tctx, _ := context.WithTimeout(ctx, 10*time.Minute)
	if err := waitForInternetAccess(tctx); err != nil {
		log.Fatalf("No internet access available: %v", err)
	}

	*configFile = os.ExpandEnv(*configFile)
	log.Printf("Started with config: %s", *configFile)

	if *port > 0 {
		go func() {
			mux := http.NewServeMux()
			if err := setupPrometheusExport(mux); err != nil {
				log.Printf("Could not set up Prometheus export: %v", err)
			}
			if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), mux); err != nil {
				log.Fatalf("Could not set up HTTP server: %v", err)
			}
		}()
	}

	cfg := &config.Sync{}
	if err := cfg.Parse(*configFile); err != nil {
		log.Fatalf("Could not load config from %q: %v", *configFile, err)
	}
	cfgMover := &mover.EmbeddedConfig{}
	if err := cfgMover.Parse(*configFile); err != nil {
		log.Fatalf("Could not load mover config from %q: %v", *configFile, err)
	}
	mv := mover.New(cfgMover.Mover)

	enc, err := crypt.New(cfg.AESPassphrase)
	if err != nil {
		log.Fatalf("Could not set up encryption/decryption with passphrase %q: %v", cfg.AESPassphrase, err)
	}

	creds, err := json.Marshal(cfg.Credentials)
	if err != nil {
		log.Fatalf("Could not serialize credentials: %v", err)
	}
	s, err := storage.New(ctx, cfg.BucketName, creds)
	if err != nil {
		log.Fatalf("Could not initialize storage: %v", err)
	}

	if err := setupStackdriverExport(cfg.Credentials["project_id"], creds); err != nil {
		log.Printf("Could not set up Stackdriver export: %v", err)
	}

	view.SetReportingPeriod(cfg.Interval.Duration / 2)

	if err := registerViews(); err != nil {
		log.Fatalf("Could not set up view for monitoring: %v", err)
	}

	m := manifest.New(cfg.Include, cfg.Exclude)
	data, err := s.Download(ctx, cfg.RemoteManifestFile)
	if err != nil {
		log.Printf("Could not restore manifest from remote file %q: %v", cfg.RemoteManifestFile, err)
		log.Printf("Initializing empty manifest")
	} else {
		data, err := enc.Decrypt(data)
		if err != nil {
			log.Fatalf("Could not decrypt remote manifest file: %v", err)
		}
		var buf bytes.Buffer
		if _, err := buf.Write(data); err != nil {
			log.Fatalf("Could not write remote manifest file to buffer: %v", err)
		}
		if err := m.Load(&buf); err != nil {
			log.Printf("Could not load manifest from remote file: %v", err)
		}
	}
	for {
		if _, err := mv.Scan(*dryRun); err != nil {
			log.Printf("Could not perform moves: %v", err)
		}

		changedEntries := 0
		for src, dst := range cfg.Dirs {
			changed, err := m.Update(src)
			if err != nil {
				log.Printf("Breaking update loop due to error: %v", err)
				break
			}
			for _, e := range changed {
				changedEntries++
				dstfn := strings.Replace(e, src, dst, 1)
				if err := upload(ctx, s, enc, e, dstfn); err != nil {
					log.Printf("Could not upload %q to %q: %v", e, dstfn, err)
					stats.Record(ctx, uploadedFilesErrCounter.M(1))
				}
				stats.Record(ctx, uploadedFilesCounter.M(1))
			}
		}
		if changedEntries > 0 {
			var buf bytes.Buffer
			if err := m.Dump(&buf); err != nil {
				log.Printf("Could not dump manifest to buffer: %v", err)
				break
			}
			if err := uploadContent(ctx, s, enc, cfg.RemoteManifestFile, buf.Bytes()); err != nil {
				log.Printf("Could not upload %q to %q: %v", cfg.ManifestFile, cfg.RemoteManifestFile, err)
			}
		}
		log.Printf("Changed entries %d; Sleeping %v", changedEntries, cfg.Interval)
		time.Sleep(cfg.Interval.Duration)
	}
}
