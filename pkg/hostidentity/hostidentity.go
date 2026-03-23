package hostidentity

import "encoding/asn1"

// RenewalData represents the JSON data in the renewal extension
type RenewalData struct {
	SerialNumber string `json:"sn"`  // Hex-encoded serial number of the old certificate
	Signature    string `json:"sig"` // Base64-encoded ECDSA signature
}

// RenewalExtensionOID is the custom OID for the renewal extension. 63991 is Fleet's IANA private enterprise number
// 1.3.6.1.4.1.63991.1.1
var RenewalExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 63991, 1, 1}
