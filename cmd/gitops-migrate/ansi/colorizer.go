package ansi

// Colorizer is a simple closure generator. You stick a color in and get back
// a closure which accepts a string and wraps that string in the ANSI color
// specified.
func Colorizer(color string) func(string) string {
	return func(s string) string {
		return color + s + Reset
	}
}
