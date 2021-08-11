package osquery_utils

// EmptyToZero Sometimes osquery gives us empty string where we expect an integer.
// We change the to "0" so it can be handled by the appropriate string to
// integer conversion function, as these will err on ""
func EmptyToZero(val string) string {
	if val == "" {
		return "0"
	}
	return val
}
