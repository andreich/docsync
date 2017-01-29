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
	configFile = flag.String("config", "$HOME/.docsync/config.json", "Configuration file for uploding data.")
	filename   = flag.String("filename", "", "The file to upload to Google Cloud")
)

func main() {
	flag.Parse()
	cfg := &config.Upload{}
	*configFile = os.ExpandEnv(*configFile)
	if err := cfg.Parse(*configFile); err != nil {
		log.Fatalf("Could not parse config from %q: %v", *configFile, err)
	}

	bytes, err := ioutil.ReadFile(*filename)
	if err != nil {
		log.Fatalf("Could not read file %q: %v", *filename, err)
	}

	enc, err := crypt.New(cfg.AESPassphrase)
	if err != nil {
		log.Fatalf("Could not create encryption: %v", err)
	}
	log.Printf("Encrypting with passphrase: %q", cfg.AESPassphrase)

	data, err := enc.Encrypt(bytes)
	if err != nil {
		log.Fatalf("Encryption failed: %v", err)
	}

	log.Printf("Uploading %q (%d bytes)", *filename, len(data))

	ctx := context.Background()
	s, err := storage.New(ctx, cfg.CredentialsFile, cfg.BucketName)
	if err != nil {
		log.Fatalf("Could not initialize storage: %v", err)
	}
	if err := s.Upload(ctx, *filename, data); err != nil {
		log.Fatalf("Could not upload %q: %v", *filename, err)
	}
}
