#!/bin/bash

set -e

# Requirements:
#   - ripgrep
#   - gnuplot 

# Get PID of the osquery worker process.
if [ -n "$OSQUERYD_PID" ]; then
	osquery_pid=$OSQUERYD_PID
else
	osquery_pid=$(ps aux | grep -E "osqueryd\s*$" | awk {'print $2'})
fi

# Extract CPU and memory data points from logs.
rg " (\d\d:\d\d:\d\d).* pid: $osquery_pid, cpu: (\d+)ms/\d+ms, memory: ([\d.]+)" -or '$1 $2 $3' /tmp/osqueryd.log > /tmp/osqueryd.dat

# Generate gnuplot commands and render CPU and memory data points.
cat <<EOF > gnuplot_commands.txt
set xdata time
set timefmt "%H:%M:%S"
set format x "%H:%M"
set key off
set xtics rotate by -45
set terminal jpeg

set title 'Memory (MB)'
set output 'osquery_worker_memory.jpg'
plot '/tmp/osqueryd.dat' using 1:3 with linespoints linetype -1 linewidth 1 title 'Memory (MB)'

set title 'CPU'
set output 'osquery_worker_cpu.jpg'
set yrange [0:24000]
#
# The calculation used by osquery for CPU limit is:
# check_interval * number_of_physical_cores * (percent_cpu_limit / 100)
# where default values are: check_interval=3000ms, percent_cpu_limit=10%.
# On my Macbook with 4 physical core this gives 1200ms.
#
plot '/tmp/osqueryd.dat' using 1:2 with linespoints linetype -1 linewidth 1 title 'CPU', 1200 linecolor 1
EOF

gnuplot < gnuplot_commands.txt
rm gnuplot_commands.txt

open osquery_worker_cpu.jpg osquery_worker_memory.jpg
