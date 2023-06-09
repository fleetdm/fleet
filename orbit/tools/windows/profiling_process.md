# Process profiling

To measure the CPU and memory consumption of a process in Windows, you can use the
`process_profile_tool.ps1` tool. 

To use this tool, you just need to pass the process ID (PID) as an argument through the `TargetPID`
parameter. The script will run and take samples every 10 seconds by default (this can be adjusted
using the `SampleIntervalInSecs"` parameter). The script will stop taking samples either when the
process finishes or when a key is pressed while the script is running. As a result, the script will
generate two PNG files in the same directory, showing the CPU and memory consumption of the process.

Example execution below:
```
PS C:\code\PoCs> . .\process_profile.ps1 -TargetPID 32652
Key Pressed. Stopping script...
PS C:\code\PoCs>
```