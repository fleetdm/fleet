#!/bin/bash

DB_DIR=$1

# Generate gnuplot commands to render CPU and memory data points
# from a cpu_and_mem.dat file under $DB_DIR.

cat <<EOF > gnuplot_commands.txt
set xdata time
set timefmt "%H:%M:%S"
set format x "%H:%M"
set xtics rotate by -45
set terminal jpeg

set ylabel '% CPU'
set yrange [0:800]
set ytics nomirror

set y2label 'Memory (MB)'
set y2range [0:5000]
set y2tics 0, 500

set output '$DB_DIR/cpu_and_mem.jpg'

plot '$DB_DIR/cpu_and_mem.dat' using 1:2 axis x1y1 with linespoints linetype -1 linecolor rgb 'blue' linewidth 1 title '% CPU', \
     '$DB_DIR/cpu_and_mem.dat' using 1:3 axis x1y2 with linespoints linetype -1 linecolor rgb 'red' linewidth 1 title 'Memory (MB)'
EOF

gnuplot < gnuplot_commands.txt
rm gnuplot_commands.txt

open $DB_DIR/cpu_and_mem.jpg
