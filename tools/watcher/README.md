# watcher

This tool allows sampling a process CPU and memory usage.

E.g. following are the steps to sample the orbit process on linux.

1 Build the tool:
```
GOOS=linux go build ./tools/watcher
```
2. Move the built tool into the linux host (`watcher` executable).
3. Start sampling:
```sh
sudo ./watcher -name orbit -sample_path ./sample.txt
```
4. Copy the sample.txt to your macOS workstation and generate some plots running:
```sh
./tools/watcher/gnuplot.sh ./sample.txt
```
