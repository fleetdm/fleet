package fleet

import "time"

type PKICertificate struct {
	Name          string     `json:"name" db:"name"`
	Cert          []byte     `json:"-" db:"cert_pem"`
	Key           []byte     `json:"-" db:"key_pem"`
	Sha256        *string    `json:"sha256" db:"sha256_hex"`
	NotValidAfter *time.Time `json:"not_valid_after" db:"not_valid_after"`
}
