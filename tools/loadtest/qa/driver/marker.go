package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// stopMarker is the pause marker: a local file, or an S3 object when the driver runs where nobody can touch its
// files (ECS). Whoever creates it pauses the workload; removing it resumes the workload.
type stopMarker interface {
	// get returns the marker's first line and whether it exists.
	get(ctx context.Context) (string, bool, error)
	// create writes the marker unless it exists.
	create(ctx context.Context, line string) error
	String() string
}

func newStopMarker(ctx context.Context, where, region string) (stopMarker, error) {
	rest, ok := strings.CutPrefix(where, "s3://")
	if !ok {
		return &fileMarker{path: where}, nil
	}
	bucket, key, ok := strings.Cut(rest, "/")
	if !ok || bucket == "" || key == "" {
		return nil, fmt.Errorf("stop file %q: want s3://bucket/key", where)
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS config: %w", err)
	}
	return &s3Marker{client: s3.NewFromConfig(cfg), bucket: bucket, key: key}, nil
}

type fileMarker struct{ path string }

func (m *fileMarker) String() string { return m.path }

func (m *fileMarker) get(context.Context) (string, bool, error) {
	b, err := os.ReadFile(m.path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false, nil
	case err != nil:
		return "", false, err
	}
	return firstLine(b), true, nil
}

func (m *fileMarker) create(_ context.Context, line string) error {
	f, err := os.OpenFile(m.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644) //nolint:gosec // a marker file, not a secret
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = f.WriteString(line + "\n")
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// s3Marker reads the object at most every s3MarkerEvery, because the workload asks before every action.
type s3Marker struct {
	client      *s3.Client
	bucket, key string

	mu     sync.Mutex
	readAt time.Time
	line   string
	exists bool
}

const s3MarkerEvery = 30 * time.Second

func (m *s3Marker) String() string { return "s3://" + m.bucket + "/" + m.key }

func (m *s3Marker) get(ctx context.Context) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if time.Since(m.readAt) < s3MarkerEvery {
		return m.line, m.exists, nil
	}
	out, err := m.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(m.bucket), Key: aws.String(m.key)})
	var nsk *s3types.NoSuchKey
	switch {
	case errors.As(err, &nsk):
		m.line, m.exists = "", false
	case err != nil:
		return m.line, m.exists, fmt.Errorf("read %s: %w", m, err)
	default:
		b, rerr := io.ReadAll(io.LimitReader(out.Body, 4096))
		out.Body.Close()
		if rerr != nil {
			return m.line, m.exists, fmt.Errorf("read %s: %w", m, rerr)
		}
		m.line, m.exists = firstLine(b), true
	}
	m.readAt = time.Now()
	return m.line, m.exists, nil
}

func (m *s3Marker) create(ctx context.Context, line string) error {
	_, err := m.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(m.bucket), Key: aws.String(m.key), Body: bytes.NewReader([]byte(line + "\n")),
		IfNoneMatch: aws.String("*"), // keep an existing marker (a person's reason) as it is
	})
	if err != nil && !strings.Contains(err.Error(), "PreconditionFailed") {
		return fmt.Errorf("write %s: %w", m, err)
	}
	m.mu.Lock()
	m.readAt = time.Time{} // read it again next time
	m.mu.Unlock()
	return nil
}

func firstLine(b []byte) string {
	return strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
}
