package fleet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// InstallerReader is the content of a software installer as consumed by the
// metadata parsers and the installer store. It is implemented by
// TempFileReader (local temp file) and RemoteInstallerReader (staging object
// in the installer store, read with ranged requests).
type InstallerReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	// Rewind seeks back to the start of the content.
	Rewind() error
	// Size returns the total size of the content in bytes.
	Size() (int64, error)
}

// InstallerReadCloser is an InstallerReader that must be closed to release
// its backing storage (temp file or staging object).
type InstallerReadCloser interface {
	InstallerReader
	io.Closer
}

// KnownSHA256er is implemented by installer readers whose SHA256 was already
// computed (e.g. while streaming the bytes to the store), so that parsers don't
// need to read the whole content again just to hash it.
type KnownSHA256er interface {
	KnownSHA256() []byte
}

// StreamingInstallerStore is implemented by the object-store backed software
// installer stores (S3, Azure). It lets Fleet stream installer bytes to a
// staging object without buffering them on local disk, inspect them with
// ranged reads, and then promote them to their content-addressed key.
//
// Staging objects live under the same prefix as staged (client) uploads, so
// a staged upload ID is a valid staging ID and the staged uploads cleanup cron
// removes abandoned staging objects.
type StreamingInstallerStore interface {
	// PutStaging streams r to the staging object stagingID.
	PutStaging(ctx context.Context, stagingID string, r io.Reader) error
	// GetStagingRange returns the n bytes of the staging object starting at off.
	GetStagingRange(ctx context.Context, stagingID string, off, n int64) (io.ReadCloser, error)
	// StagingSize returns the size of the staging object.
	StagingSize(ctx context.Context, stagingID string) (int64, error)
	// Promote copies the staging object to the installer key installerID
	// (skipping the copy if that installer already exists) and deletes the
	// staging object (best effort).
	Promote(ctx context.Context, stagingID, installerID string) error
	// CopyStaging copies the staging object srcStagingID to dstStagingID.
	CopyStaging(ctx context.Context, srcStagingID, dstStagingID string) error
	// DeleteStaging deletes the staging object. Deleting a missing object is
	// not an error.
	DeleteStaging(ctx context.Context, stagingID string) error
}

// ErrStagingChanged is returned (wrapped) by an ETagStagingStore when a
// staging object no longer has the ETag it was pinned to.
var ErrStagingChanged = errors.New("staging object changed")

// ETagStagingStore is optionally implemented by a StreamingInstallerStore that
// can make reads and promotion of a staging object conditional on its ETag.
// It lets Fleet use a staging object that someone else can still overwrite (a
// staged client upload, whose presigned URL stays valid) in place instead of
// copying it first: if the object changes after it was pinned, every read and
// the promotion fail with ErrStagingChanged.
type ETagStagingStore interface {
	// StagingETag returns the size and ETag of the staging object. An empty
	// ETag means the store can't pin this object, and it must be copied.
	StagingETag(ctx context.Context, stagingID string) (size int64, etag string, err error)
	// GetStagingRangeIfMatch is GetStagingRange, failing if the staging
	// object's ETag isn't etag.
	GetStagingRangeIfMatch(ctx context.Context, stagingID, etag string, off, n int64) (io.ReadCloser, error)
	// PromoteIfMatch copies the staging object to the installer key
	// installerID (skipping the copy if that installer already exists),
	// failing if the staging object's ETag isn't etag. Unlike Promote, it
	// doesn't delete the staging object, which isn't Fleet's own.
	PromoteIfMatch(ctx context.Context, stagingID, etag, installerID string) error
}

// InstallerStreamValidator validates installer content while it is streamed
// (e.g. to the installer store), so that parsers don't need to read all of it
// again. Write must never fail: a validator that gives up just ignores the
// rest of the content.
type InstallerStreamValidator interface {
	io.Writer
	// Finish is called once all the content was written, or when streaming
	// failed. It returns the installer format the content was fully validated
	// as, or "" if it wasn't (the parser then validates it as usual).
	Finish() string
}

// PrevalidatedInstaller is implemented by installer readers whose content was
// already fully validated by an InstallerStreamValidator.
type PrevalidatedInstaller interface {
	// PrevalidatedFormat returns the format the content was validated as, or
	// "" if it wasn't.
	PrevalidatedFormat() string
}

const (
	remoteInstallerBlockSize   = 1 << 20 // 1 MiB
	remoteInstallerCacheBlocks = 16
)

