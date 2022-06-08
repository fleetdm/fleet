package oval_parsed

type DataType int

const (
	Binary DataType = iota
	Boolean
	EvrString
	FilesetRevision
	Float
	IosVersion
	Int
	Ipv4Address
	Ipv6Address
	String
	Version
)

// NewDataType encodes a 'SimpleDataTypeEnumeration' into an int.
// See: https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#SimpleDatatypeEnumeration
func NewDataType(val string) DataType {
	switch val {
	case "binary":
		return Binary
	case "boolean":
		return Boolean
	case "evr_string":
		return EvrString
	case "fileset_revision":
		return FilesetRevision
	case "float":
		return Float
	case "ios_version":
		return IosVersion
	case "int":
		return Int
	case "ipv4_address":
		return Ipv4Address
	case "ipv6_address":
		return Ipv6Address
	case "string":
		return String
	case "version":
		return Version
	default:
		return String
	}
}
