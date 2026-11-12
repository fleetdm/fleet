"""Run an FMA output's exists/patched/open queries through real osquery.

Also runs "patched@bumped": the patched query with every version_compare
literal raised past anything real. With the app installed it must return 0
rows, or the patch policy can never flag an outdated host.

  darwin output, on this Mac:
    python3 -I check_queries.py ee/maintained-apps/outputs/<app>/darwin.json
    # Test a bundle you haven't installed (a mounted DMG, an extracted zip):
    python3 -I check_queries.py <output> --app-path "/Volumes/X/App.app"

  windows output: writes a .ps1 to run as SYSTEM on a Windows host or VM:
    python3 -I check_queries.py ee/maintained-apps/outputs/<app>/windows.json --ps1 check.ps1
    # then on the host: powershell -NoProfile -ExecutionPolicy Bypass -File check.ps1

--app-path constrains every `FROM apps` to that path, so nothing has to be in
/Applications (an enrolled Mac reports what's there to Fleet).
"""
import argparse
import json
import os
import re
import subprocess
import sys

DEFAULT_OSQUERYD = "/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd"


def skip_quoted(s, i):
    """Return the index of the quote that closes the literal opening at s[i]."""
    while True:
        i = s.index("'", i + 1)
        if i + 1 < len(s) and s[i + 1] == "'":  # '' is an escaped quote
            i += 1
            continue
        return i


def split_args(s, start):
    """Split the call arguments that begin at s[start] (just past "(").

    Returns (args, index just past the closing paren). Quote- and paren-aware,
    so rtrim(x, '.0') and regex_match(x, 'a,b', 1) stay one argument.
    """
    args, depth, i, cur = [], 0, start, start
    while i < len(s):
        c = s[i]
        if c == "'":
            i = skip_quoted(s, i)
        elif c == "(":
            depth += 1
        elif c == ")":
            if depth == 0:
                args.append(s[cur:i])
                return args, i + 1
            depth -= 1
        elif c == "," and depth == 0:
            args.append(s[cur:i])
            cur = i + 1
        i += 1
    raise ValueError("unbalanced parentheses in query")


def bump(version):
    m = re.match(r"^(\D*)(\d+)(.*)$", version)
    return f"{m.group(1)}{int(m.group(2)) + 1000}{m.group(3)}" if m else None


def bump_patched(sql):
    """Return (bumped sql, [(column expr, old, new)]) for each version_compare literal."""
    out, changes, pos = [], [], 0
    for m in re.finditer(r"version_compare\s*\(", sql, re.I):
        if m.start() < pos:
            continue
        args, end = split_args(sql, m.end())
        lit = re.fullmatch(r"'([^']*)'", args[-1].strip()) if len(args) == 2 else None
        if not lit or bump(lit.group(1)) is None:
            continue
        new = bump(lit.group(1))
        out.append(sql[pos:m.end()] + args[0] + ", '" + new + "')")
        changes.append((args[0].strip(), lit.group(1), new))
        pos = end
    out.append(sql[pos:])
    return "".join(out), changes


def clause_end(s, i):
    """Index where the clause starting at s[i] ends: an unmatched ")", a ";", or the end."""
    depth = 0
    while i < len(s):
        c = s[i]
        if c == "'":
            i = skip_quoted(s, i)
        elif c == "(":
            depth += 1
        elif c == ")":
            if depth == 0:
                return i
            depth -= 1
        elif c == ";" and depth == 0:
            return i
        i += 1
    return i


def constrain_apps(sql, paths):
    """Inject `path IN (...)` into every WHERE clause that reads the apps table."""
    in_list = ", ".join("'" + p.replace("'", "''") + "'" for p in paths)
    # A JOIN would make SQLite rerun the apps generator once per process row.
    sql = re.sub(r"\bFROM\s+apps\s+(\w+)\s+JOIN\b", r"FROM apps \1 CROSS JOIN", sql, flags=re.I)
    result, pos = [], 0
    for m in re.finditer(r"\bFROM\s+apps\b(?:\s+(?!WHERE\b|CROSS\b|JOIN\b)(\w+))?", sql, re.I):
        if m.start() < pos:
            continue
        # This SELECT's WHERE sits at the same paren depth, before its subquery closes.
        end = clause_end(sql, m.end())
        where = None
        depth, i = 0, m.end()
        while i < end:
            c = sql[i]
            if c == "'":
                i = skip_quoted(sql, i)
            elif c == "(":
                depth += 1
            elif c == ")":
                depth -= 1
            elif depth == 0 and re.match(r"WHERE\b", sql[i:], re.I):
                where = i + len("WHERE")
                break
            i += 1
        if where is None:
            continue
        col = f"{m.group(1)}.path" if m.group(1) else "path"
        result.append(sql[pos:where] + f" {col} IN ({in_list}) AND (" + sql[where:end] + ")")
        pos = end
    result.append(sql[pos:])
    return "".join(result)


