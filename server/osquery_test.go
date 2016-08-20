package server

type MockOsqueryResultHandler struct{}

func (h *MockOsqueryResultHandler) HandleResultLog(log OsqueryResultLog, nodeKey string) error {
	return nil
}

type MockOsqueryStatusHandler struct{}

func (h *MockOsqueryStatusHandler) HandleStatusLog(log OsqueryStatusLog, nodeKey string) error {
	return nil
}
