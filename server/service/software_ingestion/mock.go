package software_ingestion

//go:generate go run ../../mock/mockimpl/impl.go -o datastore_mock.go "ds *MockSoftwareDatastore" "SoftwareDatastore"

// Mock declarations for the SoftwareIngestionService interfaces
// This ensures we have proper mocks for testing the service

var _ SoftwareDatastore = (*MockSoftwareDatastore)(nil)
var _ SoftwareIngestionService = (*MockSoftwareIngestionService)(nil)