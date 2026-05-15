package update

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"path"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"
)

// Note(roberto): most of the code below has been taken from the go-tuf repo
// test suite with minor adaptations to suit our needs.

func newFakeRemoteStore() *fakeRemoteStore {
	return &fakeRemoteStore{
		meta:    make(map[string]*fakeFile),
		targets: make(map[string]*fakeFile),
	}
}

type fakeRemoteStore struct {
	meta    map[string]*fakeFile
	targets map[string]*fakeFile
}

func (f *fakeRemoteStore) GetMeta(name string) (io.ReadCloser, int64, error) {
	return f.get(name, f.meta)
}

func (f *fakeRemoteStore) GetTarget(path string) (io.ReadCloser, int64, error) {
	return f.get(path, f.targets)
}

func (f *fakeRemoteStore) get(name string, store map[string]*fakeFile) (io.ReadCloser, int64, error) {
	file, ok := store[name]
	if !ok {
		return nil, 0, client.ErrNotFound{File: name}
	}
	return file, file.size, nil
}

func newFakeFile(b []byte) *fakeFile {
	return &fakeFile{buf: bytes.NewReader(b), size: int64(len(b))}
}

type fakeFile struct {
	buf       *bytes.Reader
	bytesRead int
	size      int64
}

func (f *fakeFile) Read(p []byte) (int, error) {
	n, err := f.buf.Read(p)
	f.bytesRead += n
	return n, err
}

func (f *fakeFile) Close() error {
	_, err := f.buf.Seek(0, io.SeekStart)
	return err
}

type withTUF struct {
	store       tuf.LocalStore
	repo        *tuf.Repo
	remote      *fakeRemoteStore
	expiredTime time.Time
	keyIDs      map[string][]string
	local       client.LocalStore
	client      *client.Client
	mockFiles   map[string][]byte

	s *suite.Suite
}

func (ts *withTUF) SetupSuite() {
	t := ts.s.T()
	ts.mockFiles = map[string][]byte{
		"nudge/macos/stable/nudge.app.tar.gz":       ts.memTarGz("/Nudge.app/Contents/MacOS/Nudge", "nudge"),
		"osqueryd/macos/stable/osqueryd.app.tar.gz": ts.memTarGz("osqueryd", "osqueryd"),
		"escrowBuddy/macos/stable/escrowBuddy.pkg":  {},
	}
	ts.store = tuf.MemoryStore(nil, ts.mockFiles)

	var err error
	ts.repo, err = tuf.NewRepo(ts.store)
	require.NoError(t, err)

	require.NoError(t, ts.repo.Init(false))
	ts.keyIDs = map[string][]string{
		"root":      ts.genKey("root"),
		"targets":   ts.genKey("targets"),
		"snapshot":  ts.genKey("snapshot"),
		"timestamp": ts.genKey("timestamp"),
	}

	ts.remote = newFakeRemoteStore()
	ts.client = ts.newClient()
	ts.addRemoteTarget("osqueryd/macos/stable/osqueryd.app.tar.gz")
	ts.expiredTime = time.Now().Add(time.Hour)
}

func (ts *withTUF) addRemoteTarget(filePath string) {
	t := ts.s.T()
	require.NoError(t, ts.repo.AddTarget(filePath, nil))
	require.NoError(t, ts.repo.Snapshot())
	require.NoError(t, ts.repo.Timestamp())
	require.NoError(t, ts.repo.Commit())
	ts.syncRemote()
	ts.remote.targets[filePath] = newFakeFile(ts.mockFiles[filePath])
	ts.syncLocal()
}

func (ts *withTUF) genKey(role string) []string {
	ids, err := ts.repo.GenKey(role)
	require.NoError(ts.s.T(), err)
	return ids
}

func (ts *withTUF) syncLocal() {
	t := ts.s.T()
	meta, err := ts.store.GetMeta()
	require.NoError(t, err)
	for k, v := range meta {
		require.NoError(t, ts.local.SetMeta(k, v))
	}
}

func (ts *withTUF) syncRemote() {
	meta, err := ts.store.GetMeta()
	require.NoError(ts.s.T(), err)
	for name, data := range meta {
		ts.remote.meta[name] = newFakeFile(data)
	}
}

func (ts *withTUF) rootMeta() []byte {
	meta, err := ts.repo.GetMeta()
	require.NoError(ts.s.T(), err)
	rootMeta, ok := meta["root.json"]
	require.True(ts.s.T(), ok)
	return rootMeta
}

func (ts *withTUF) newClient() *client.Client {
	ts.local = client.MemoryLocalStore()
	client := client.NewClient(ts.local, ts.remote)
	require.NoError(ts.s.T(), client.Init(ts.rootMeta()))
	return client
}

func (ts *withTUF) memTarGz(name, contents string) []byte {
	t := ts.s.T()
	out := bytes.NewBuffer([]byte{})
	s := bufio.NewWriter(out)
	gw := gzip.NewWriter(s)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	if path.Dir(name) != name {
		err := tw.WriteHeader(
			&tar.Header{
				Name: path.Dir(name) + "/",
			},
		)
		require.NoError(t, err)
	}
	err := tw.WriteHeader(
		&tar.Header{
			Name: name,
			Size: int64(len(contents)),
		},
	)
	require.NoError(t, err)
	_, err = io.Copy(tw, bytes.NewReader([]byte(contents)))
	require.NoError(t, err)
	err = tw.Close()
	require.NoError(t, err)
	err = gw.Close()
	require.NoError(t, err)
	err = s.Flush()
	require.NoError(t, err)
	return out.Bytes()
}
