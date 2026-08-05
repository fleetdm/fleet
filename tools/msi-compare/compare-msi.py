#!/usr/bin/env python3
"""Compare two fleetd Windows installers (.msi) byte by byte, except known
build-time nondeterminism (timestamps and freshly generated GUIDs).

Two MSIs built from the same inputs are never bit-identical: WiX (and any
correct replacement) generates a fresh ProductCode, PackageCode, and
per-component GUIDs on every build, stamps creation timestamps in the
summary-information stream and in CAB directory entries, and the embedded
CAB's compressed bytes depend on the deflate implementation. This script
normalizes exactly that nondeterminism and requires everything else to match
byte for byte:

  1. Table list and stream list must match exactly.
  2. Every installer table (dumped with msidump as .idt) must match byte for
     byte after normalizing nondeterministic GUIDs. GUIDs that must remain
     stable across builds (UpgradeCode, fixed component GUIDs authored in
     main.wxs) are NOT normalized, so a regression in those is caught.
  3. The summary-information stream must match after masking the
     created/last-saved timestamps and normalizing the package code GUID.
  4. Binary table streams (e.g. the WixCA custom-action DLL) must be
     byte-identical.
  5. The embedded CAB(s) must contain the same entries in the same order
     with the same sizes (dates/times masked).
  6. Every payload file extracted from the installer must be byte-identical.

Exit code 0 when all checks pass, 1 otherwise.

Requires msitools (msiinfo, msidump, msiextract) and cabextract:
  brew install msitools cabextract

Usage: compare-msi.py [--keep] reference.msi candidate.msi
"""

import argparse
import difflib
import filecmp
import os
import re
import shutil
import subprocess
import sys
import tempfile

# GUIDs that are required to be identical in both MSIs (never normalized).
# From orbit/pkg/packaging/windows_templates.go (main.wxs template).
STABLE_GUIDS = {
    "B681CB20-107E-428A-9B14-2D3C1AFED244",  # UpgradeCode
    "A7DFD09E-2D2B-4535-A04F-5D4DE90F3863",  # Component C_ORBITROOT
    "AF347B4E-B84B-4DD4-9C4D-133BE17B613D",  # Component C_ORBITBIN
}

GUID_RE = re.compile(r"\{([0-9A-Fa-f]{8}(?:-[0-9A-Fa-f]{4}){3}-[0-9A-Fa-f]{12})\}")

failures = []


def check(name, ok, detail=""):
    status = "PASS" if ok else "FAIL"
    print(f"[{status}] {name}")
    if not ok:
        if detail:
            print(indent(detail))
        failures.append(name)


def indent(text, prefix="       "):
    return "\n".join(prefix + line for line in text.splitlines())


def run(cmd, **kwargs):
    return subprocess.run(cmd, check=True, capture_output=True, text=False, **kwargs).stdout


def unified_diff(a_text, b_text, a_name, b_name, limit=200):
    diff = list(
        difflib.unified_diff(
            a_text.splitlines(keepends=True),
            b_text.splitlines(keepends=True),
            fromfile=a_name,
            tofile=b_name,
        )
    )
    if len(diff) > limit:
        diff = diff[:limit] + [f"... diff truncated ({len(diff) - limit} more lines)\n"]
    return "".join(diff)


class GuidNormalizer:
    """Replaces nondeterministic GUIDs with positional placeholders.

    The mapping is per-MSI and keyed by order of first appearance across all
    artifacts, processed in a fixed order. If both MSIs introduce their fresh
    GUIDs at the same positions, the placeholders line up; any structural
    difference (e.g. one GUID reused where the other MSI uses two distinct
    ones) still shows up in the diff.
    """

    def __init__(self):
        self.mapping = {}

    def normalize(self, text):
        def repl(m):
            guid = m.group(1).upper()
            if guid in STABLE_GUIDS:
                return m.group(0)
            if guid not in self.mapping:
                self.mapping[guid] = f"{{GUID-{len(self.mapping) + 1:04d}}}"
            return self.mapping[guid]

        return GUID_RE.sub(repl, text)


