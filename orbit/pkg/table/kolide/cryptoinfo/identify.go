// Package cryptoinfo is designed to examine keys and certificates on
// disk, and return information about them. It is designed to work
// with dataflatten, and may eventually it may replace pkg/keyidentifier
package cryptoinfo

// identifierSignature is an internal type to denote the identification functions. It's
// used to add a small amount of clarity to the array of possible identifiers.
type identifierSignature func(data []byte, password string) (results []*KeyInfo, err error)

var defaultIdentifiers = []identifierSignature{
	tryP12,
	tryDer,
	tryPem,
}

// Identify examines a []byte and attempts to descern what
// cryptographic material is contained within.
func Identify(data []byte, password string) ([]*KeyInfo, error) {

	// Try the identifiers. Some future work might be to allow
	// callers to specify identifier order, or to try to discern
	// it from the file extension. But meanwhile, just try everything.
	for _, fn := range defaultIdentifiers {
		res, err := fn(data, password)
		if err == nil {
			return res, nil
		}
	}

	// If we can't parse anything, return nothing. It's not a fatal error, and it's
	// somewhat obvious from context that nothing was parsed.
	return nil, nil
}
