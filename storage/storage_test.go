package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"cloud.google.com/go/httpreplay"
	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/option"

	googleStorage "cloud.google.com/go/storage"
)

const replayFile = "storage.replay"

var record = flag.Bool("record", false, "Record requests.")
var rng *rand.Rand

func TestMain(m *testing.M) {
	rng = rand.New(rand.NewSource(0))
	cleanup := initIntegrationTest()
	exit := m.Run()
	if err := cleanup(); err != nil {
		log.Printf("Post-test cleanup failed: %v", err)
	}
	os.Exit(exit)
}

func initIntegrationTest() func() error {
	flag.Parse()
	switch {
	case httpreplay.Supported() && *record:
		initial, err := json.Marshal(time.Now().UTC())
		if err != nil {
			log.Fatalf("Could not marshal initial bytes: %v", err)
		}
		recorder, err := httpreplay.NewRecorder(replayFile, initial)
		if err != nil {
			log.Fatalf("Could not initialize recorder: %v", err)
		}
		newClient = func(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
			opts = append(opts, option.WithScopes(googleStorage.ScopeReadWrite))
			client, err := recorder.Client(ctx, opts...)
			if err != nil {
				return nil, err
			}
			return googleStorage.NewClient(ctx, option.WithHTTPClient(client))
		}
		return func() error {
			return recorder.Close()
		}
	case httpreplay.Supported():
		replayer, err := httpreplay.NewReplayer(replayFile)
		if err != nil {
			log.Fatalf("Could not initialize replayer: %v", err)
		}
		newClient = func(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
			client, err := replayer.Client(ctx)
			if err != nil {
				return nil, err
			}
			return googleStorage.NewClient(ctx, option.WithHTTPClient(client))
		}
		return func() error {
			return replayer.Close()
		}
	default:
		log.Fatal("httpreplay not supported")
	}
	return func() error {
		return nil
	}
}

func projectID(cnt []byte) string {
	res := map[string]string{}
	if err := json.Unmarshal(cnt, &res); err != nil {
		log.Fatalf("could not unmarshal JSON: %v", err)
	}
	if _, found := res["project_id"]; !found {
		log.Fatal("could not read project ID: missing project_id key")
	}
	return res["project_id"]
}

func prefix(in string) string {
	return fmt.Sprintf("storage-sync-test-%s", in)
}

func TestUploadDownload(t *testing.T) {
	ctx := context.Background()
	log.Printf("env: %+v", os.Environ())
	creds := []byte(os.Getenv("DOCSYNC_CREDS_JSON"))
	var err error
	if len(creds) == 0 {
		creds, err = ioutil.ReadFile("creds.json")
		if err != nil {
			t.Fatalf("could not load credentials: %v", err)
		}
	}

	uuid.SetRand(rng)
	bucket := prefix(uuid.New().String())
	// mlog "github.com/google/martian/log"
	// mlog.SetLevel(mlog.Info)

	client, err := newClient(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		t.Errorf("Could not create client: %v", err)
	}
	if err := client.Bucket(bucket).Create(ctx, projectID(creds), &googleStorage.BucketAttrs{
		StorageClass:      "REGIONAL",
		Location:          "europe-west1",
		VersioningEnabled: false,
	}); err != nil {
		t.Errorf("Could not create bucket: %v", err)
	}
	defer func() {
		if err := client.Bucket(bucket).Delete(ctx); err != nil {
			log.Printf("Could not clean up bucket %q: %v", bucket, err)
		}
	}()

	s, err := New(ctx, bucket, creds)
	if err != nil {
		t.Fatalf("could not create Storage: %v", err)
	}

	var want []string
	for i := 0; i < 10; i++ {
		filename := prefix(uuid.New().String())
		want = append(want, filename)

		write := []byte(fmt.Sprintf("this is a test file %d", i))
		if err := s.Upload(ctx, filename, write); err != nil {
			t.Fatalf("could not upload: %v", err)
		}
	}
	for i, filename := range want {
		wantContent := []byte(fmt.Sprintf("this is a test file %d", i))
		if got, err := s.Download(ctx, filename); err != nil {
			t.Fatalf("could not download: %v", err)
		} else if !bytes.Equal(got, wantContent) {
			t.Errorf("content mismatch:\nuploaded %+v\ndownloaded %+v", wantContent, got)
		}
	}

	l, err := s.List(ctx, "storage-")
	if err != nil {
		t.Errorf("could not list from bucket %q: %v", bucket, err)
	}
	sort.Strings(l)
	sort.Strings(want)
	if !reflect.DeepEqual(l, want) {
		t.Errorf("listing mismatch: got %v, want %v", l, want)
	}
	for _, obj := range l {
		if err := client.Bucket(bucket).Object(obj).Delete(ctx); err != nil {
			t.Errorf("could not delete object %q: %v", obj, err)
		}
	}
}
