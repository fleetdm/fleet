#!/bin/bash

# Generate gnuplot commands to render CPU and memory data points
# from a cpu_and_mem.dat file under $DB_DIR.
SAMPLE_PATH=$1

cat <<EOF > gnuplot_commands.txt
set xdata time
set timefmt "%H:%M:%S"
set format x "%H:%M"
set xtics rotate by -45
set terminal jpeg

set ylabel '% CPU'
set yrange [0:10]
set ytics nomirror

set y2label 'Memory (MB)'
set y2range [0:1024]
set y2tics 0, 100

set output 'output.jpg'

plot '$SAMPLE_PATH' using 1:2 axis x1y1 with linespoints linetype -1 linecolor rgb 'blue' linewidth 1 title '% CPU', \
     '$SAMPLE_PATH' using 1:3 axis x1y2 with linespoints linetype -1 linecolor rgb 'red' linewidth 1 title 'Memory (MB)'
EOF

gnuplot < gnuplot_commands.txt
rm gnuplot_commands.txt

open output.jpg
