package io

import "context"

type TestData struct {
	RemoteList          map[MetadataFileName]string
	RemoteListError     error
	RemoteDownloaded    []string
	RemoteDownloadError error
	LocalList           []MetadataFileName
	LocalListError      error
	LocalDeleted        []MetadataFileName
	LocalDeleteError    error
}

type GhMock struct{ TestData *TestData }

func (gh GhMock) MSRCBulletins(ctx context.Context) (map[MetadataFileName]string, error) {
	return gh.TestData.RemoteList, gh.TestData.RemoteListError
}

func (gh GhMock) MacOfficeReleaseNotes(ctx context.Context) (MetadataFileName, string, error) {
	for k, v := range gh.TestData.RemoteList {
		return k, v, gh.TestData.RemoteListError
	}
	return MetadataFileName{}, "", gh.TestData.RemoteListError
}

func (gh GhMock) Download(url string) (string, error) {
	gh.TestData.RemoteDownloaded = append(gh.TestData.RemoteDownloaded, url)
	return "", gh.TestData.RemoteDownloadError
}

type FsMock struct{ TestData *TestData }

func (fs FsMock) MSRCBulletins() ([]MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs FsMock) MacOfficeReleaseNotes() ([]MetadataFileName, error) {
	return fs.TestData.LocalList, fs.TestData.LocalListError
}

func (fs FsMock) Delete(d MetadataFileName) error {
	fs.TestData.LocalDeleted = append(fs.TestData.LocalDeleted, d)
	return fs.TestData.LocalDeleteError
}
