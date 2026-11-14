"""Print a Windows installer's CPU architecture and VERSIONINFO ProductVersion.

Usage: pe_info.py <file>   (the first few MB of the installer is usually enough:
       curl -sL -r 0-4194303 -o head.bin <url>)
"""
import re
import struct
import sys

MACHINES = {0x14C: "x86", 0x8664: "x64", 0xAA64: "arm64"}

with open(sys.argv[1], "rb") as f:
    d = f.read()
if d[:2] != b"MZ":
    sys.exit("not a PE file")
pe = struct.unpack_from("<I", d, 0x3C)[0]
machine = struct.unpack_from("<H", d, pe + 4)[0]
print(f"machine: {MACHINES.get(machine, hex(machine))}")

for key in ("ProductVersion", "FileVersion", "ProductName", "CompanyName"):
    k = key.encode("utf-16-le")
    m = re.search(re.escape(k) + rb"\x00\x00", d)
    if not m:
        print(f"{key}: (not in this range)")
        continue
    t = d[m.end():m.end() + 512]
    # Values are UTF-16LE after zero padding; walk in 2-byte units so the last
    # character isn't cut off by a misaligned terminator match.
    i = 0
    while i < len(t) - 1 and t[i:i + 2] == b"\x00\x00":
        i += 2
    j = i
    while j < len(t) - 1 and t[j:j + 2] != b"\x00\x00":
        j += 2
    print(f"{key}: {t[i:j].decode('utf-16-le', 'ignore')}")
