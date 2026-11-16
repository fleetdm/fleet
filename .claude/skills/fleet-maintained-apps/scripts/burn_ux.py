"""Extract a WiX Burn bundle's UX cabinet, which holds BurnManifest.xml.

The cabinet sits near the start of the bundle, so a range request is enough even
for a multi-GB installer:

    curl -sL -r 0-31457279 -o head.bin "<InstallerUrl>"
    python3 burn_ux.py head.bin ux.cab
    7zz x -y -oux ux.cab >/dev/null
    grep -o '<Arp [^>]*>' ux/0            # DisplayName, DisplayVersion, Publisher
    grep -o '<Registration [^>]*>' ux/0   # ExecutableName, PerMachine
"""
import struct
import sys

with open(sys.argv[1], "rb") as f:
    d = f.read()
pe = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, pe + 6)[0]
opt = struct.unpack_from("<H", d, pe + 20)[0]
sec = pe + 24 + opt
raw = None
for i in range(nsec):
    h = sec + 40 * i
    if d[h:h + 8].rstrip(b"\0") == b".wixburn":
        raw = struct.unpack_from("<I", d, h + 20)[0]
if raw is None:
    sys.exit("no .wixburn section: not a Burn bundle")
# Header after the 24-byte magic/version/bundle-GUID prefix: stub size, original
# checksum, signature offset, signature size, format, container count, UX size,
# attached container size.
stub, _, _, _, _, _, ux, _ = struct.unpack_from("<8I", d, raw + 24)
if stub + ux > len(d):
    sys.exit(f"need the first {stub + ux} bytes; fetch a larger range")
with open(sys.argv[2], "wb") as f:
    f.write(d[stub:stub + ux])
print(f"wrote UX cabinet ({ux} bytes) to {sys.argv[2]}")
