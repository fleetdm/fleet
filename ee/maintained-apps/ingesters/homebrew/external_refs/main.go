package externalrefs

var Funcs = map[string]func(...interface{}) (string, error){
	"MicrosoftVersionFromReleaseNotes": MicrosoftVersionFromReleaseNotes,
}
