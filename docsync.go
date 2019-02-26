package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/andreich/docsync/config"
	"github.com/andreich/docsync/crypt"
	"github.com/andreich/docsync/manifest"
	"github.com/andreich/docsync/storage"
)

var (
	configFile = flag.String("config", "$HOME/.docsync/config.json", "The configuration file to read.")
)

func uploadContent(ctx context.Context, s storage.Storage, enc crypt.Encryption, dst string, data []byte) error {
	data, err := enc.Encrypt(data)
	if err != nil {
		return err
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
	*configFile = os.ExpandEnv(*configFile)
	log.Printf("Started with config: %s", *configFile)

	cfg := &config.Sync{}
	if err := cfg.Parse(*configFile); err != nil {
		log.Fatalf("Could not load config from %q: %v", *configFile, err)
	}
	fmt.Printf("%+v\n", cfg)

	enc, err := crypt.New(cfg.AESPassphrase)
	if err != nil {
		log.Fatalf("Could not set up encryption/decryption with passphrase %q: %v", cfg.AESPassphrase, err)
	}

	ctx := context.Background()
	creds, err := json.Marshal(cfg.Credentials)
	if err != nil {
		log.Fatalf("Could not serialize credentials: %v", err)
	}
	s, err := storage.New(ctx, cfg.BucketName, creds)
	if err != nil {
		log.Fatalf("Could not initialize storage: %v", err)
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
		changedEntries := 0
		for src, dst := range cfg.Dirs {
			changed, err := m.Update(src)
			if err != nil {
				log.Printf("Breaking main loop due to error: %v", err)
				break
			}
			for _, e := range changed {
				changedEntries++
				dstfn := strings.Replace(e, src, dst, 1)
				if err := upload(ctx, s, enc, e, dstfn); err != nil {
					log.Printf("Could not upload %q to %q: %v", e, dstfn, err)
				}
			}
		}
		log.Printf("Changed entries %d; Sleeping %v", changedEntries, cfg.Interval)
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
		time.Sleep(cfg.Interval.Duration)
	}
}