def run_osquery(osqueryd, sql):
    try:
        p = subprocess.run(
            [osqueryd, "-S", "--disable_extensions", "--json", sql],
            capture_output=True, text=True, timeout=180,
        )
    except subprocess.TimeoutExpired:
        return "ERROR: timed out"
    if p.returncode != 0:
        msg = (p.stderr or p.stdout).strip().splitlines()
        return "ERROR: " + (msg[-1] if msg else f"exit {p.returncode}")
    try:
        return len(json.loads(p.stdout))
    except json.JSONDecodeError:
        return "ERROR: unparseable output: " + p.stdout.strip()[:200]


def ps_here_string(sql):
    # Windows PowerShell 5.1 passes an embedded " to a native exe only as \".
    return "@'\n" + sql.replace('"', '\\"') + "\n'@"


PS1_HEADER = r"""# Generated by .claude/skills/new-fma/scripts/check_queries.py. Run as SYSTEM:
#   powershell -NoProfile -ExecutionPolicy Bypass -File <this file>
$osq = 'C:\Program Files\osquery\osqueryi.exe'
$osqArgs = @('--json')
if (-not (Test-Path -LiteralPath $osq)) { $osq = (Get-Command osqueryi.exe -ErrorAction SilentlyContinue).Source }
if (-not $osq) {
    # A host with only fleetd has osqueryd, which runs ad hoc queries in shell mode.
    $osq = Get-ChildItem 'C:\Program Files\Orbit\bin\osqueryd\windows\*\osqueryd.exe' -ErrorAction SilentlyContinue |
        Select-Object -First 1 -ExpandProperty FullName
    $osqArgs = @('-S', '--disable_extensions', '--json')
}
if (-not $osq) { Write-Host 'Neither osqueryi.exe nor fleetd''s osqueryd.exe was found'; Exit 2 }
function Get-Rows([string]$Sql) {
    $out = & $osq @osqArgs $Sql 2>&1
    if ($LASTEXITCODE -ne 0) { return "ERROR: $($out -join ' ')" }
    $text = $out | Out-String
    # An empty result prints "[", blank lines, "]", which ConvertFrom-Json turns into one $null.
    if (($text -replace '\s', '') -eq '[]') { return 0 }
    return @($text | ConvertFrom-Json | Where-Object { $null -ne $_ }).Count
}
$queries = [ordered]@{}
"""


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("output", help="ee/maintained-apps/outputs/<app>/<platform>.json")
    ap.add_argument("--app-path", action="append", default=[], help="constrain the apps table to this bundle (repeatable)")
    ap.add_argument("--osqueryd", default=os.environ.get("OSQ", DEFAULT_OSQUERYD))
    ap.add_argument("--ps1", help="windows outputs: write a PowerShell checker here instead of running")
    a = ap.parse_args()

    with open(a.output, encoding="utf-8") as f:
        v = json.load(f)["versions"][0]
    q = v.get("queries", {})
    windows = a.output.endswith("windows.json")
    print(f"{os.path.basename(os.path.dirname(os.path.abspath(a.output)))} {v['version']}")

    checks = [(k, q[k]) for k in ("exists", "patched", "open") if q.get(k)]
    for k in ("exists", "patched", "open"):
        if not q.get(k):
            print(f"  no {k} query" + (": Patch when closed silently does nothing" if k == "open" else ""))
    if q.get("patched"):
        bumped, changes = bump_patched(q["patched"])
        for col, old, new in changes:
            print(f"  patched compares {col} to '{old}' (bumped: '{new}')")
        if changes:
            checks.append(("patched@bumped", bumped))
        else:
            print("  patched has no version_compare literal to bump: check it by hand")

    if windows or a.ps1:
        if not a.ps1:
            sys.exit("windows output: pass --ps1 <file>, then run that file as SYSTEM on a Windows host")
        body = PS1_HEADER + "".join(f"$queries['{k}'] = {ps_here_string(s)}\n" for k, s in checks)
        body += "foreach ($k in $queries.Keys) { '  {0,-16} {1}' -f $k, (Get-Rows $queries[$k]) }\n"
        # Windows PowerShell 5.1 reads a BOM-less script as ANSI, which mangles non-ASCII literals.
        with open(a.ps1, "w", encoding="utf-8-sig", newline="\r\n") as f:
            f.write(body)
        print(f"wrote {a.ps1}")
    else:
        if not os.access(a.osqueryd, os.X_OK):
            sys.exit(f"osqueryd not found at {a.osqueryd}; pass --osqueryd or set OSQ (see references/queries.md)")
        for k, sql in checks:
            if a.app_path:
                sql = constrain_apps(sql, a.app_path)
            print(f"  {k:<16} {run_osquery(a.osqueryd, sql)}")

    print(
        "Expected with the app installed and closed: exists >=1, patched 1 (0 only if this host\n"
        "really is behind), patched@bumped 0, open 1. Launch the app: open 0. Quit it: open 1."
    )


if __name__ == "__main__":
    main()
