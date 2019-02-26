package storage

import (
	"context"
	"io/ioutil"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	googleStorage "cloud.google.com/go/storage"
)

// Storage is the minimal interface for a cloud storage layer.
type Storage interface {
	// Upload a file under the given name, with the provided contents.
	Upload(ctx context.Context, name string, contents []byte) error
	// Download a file with the given name.
	Download(ctx context.Context, name string) ([]byte, error)
	// List contents of the bucket.
	List(ctx context.Context, prefix string) ([]string, error)
}

var newClient = googleStorage.NewClient

// New creates a new Google Cloud Storage client.
func New(ctx context.Context, bucket string, creds []byte) (Storage, error) {
	s, err := newClient(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return nil, err
	}
	b := s.Bucket(bucket)
	return &storageImpl{
		client: s,
		bucket: b,
	}, nil
}

type storageImpl struct {
	client *googleStorage.Client
	bucket *googleStorage.BucketHandle
}

func (s *storageImpl) Upload(ctx context.Context, name string, contents []byte) error {
	obj := s.bucket.Object(name)
	w := obj.NewWriter(ctx)
	if _, err := w.Write(contents); err != nil {
		return err
	}
	return w.Close()
}

func (s *storageImpl) Download(ctx context.Context, name string) ([]byte, error) {
	obj := s.bucket.Object(name)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, r.Close()
}

func (s *storageImpl) List(ctx context.Context, prefix string) ([]string, error) {
	var q *googleStorage.Query
	if prefix != "" {
		q = &googleStorage.Query{
			Prefix: prefix,
		}
	}
	var res []string
	it := s.bucket.Objects(ctx, q)
	for {
		objattr, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		res = append(res, objattr.Name)
	}
	return res, nil
}
