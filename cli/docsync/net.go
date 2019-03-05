package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

func checkInternetAccess() error {
	urls := []string{
		"https://github.com/andreich/docsync",
		"https://travis-ci.org/andreich/docsync",
		"https://www.google.com",
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	var lastErr error
	var res *http.Response
	for _, url := range urls {
		res, lastErr = client.Get(url)
		if lastErr == nil {
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("not yet connected")
}

func waitForInternetAccess(ctx context.Context) error {
	t := time.NewTicker(30 * time.Second)
	if err := checkInternetAccess(); err == nil {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
			if err := checkInternetAccess(); err != nil {
				log.Printf("Internet Check: %v", err)
			} else {
				return nil
			}
		}
	}
}
