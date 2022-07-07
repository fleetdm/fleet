package oval_input

// rpmVerifyFileBehaviors see
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#RpmVerifyFileBehaviors
type rpmVerifyFileBehaviors struct {
	NoLinkTo      bool `xml:"nolinkto,attr"`
	NoMd5         bool `xml:"nomd5,attr"`
	NoSize        bool `xml:"nosize,attr"`
	NoUser        bool `xml:"nouser,attr"`
	NoGroup       bool `xml:"nogroup,attr"`
	NoMtime       bool `xml:"nomtime,attr"`
	NoMode        bool `xml:"nomode,attr"`
	NoRev         bool `xml:"nordev,attr"`
	NoConfigFiles bool `xml:"noconfigfiles,attr"`
	NoGhostFiles  bool `xml:"noghostfiles,attr"`
}

// RpmVerifyFileObjectXML see
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#rpmverifyfile_object
type RpmVerifyFileObjectXML struct {
	Id        string                  `xml:"id,attr"`
	Behaviors *rpmVerifyFileBehaviors `xml:"behaviors"`
	Name      SimpleTypeXML           `xml:"name"`
	Epoch     SimpleTypeXML           `xml:"epoch"`
	Version   SimpleTypeXML           `xml:"version"`
	Release   SimpleTypeXML           `xml:"release"`
	Arch      SimpleTypeXML           `xml:"arch"`
	FilePath  SimpleTypeXML           `xml:"filepath"`
}
