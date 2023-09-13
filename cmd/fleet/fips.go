//go:build boringcrypto
// +build boringcrypto

//
// This is used to force TLSv1.2 and FIPS ciphers regardless of the runtime settings.
//

package main

import _ "crypto/tls/fipsonly"
