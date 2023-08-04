package main

func main() {
	count := 2000000000
	x := 5678597658.7865789658765
	for i := 0; i < count; i++ {
		x = x / 1.00000005454
	}
	for i := 0; i < count; i++ {
		x = x * 1.00000005454
	}
}
