package client

// uintValueOrZero returns the uint pointed to by v, or 0 if v is nil.
func uintValueOrZero(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}