class MSI:
    def __init__(self, path, workdir):
        self.path = path
        self.workdir = workdir
        self.guids = GuidNormalizer()
        os.makedirs(workdir)

    def tables(self):
        out = run(["msiinfo", "tables", self.path]).decode()
        return sorted(out.split())

    def streams(self):
        out = run(["msiinfo", "streams", self.path]).decode()
        return sorted(out.splitlines())

    def dump_tables(self):
        """Dump all tables as .idt files, return dir path."""
        d = os.path.join(self.workdir, "tables")
        os.makedirs(d)
        # cwd matters: msidump writes Binary stream dumps into the current
        # directory regardless of -d.
        run(["msidump", "-t", "-d", d, os.path.abspath(self.path)], cwd=d)
        return d

    def table_texts(self):
        """{table_file_name: guid-normalized idt text}, in sorted name order."""
        d = self.dump_tables()
        texts = {}
        for name in sorted(os.listdir(d)):
            path = os.path.join(d, name)
            if not name.endswith(".idt") or not os.path.isfile(path):
                continue  # msidump also writes binary stream dumps
            with open(path, encoding="utf-8", errors="replace") as f:
                texts[name] = self.guids.normalize(f.read())
        return texts

    def suminfo(self):
        out = run(["msiinfo", "suminfo", self.path]).decode()
        lines = []
        for line in out.splitlines():
            if line.startswith(("Created:", "Last saved:")):
                line = line.split(":", 1)[0] + ": <TIMESTAMP>"
            lines.append(line)
        return self.guids.normalize("\n".join(lines) + "\n")

    def extract_stream(self, name):
        return run(["msiinfo", "extract", self.path, name])

    def cabinet_names(self):
        """Embedded cabinet stream names from the Media table, in disk order."""
        media = None
        d = os.path.join(self.workdir, "tables")
        media_path = os.path.join(d, "Media.idt")
        if os.path.exists(media_path):
            with open(media_path, encoding="utf-8", errors="replace") as f:
                media = f.read()
        if media is None:
            return []
        cabs = []
        rows = media.splitlines()[3:]  # skip idt 3-line header
        for row in rows:
            cols = row.split("\t")
            if len(cols) >= 4 and cols[3].startswith("#"):
                cabs.append(cols[3][1:])
        return cabs

    def cab_listing(self, cab_name):
        """cabextract -l listing with dates/times masked; includes entry order."""
        cab_path = os.path.join(self.workdir, cab_name)
        with open(cab_path, "wb") as f:
            f.write(self.extract_stream(cab_name))
        out = run(["cabextract", "-l", cab_path]).decode()
        lines = []
        for line in out.splitlines():
            # "    123456 | 12.03.2024 10:11:12 | name"
            m = re.match(r"^(\s*\d+ \| )\S+ \S+( \| .*)$", line)
            if m:
                line = m.group(1) + "<DATE> <TIME>" + m.group(2)
            if line.startswith("Viewing cabinet:"):
                line = "Viewing cabinet: <PATH>"
            lines.append(line)
        return "\n".join(lines) + "\n"

    def extract_payload(self):
        d = os.path.join(self.workdir, "payload")
        os.makedirs(d)
        run(["msiextract", "-C", d, self.path])
        return d


