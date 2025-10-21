package notarize

// Status is an interface that can be implemented to receive status callbacks.
//
// All the methods in this interface must NOT block for too long or it'll
// block the notarization process.
type Status interface {
	// Submitting is called when the file is being submitted for notarization.
	Submitting()

	// Submitted is called when the file is submitted to Apple for notarization.
	// The arguments give you access to the requestUUID to query more information.
	Submitted(requestUUID string)

	// InfoStatus is called as the status of the submitted package changes.
	// The info argument contains additional information about the status.
	// Note that some fields in the info argument may not be populated, please
	// refer to the docs.
	InfoStatus(Info)

	// LogStatus is called as the status of the submitted package changes.
	LogStatus(Log)
}

// noopStatus implements Status and does nothing.
type noopStatus struct{}

func (noopStatus) Submitting()      {}
func (noopStatus) Submitted(string) {}
func (noopStatus) InfoStatus(Info)  {}
func (noopStatus) LogStatus(Log)    {}

// Assert that we always implement it
var _ Status = noopStatus{}