// RemoteInstallerReader implements InstallerReader over a staging object of
// a StreamingInstallerStore. Sequential reads (Read) use a single streaming
// request from the current offset; random reads (ReadAt) fetch fixed-size
// blocks with ranged requests and keep the most recent ones in a small cache,
// as the parsers do many small reads close to each other.
//
// It keeps the context it was created with for the requests it makes, and is
// not safe for concurrent use.
type RemoteInstallerReader struct {
	ctx       context.Context // io interfaces don't take a context
	store     StreamingInstallerStore
	stagingID string
	size      int64
	sha256    []byte
	// etag, if set, pins the staging object: see ETagStagingStore.
	etag               string
	prevalidatedFormat string

	off     int64
	body    io.ReadCloser
	bodyOff int64

	blocks     map[int64][]byte
	blockOrder []int64

	deleteOnClose bool
}

var (
	_ InstallerReader       = (*RemoteInstallerReader)(nil)
	_ KnownSHA256er         = (*RemoteInstallerReader)(nil)
	_ PrevalidatedInstaller = (*RemoteInstallerReader)(nil)
	_ InstallerReadCloser   = (*RemoteInstallerReader)(nil)
	_ InstallerReadCloser   = (*TempFileReader)(nil)
)

// NewPinnedRemoteInstallerReader is NewRemoteInstallerReader for a staging
// object pinned to its ETag (see ETagStagingStore): the reader's requests and
// Promote fail with ErrStagingChanged if the object no longer has that ETag.
func NewPinnedRemoteInstallerReader(ctx context.Context, store StreamingInstallerStore, stagingID string, size int64, etag, sha256Hex string) (*RemoteInstallerReader, error) {
	if _, ok := store.(ETagStagingStore); !ok || etag == "" {
		return nil, errors.New("installer store can't pin staging objects")
	}
	r, err := NewRemoteInstallerReader(ctx, store, stagingID, size, sha256Hex)
	if err != nil {
		return nil, err
	}
	r.etag = etag
	return r, nil
}

// NewRemoteInstallerReader returns a reader for the staging object stagingID
// of the given size. sha256Hex is the hex-encoded SHA256 of the content
// (optional; when empty, KnownSHA256 returns nil).
func NewRemoteInstallerReader(ctx context.Context, store StreamingInstallerStore, stagingID string, size int64, sha256Hex string) (*RemoteInstallerReader, error) {
	var sum []byte
	if sha256Hex != "" {
		var err error
		if sum, err = hex.DecodeString(sha256Hex); err != nil {
			return nil, fmt.Errorf("decoding installer sha256: %w", err)
		}
	}
	return &RemoteInstallerReader{
		ctx:       ctx,
		store:     store,
		stagingID: stagingID,
		size:      size,
		sha256:    sum,
		blocks:    make(map[int64][]byte, remoteInstallerCacheBlocks),
	}, nil
}

// StageInstaller streams body to a new staging object of store, computing
// its size and SHA256 on the way, and returns a reader over it. The staging
// object is deleted when the reader is closed, unless it was promoted first.
// The validators see the content as it streams by; the format the first one
// fully validated is reported by the reader's PrevalidatedFormat.
//
// uploadCtx is used for the upload, readerCtx for the requests the returned
// reader makes later. If reading body fails, that error is returned as is so
// that callers can inspect it (e.g. size limit or stall).
func StageInstaller(uploadCtx, readerCtx context.Context, store StreamingInstallerStore, body io.Reader, validators ...InstallerStreamValidator) (*RemoteInstallerReader, error) {
	stagingID := uuid.NewString()
	h := sha256.New()
	writers := []io.Writer{h}
	for _, v := range validators {
		writers = append(writers, v)
	}
	// Finish releases the validators' resources, so it must run on failure too.
	finishValidators := func() string {
		var format string
		for _, v := range validators {
			if f := v.Finish(); format == "" {
				format = f
			}
		}
		return format
	}
	cr := &countingReader{r: io.TeeReader(body, io.MultiWriter(writers...))}
	if err := store.PutStaging(uploadCtx, stagingID, cr); err != nil {
		finishValidators()
		// The upload may have committed before failing; the staged uploads
		// cleanup is the backstop if this fails too.
		_ = store.DeleteStaging(context.WithoutCancel(readerCtx), stagingID)
		if cr.err != nil {
			return nil, cr.err
		}
		return nil, err
	}
	format := finishValidators()
	r, err := NewRemoteInstallerReader(readerCtx, store, stagingID, cr.n, hex.EncodeToString(h.Sum(nil)))
	if err != nil {
		_ = store.DeleteStaging(context.WithoutCancel(readerCtx), stagingID)
		return nil, err
	}
	r.deleteOnClose = true
	r.prevalidatedFormat = format
	return r, nil
}

// countingReader counts the bytes read and records the first read error other
// than io.EOF.
type countingReader struct {
	r   io.Reader
	n   int64
	err error
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if err != nil && !errors.Is(err, io.EOF) && c.err == nil {
		c.err = err
	}
	return n, err
}

// SetDeleteOnClose sets whether Close deletes the staging object.
func (r *RemoteInstallerReader) SetDeleteOnClose(v bool) { r.deleteOnClose = v }

