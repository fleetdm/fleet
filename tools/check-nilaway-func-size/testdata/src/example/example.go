package example

// tooBig branches enough times to exceed the threshold the test sets.
func tooBig(a, b, c, d int) int { // want "tooBig has \\d+ CFG blocks, over the limit of 5"
	if a > 0 {
		a++
	}
	if b > 0 {
		b++
	}
	if c > 0 {
		c++
	}
	if d > 0 {
		d++
	}
	return a + b + c + d
}

// smallEnough stays under the threshold and must not be reported.
func smallEnough(a int) int {
	if a > 0 {
		return a
	}
	return -a
}

// closuresAreNotGated keeps its own CFG small while nesting a heavily branching function literal.
// nilaway only size-checks literals under its experimental-anonymous-function flag, which Fleet does
// not enable, so this must not be reported.
func closuresAreNotGated() func(int) int {
	return func(n int) int {
		if n > 1 {
			n++
		}
		if n > 2 {
			n++
		}
		if n > 3 {
			n++
		}
		if n > 4 {
			n++
		}
		if n > 5 {
			n++
		}
		return n
	}
}
