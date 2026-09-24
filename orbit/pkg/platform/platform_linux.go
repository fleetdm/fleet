//go:build linux

package platform

import (
	gopsutil_process "github.com/shirou/gopsutil/v4/process"
)

func init() {
	// Enable gopsutil's boot-time cache (Linux only).
	//
	// gopsutil_process.Processes() builds a Process for every PID, and the
	// constructor (NewProcess -> CreateTime -> fillFromStat) computes each
	// process' creation time, which requires the system boot time. By default
	// gopsutil does NOT cache the boot time, so on Linux it re-reads /proc/stat
	// (or /proc/uptime on containerized hosts) once per process, on every call.
	//
	// orbit enumerates the full process table on a recurring basis (e.g. the
	// Fleet Desktop watchdog polls every 15s via GetProcessesByName), so this
	// caused the host-wide /proc/stat file to be read N times per poll, where N
	// is the total number of running processes. The btime field lives near the
	// end of /proc/stat, so each read scans the entire file just to recover a
	// single constant value.
	//
	// The system boot time does not change for the lifetime of the orbit
	// process, so caching it is safe and collapses those repeated reads into a
	// single one. This is scoped to Linux because that is where the redundant
	// file reads occur; macOS and Windows obtain boot time via syscall/sysctl.
	//
	// Note: orbit only reads process Name/Pid and never a process' CreateTime,
	// so the cache cannot surface a stale value (the gopsutil README warns that
	// a cached boot time can drift if NTP steps the clock after boot, which only
	// affects CreateTime).
	gopsutil_process.EnableBootTimeCache(true)
}
