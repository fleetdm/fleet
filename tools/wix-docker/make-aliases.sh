#!/bin/sh

# This script creates shell scripts that simulate adding all of the WiX binaries
# to the PATH. `wine /home/wine/wix/light.exe will be able to be called with
# just `light`.

mkdir -p /home/wine/bin
binpath=/home/wine/bin

for exe in $(ls /home/wine/wix | grep .exe$); do
    name=$(echo $exe | cut -d '.' -f 1)

    cat > $binpath/$name << EOF
#!/bin/sh
wine /home/wine/wix/$exe \$@
EOF
    chmod +x $binpath/$name
done