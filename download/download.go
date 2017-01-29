package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/andreich/docsync/config"
	"github.com/andreich/docsync/crypt"
	"github.com/andreich/docsync/storage"
)

var (
	configFile  = flag.String("config", "$HOME/.docsync/config.json", "Configuration file for downloading data.")
	filename    = flag.String("filename", "", "The file to download from Google Cloud")
	destination = flag.String("destination", "", "The name of the file locally.")
)

func main() {
	flag.Parse()
	*configFile = os.ExpandEnv(*configFile)

	if *filename == "" {
		log.Fatal("--fiename is required.")
	}
	if *destination == "" {
		log.Fatal("--destination is required.")
	}

	cfg := &config.Upload{}
	if err := cfg.Parse(*configFile); err != nil {
		log.Fatalf("Could not parse config from %q: %v", *configFile, err)
	}

	enc, err := crypt.New(cfg.AESPassphrase)
	if err != nil {
		log.Fatalf("Could not create decryption: %v", err)
	}
	log.Printf("Decrypting with passphrase: %q", cfg.AESPassphrase)

	log.Printf("Downloading %q to %q", *filename, *destination)

	ctx := context.Background()
	s, err := storage.New(ctx, cfg.CredentialsFile, cfg.BucketName)
	if err != nil {
		log.Fatalf("Could not initialize storage: %v", err)
	}
	if data, err := s.Download(ctx, *filename); err != nil {
		log.Fatalf("Could not download %q: %v", *filename, err)
	} else {
		bytes, err := enc.Decrypt(data)
		if err != nil {
			log.Fatalf("Decryption failed: %v", err)
		}
		if err := ioutil.WriteFile(*destination, bytes, 0600); err != nil {
			log.Fatalf("Could not write bytes to %q: %v", *destination, err)
		}
	}
}
