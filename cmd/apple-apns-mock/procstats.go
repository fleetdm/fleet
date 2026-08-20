package main

import (
	"bufio"
	"os"
	"runtime/metrics"
	"strconv"
	"strings"
	"syscall"
)

// openFDs reports how many file descriptors the process holds, or -1 where
// that cannot be determined (anywhere without /proc, so everywhere but Linux).
//
// One descriptor per live SSE stream makes this the number that decides how
// many more connections the process can take, and nothing else reports it: a
// container that has run out of descriptors holds its memory and CPU flat
// while refusing every new connection, which looks like a healthy idle
// process from the outside.
func openFDs() int {
	f, err := os.Open("/proc/self/fd")
	if err != nil {
		return -1
	}
	defer f.Close()

	// Counted in chunks rather than with os.ReadDir so that 150k descriptors
	// don't turn this into a 150k-element allocation on every call.
	n := 0
	for {
		names, err := f.Readdirnames(4096)
		n += len(names)
		if err != nil || len(names) == 0 {
			break
		}
	}
	return n - 1 // f itself appears in the listing
}

// fdLimit reports the soft RLIMIT_NOFILE the process is actually running
// under, which is the ceiling openFDs climbs toward. Worth logging next to the
// count because the limit a task definition asks for and the limit the
// platform grants are not always the same number, and the difference is
// invisible until connections start failing.
func fdLimit() uint64 {
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		return 0
	}
	return uint64(rl.Cur)
}

// runtimeMem reports what the Go runtime knows it is using. Deliberately not
// RSS: on macOS pages the runtime has already released still count against the
// process, which overstated a 40k-connection run by 3x.
//
// None of this covers the kernel's per-socket buffers, which are charged to
// the same container memory limit at roughly a gigabyte per 150k sockets and
// are something the Go runtime can neither see nor shrink. A task that looks
// half idle by these numbers can still be close to its cgroup ceiling.
func runtimeMem() (heap, stacks, inUse uint64) {
	samples := []metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(samples)
	return samples[0].Value.Uint64(),
		samples[1].Value.Uint64(),
		samples[2].Value.Uint64() - samples[3].Value.Uint64()
}

// osThreads reports how many OS threads the process has, or -1 where that
// cannot be determined. Not the same thing as the goroutine count, and the one
// the Go runtime will kill the process over: exceeding its 10,000-thread limit
// is a fatal error, not a panic, and it happens with CPU and memory both flat
// because threads parked in syscalls burn neither. The runtime exposes no way
// to read this, so it comes from /proc.
func osThreads() int {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return -1
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		rest, ok := strings.CutPrefix(sc.Text(), "Threads:")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return -1
		}
		return n
	}
	return -1
}