def compare_trees(a_dir, b_dir):
    """Byte-compare two directory trees; returns list of difference strings."""
    diffs = []

    def walk(rel):
        a = os.path.join(a_dir, rel)
        b = os.path.join(b_dir, rel)
        a_entries = sorted(os.listdir(a)) if os.path.isdir(a) else None
        b_entries = sorted(os.listdir(b)) if os.path.isdir(b) else None
        if a_entries is None or b_entries is None:
            if a_entries is not None or b_entries is not None:
                diffs.append(f"type mismatch: {rel}")
                return
            if not filecmp.cmp(a, b, shallow=False):
                diffs.append(f"content differs: {rel} ({os.path.getsize(a)} vs {os.path.getsize(b)} bytes)")
            return
        for name in sorted(set(a_entries) - set(b_entries)):
            diffs.append(f"only in reference: {os.path.join(rel, name)}")
        for name in sorted(set(b_entries) - set(a_entries)):
            diffs.append(f"only in candidate: {os.path.join(rel, name)}")
        for name in sorted(set(a_entries) & set(b_entries)):
            walk(os.path.join(rel, name))

    walk("")
    return diffs


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--keep", action="store_true", help="keep the temporary work directory for inspection")
    parser.add_argument("reference", help="reference .msi (e.g. built by released fleetctl / WiX)")
    parser.add_argument("candidate", help="candidate .msi (e.g. built by the pure Go implementation)")
    args = parser.parse_args()

    for tool in ("msiinfo", "msidump", "msiextract", "cabextract"):
        if shutil.which(tool) is None:
            print(f"error: required tool not found on PATH: {tool}", file=sys.stderr)
            print("install with: brew install msitools cabextract", file=sys.stderr)
            return 2

    tmp = tempfile.mkdtemp(prefix="msi-compare-")
    print(f"work dir: {tmp}")
    ref = MSI(args.reference, os.path.join(tmp, "reference"))
    cand = MSI(args.candidate, os.path.join(tmp, "candidate"))

    try:
        # 1. Table and stream lists.
        rt, ct = ref.tables(), cand.tables()
        check("table list", rt == ct, unified_diff("\n".join(rt) + "\n", "\n".join(ct) + "\n", "reference", "candidate"))
        rs, cs = ref.streams(), cand.streams()
        check("stream list", rs == cs, unified_diff("\n".join(rs) + "\n", "\n".join(cs) + "\n", "reference", "candidate"))

        # 2. Table contents (GUID-normalized, otherwise byte-exact).
        ref_tables = ref.table_texts()
        cand_tables = cand.table_texts()
        for name in sorted(set(ref_tables) | set(cand_tables)):
            if name == "_SummaryInformation.idt":
                continue  # compared via suminfo below (needs timestamp masking)
            a = ref_tables.get(name)
            b = cand_tables.get(name)
            if a is None or b is None:
                check(f"table {name}", False, "missing in " + ("reference" if a is None else "candidate"))
                continue
            check(f"table {name}", a == b, unified_diff(a, b, f"reference/{name}", f"candidate/{name}"))

        # 3. Summary information (timestamps masked, package code normalized).
        a, b = ref.suminfo(), cand.suminfo()
        check("summary information", a == b, unified_diff(a, b, "reference/suminfo", "candidate/suminfo"))

        # 4. Binary streams must be byte-identical.
        binary_streams = [s for s in rs if s.startswith("Binary.")]
        for s in binary_streams:
            if s not in cs:
                continue  # already reported by stream list check
            same = ref.extract_stream(s) == cand.extract_stream(s)
            check(f"binary stream {s}", same, "binary content differs")

        # 5. Embedded CAB structure (entry order, names, sizes; dates masked).
        ref_cabs, cand_cabs = ref.cabinet_names(), cand.cabinet_names()
        check(
            "cabinet names",
            ref_cabs == cand_cabs,
            f"reference: {ref_cabs}\ncandidate: {cand_cabs}",
        )
        if ref_cabs == cand_cabs:
            for cab in ref_cabs:
                a, b = ref.cab_listing(cab), cand.cab_listing(cab)
                check(f"cabinet structure {cab}", a == b, unified_diff(a, b, f"reference/{cab}", f"candidate/{cab}"))

        # 6. Payload files byte-by-byte.
        diffs = compare_trees(ref.extract_payload(), cand.extract_payload())
        check("payload files", not diffs, "\n".join(diffs))
    finally:
        if args.keep:
            print(f"keeping work dir: {tmp}")
        else:
            shutil.rmtree(tmp, ignore_errors=True)

    if failures:
        print(f"\n{len(failures)} check(s) FAILED")
        return 1
    print("\nAll checks passed: installers are equivalent (modulo timestamps and fresh GUIDs)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