// StagingID returns the ID of the staging object read by r.
func (r *RemoteInstallerReader) StagingID() string { return r.stagingID }

// KnownSHA256 returns the SHA256 the reader was created with, if any.
func (r *RemoteInstallerReader) KnownSHA256() []byte { return r.sha256 }

// PrevalidatedFormat returns the format the content was validated as while
// it was streamed, if any.
func (r *RemoteInstallerReader) PrevalidatedFormat() string { return r.prevalidatedFormat }

// SetPrevalidatedFormat records that the content was fully validated as the
// given format (see InstallerStreamValidator).
func (r *RemoteInstallerReader) SetPrevalidatedFormat(format string) { r.prevalidatedFormat = format }

// Promote stores the content under the installer key installerID with a
// server-side copy (see StreamingInstallerStore.Promote). A pinned staging
// object is only copied if it still has its ETag, and isn't deleted.
func (r *RemoteInstallerReader) Promote(ctx context.Context, installerID string) error {
	if r.etag != "" {
		return r.store.(ETagStagingStore).PromoteIfMatch(ctx, r.stagingID, r.etag, installerID)
	}
	return r.store.Promote(ctx, r.stagingID, installerID)
}

func (r *RemoteInstallerReader) getRange(off, n int64) (io.ReadCloser, error) {
	if r.etag != "" {
		return r.store.(ETagStagingStore).GetStagingRangeIfMatch(r.ctx, r.stagingID, r.etag, off, n)
	}
	return r.store.GetStagingRange(r.ctx, r.stagingID, off, n)
}

// Size returns the size of the staging object.
func (r *RemoteInstallerReader) Size() (int64, error) { return r.size, nil }

// Rewind seeks back to the start of the content.
func (r *RemoteInstallerReader) Rewind() error {
	_, err := r.Seek(0, io.SeekStart)
	return err
}

// Seek implements io.Seeker.
func (r *RemoteInstallerReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.off + offset
	case io.SeekEnd:
		abs = r.size + offset
	default:
		return 0, errors.New("remote installer reader: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("remote installer reader: negative position")
	}
	r.off = abs
	return abs, nil
}

// Read implements io.Reader.
func (r *RemoteInstallerReader) Read(p []byte) (int, error) {
	if r.off >= r.size {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.body == nil || r.bodyOff != r.off {
		r.closeBody()
		body, err := r.getRange(r.off, r.size-r.off)
		if err != nil {
			return 0, err
		}
		r.body, r.bodyOff = body, r.off
	}
	if remaining := r.size - r.off; int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := r.body.Read(p)
	r.off += int64(n)
	r.bodyOff += int64(n)
	if errors.Is(err, io.EOF) {
		r.closeBody()
		if r.off < r.size {
			return n, io.ErrUnexpectedEOF
		}
		err = nil
	}
	return n, err
}

// ReadAt implements io.ReaderAt.
func (r *RemoteInstallerReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("remote installer reader: negative offset")
	}
	var n int
	for n < len(p) {
		pos := off + int64(n)
		if pos >= r.size {
			return n, io.EOF
		}
		blockIdx := pos / remoteInstallerBlockSize
		block, err := r.block(blockIdx)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], block[pos-blockIdx*remoteInstallerBlockSize:])
	}
	return n, nil
}

// Close releases the streaming request used by Read, if any. For readers
// created by StageInstaller, it also deletes the staging object (a no-op if it
// was already promoted), like TempFileReader.Close deletes its temp file.
func (r *RemoteInstallerReader) Close() error {
	r.closeBody()
	if r.deleteOnClose {
		return r.store.DeleteStaging(context.WithoutCancel(r.ctx), r.stagingID)
	}
	return nil
}

func (r *RemoteInstallerReader) closeBody() {
	if r.body != nil {
		_ = r.body.Close()
		r.body = nil
	}
}

func (r *RemoteInstallerReader) block(idx int64) ([]byte, error) {
	if b, ok := r.blocks[idx]; ok {
		return b, nil
	}
	start := idx * remoteInstallerBlockSize
	n := min(int64(remoteInstallerBlockSize), r.size-start)
	rc, err := r.getRange(start, n)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	b := make([]byte, n)
	if _, err := io.ReadFull(rc, b); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// A short response is a storage error, not a truncated installer, so
			// don't let parsers mistake it for one.
			return nil, fmt.Errorf("reading installer bytes %d-%d: short response from installer store", start, start+n-1)
		}
		return nil, fmt.Errorf("reading installer bytes %d-%d: %w", start, start+n-1, err)
	}
	if len(r.blockOrder) >= remoteInstallerCacheBlocks {
		delete(r.blocks, r.blockOrder[0])
		r.blockOrder = r.blockOrder[1:]
	}
	r.blocks[idx] = b
	r.blockOrder = append(r.blockOrder, idx)
	return b, nil
}
