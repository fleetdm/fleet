package oval_input

// RpmVerifyFileStateXML see
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpmverifyfile_state
type RpmVerifyFileStateXML struct {
	Id                 string         `xml:"id,attr"`
	Name               *SimpleTypeXML `xml:"name"`
	Epoch              *SimpleTypeXML `xml:"epoch"`
	Version            *SimpleTypeXML `xml:"version"`
	Arch               *SimpleTypeXML `xml:"arch"`
	Filepath           *SimpleTypeXML `xml:"filepath"`
	ExtendedName       *SimpleTypeXML `xml:"extended_name"`
	SizeDiffers        *SimpleTypeXML `xml:"size_differs"`
	ModeDiffers        *SimpleTypeXML `xml:"mode_differs"`
	Md5Differs         *SimpleTypeXML `xml:"md5_differs"`
	DeviceDiffers      *SimpleTypeXML `xml:"device_differs"`
	LinkMismatch       *SimpleTypeXML `xml:"link_mismatch"`
	OwnershipDiffers   *SimpleTypeXML `xml:"ownership_differs"`
	GroupDiffers       *SimpleTypeXML `xml:"group_differs"`
	MtimeDiffers       *SimpleTypeXML `xml:"mtime_differs"`
	CapabilitiesDiffer *SimpleTypeXML `xml:"capabilities_differ"`
	ConfigurationFile  *SimpleTypeXML `xml:"configuration_file"`
	GhostFile          *SimpleTypeXML `xml:"ghost_file"`
	LicenseFile        *SimpleTypeXML `xml:"license_file"`
	ReadmeFile         *SimpleTypeXML `xml:"readme_file	"`
	Operator           *string        `xml:"operator,attr"`
}
