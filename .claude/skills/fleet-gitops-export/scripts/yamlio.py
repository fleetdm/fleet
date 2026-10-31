"""YAML output for GitOps files.

`dump` writes block-style YAML for plain data. `Doc` edits the YAML files
`fleetctl new` generates in place, so their guidance comments survive.

fleetctl gitops expands $VAR and ${VAR} in every YAML file and profile it
reads (not in scripts) and fails when a variable isn't set. `env_escape` makes
literal dollar signs survive that expansion; `EnvRef` marks the values that
should be expanded.
"""

from __future__ import annotations

import json
import re


class EnvRef(str):
    """A value written as $NAME so fleetctl gitops fills it from the environment."""


# ---------------------------------------------------------------- escaping

_SPECIAL_VAR = set("*#$@!?-0123456789")


def _shell_name(s: str) -> str:
    # Mirrors fleet.MaybeExpand's getShellName: the variable name that follows a '$'.
    if not s:
        return ""
    if s[0] == "{":
        if len(s) > 2 and s[1] in _SPECIAL_VAR and s[2] == "}":
            return s[1]
        end = s.find("}")
        return s[1:end] if end > 1 else ""
    if s[0] in _SPECIAL_VAR:
        return s[0]
    m = re.match(r"[A-Za-z0-9_]+", s)
    return m.group(0) if m else ""


def _expandable(text: str, j: int) -> bool:
    name = _shell_name(text[j + 1:])
    if not name:
        return False
    # fleetctl leaves these for the server.
    return not (name.upper().startswith("FLEET_VAR_") or name.startswith("FLEET_SECRET_"))


def env_escape(text: str) -> str:
    """Return text that fleetctl's env expansion turns back into `text`.

    fleetctl turns an odd run of k backslashes before '$' into (k-1)/2
    backslashes and a literal '$', and expands '$NAME' after an even run. So
    any '$' it would touch gets 2k+1 backslashes.
    """
    out = []
    i, n = 0, len(text)
    while i < n:
        c = text[i]
        if c == "\\":
            j = i
            while j < n and text[j] == "\\":
                j += 1
            k = j - i
            if j < n and text[j] == "$" and (k % 2 == 1 or _expandable(text, j)):
                out.append("\\" * (2 * k + 1) + "$")
                i = j + 1
            else:
                out.append(text[i:j])
                i = j
            continue
        if c == "$" and _expandable(text, i):
            out.append("\\$")
        else:
            out.append(c)
        i += 1
    return "".join(out)


def fleet_secret_names(text: str) -> set[str]:
    return set(re.findall(r"\$\{?(FLEET_SECRET_[A-Za-z0-9_]+)", text))


# ---------------------------------------------------------------- emitting

_PLAIN = re.compile(r"(?:[A-Za-z_/]|\.\.?/)[A-Za-z0-9_ ./@()\-,+=*]*")
_AMBIGUOUS = {"y", "n", "yes", "no", "true", "false", "on", "off", "null", "~"}
_KEY_PLAIN = re.compile(r"[A-Za-z0-9_][A-Za-z0-9_\-.]*")


def _plain_ok(s: str) -> bool:
    return bool(_PLAIN.fullmatch(s)) and not s.endswith(" ") and s.lower() not in _AMBIGUOUS


def scalar(v) -> str:
    if v is None:
        return ""
    if v is True:
        return "true"
    if v is False:
        return "false"
    if isinstance(v, EnvRef):
        return "$" + v
    if isinstance(v, (int, float)):
        return json.dumps(v)
    s = str(v)
    if _plain_ok(s):
        return s
    if env_escape(s) == s:
        return json.dumps(s, ensure_ascii=False)
    # "\$" isn't a valid escape inside double quotes for standard YAML parsers, so a string
    # with a dollar sign fleetctl would expand goes in single quotes when it can, and
    # otherwise in double quotes with that dollar written as \x24, which YAML reads as "$"
    # and fleetctl's expansion never sees.
    if not any((ord(c) < 32 and c != "\t") or c == "\x7f" for c in s):
        return "'" + env_escape(s).replace("'", "''") + "'"
    quoted = json.dumps(s, ensure_ascii=False)
    return "".join("\\x24" if c == "$" and _expandable(quoted, i) else c for i, c in enumerate(quoted))


def _key(k) -> str:
    k = str(k)
    if _KEY_PLAIN.fullmatch(k) and k.lower() not in _AMBIGUOUS:
        return k
    return json.dumps(k, ensure_ascii=False)


def _block_ok(s) -> bool:
    if not isinstance(s, str) or isinstance(s, EnvRef) or "\n" not in s or "\r" in s:
        return False
    if any(ord(c) < 32 and c not in "\n\t" for c in s):
        return False
    first = next((ln for ln in s.split("\n") if ln.strip()), "")
    return bool(first) and not first[0].isspace()


def _block(s: str, ind: int) -> tuple[str, list[str]]:
    if s.endswith("\n\n"):
        header, body = "|+", s[:-1]
    elif s.endswith("\n"):
        header, body = "|", s[:-1]
    else:
        header, body = "|-", s
    pad = " " * ind
    return header, [(pad + env_escape(ln)) if ln else "" for ln in body.split("\n")]


def _emit_map(d: dict, ind: int, out: list[str]):
    pad = " " * ind
    for k, v in d.items():
        key = _key(k)
        if isinstance(v, dict):
            if v:
                out.append(f"{pad}{key}:")
                _emit_map(v, ind + 2, out)
            else:
                out.append(f"{pad}{key}: {{}}")
        elif isinstance(v, list):
            if v:
                out.append(f"{pad}{key}:")
                _emit_list(v, ind + 2, out)
            else:
                out.append(f"{pad}{key}: []")
        elif _block_ok(v):
            header, lines = _block(v, ind + 2)
            out.append(f"{pad}{key}: {header}")
            out.extend(lines)
        else:
            s = scalar(v)
            out.append(f"{pad}{key}: {s}" if s else f"{pad}{key}:")


def _emit_list(lst: list, ind: int, out: list[str]):
    pad = " " * ind
    for item in lst:
        if isinstance(item, dict) and item:
            sub: list[str] = []
            _emit_map(item, ind + 2, sub)
            sub[0] = pad + "- " + sub[0][ind + 2:]
            out.extend(sub)
        elif isinstance(item, list) and item:
            sub = []
            _emit_list(item, ind + 2, sub)
            sub[0] = pad + "- " + sub[0][ind + 2:]
            out.extend(sub)
        elif isinstance(item, dict):
            out.append(pad + "- {}")
        elif isinstance(item, list):
            out.append(pad + "- []")
        elif _block_ok(item):
            header, lines = _block(item, ind + 2)
            out.append(pad + "- " + header)
            out.extend(lines)
        else:
            out.append(pad + "- " + scalar(item))


def emit_lines(value, ind: int = 0) -> list[str]:
    out: list[str] = []
    if isinstance(value, dict):
        _emit_map(value, ind, out)
    elif isinstance(value, list):
        _emit_list(value, ind, out)
    else:
        out.append(" " * ind + scalar(value))
    return out


def dump(value) -> str:
    return "\n".join(emit_lines(value)) + "\n"


def dump_value_lines(key: str, value, ind: int) -> list[str]:
    """`key: value` as lines at the given indent."""
    out: list[str] = []
    _emit_map({key: value}, ind, out)
    return out


# ---------------------------------------------------------------- editing

_KEY_LINE = re.compile(r"^(?P<ind> *)(?P<dash>- +)?(?P<key>\"[^\"]*\"|[A-Za-z0-9_][A-Za-z0-9_\-.]*) *:(?:[ \t]+(?P<rest>.*))?$")
_BLOCK_START = re.compile(r"^[|>][-+0-9]*\s*(#.*)?$")


def _comment_col(line: str):
    s = line.lstrip(" ")
    return len(line) - len(s) if s.startswith("#") else None


def _vindent(line: str) -> int:
    """Indent of the YAML inside a `# key: value` comment, the template's convention for commented-out settings."""
    col = _comment_col(line)
    after = line[col + 1:]
    spaces = len(after) - len(after.lstrip(" "))
    return col + spaces - 1 if spaces else col - 1


class Doc:
    """A comment-preserving editor for the block-style YAML that `fleetctl new` writes."""

    def __init__(self, text: str):
        self.lines = text.split("\n")

    def text(self) -> str:
        return "\n".join(self.lines)

    # --- reading

    def _is_blank(self, i):
        return not self.lines[i].strip()

    def _is_comment(self, i):
        return _comment_col(self.lines[i]) is not None

    def _indent(self, i):
        ln = self.lines[i]
        return len(ln) - len(ln.lstrip(" "))

    def _block_body_end(self, i: int) -> int:
        """If line i opens a block scalar, the index after its body; else i + 1."""
        m = _KEY_LINE.match(self.lines[i])
        if not m or m.group("rest") is None or not _BLOCK_START.match(m.group("rest")):
            return i + 1
        base = len(m.group("ind")) + (len(m.group("dash")) if m.group("dash") else 0)
        j = i + 1
        while j < len(self.lines) and (self._is_blank(j) or self._indent(j) > base):
            j += 1
        while j > i + 1 and self._is_blank(j - 1):
            j -= 1
        return j

    def region(self, i: int) -> tuple[int, int]:
        """Lines holding the value of the key on line i: [start, end), trailing blanks excluded."""
        m = _KEY_LINE.match(self.lines[i])
        d = len(m.group("ind")) + (len(m.group("dash")) if m and m.group("dash") else 0)
        j = self._block_body_end(i)
        if j > i + 1:
            return i + 1, j
        n = len(self.lines)
        while j < n:
            if self._is_blank(j):
                j += 1
                continue
            col = _comment_col(self.lines[j])
            if col is not None:
                if col <= d:
                    break
                j += 1
                continue
            if self._indent(j) <= d:
                break
            j = self._block_body_end(j)
        end = j
        while end > i + 1 and self._is_blank(end - 1):
            end -= 1
        return i + 1, end

    def _keys_in(self, start: int, end: int, indent: int):
        i = start
        while i < end:
            if self._is_blank(i) or self._is_comment(i):
                i += 1
                continue
            m = _KEY_LINE.match(self.lines[i])
            if m and not m.group("dash") and len(m.group("ind")) == indent:
                yield m.group("key").strip('"'), i
                _, e = self.region(i)
                i = max(e, i + 1)
                continue
            i += 1

    def _child_indent(self, start: int, end: int, parent_indent: int) -> int:
        for i in range(start, end):
            if not self._is_blank(i) and not self._is_comment(i):
                return self._indent(i)
        return parent_indent + 2

    def find(self, path) -> int | None:
        start, end, indent = 0, len(self.lines), 0
        idx = None
        for k in path:
            idx = None
            for key, i in self._keys_in(start, end, indent):
                if key == k:
                    idx = i
                    break
            if idx is None:
                return None
            start, end = self.region(idx)
            indent = self._child_indent(start, end, self._indent(idx))
        return idx

    def value_text(self, path) -> str | None:
        i = self.find(path)
        if i is None:
            return None
        m = _KEY_LINE.match(self.lines[i])
        return (m.group("rest") or "").split(" #")[0].strip()

    def list_items(self, path):
        """(line, key, value) for `- path: x` / `- paths: x` items directly under path."""
        i = self.find(path)
        if i is None:
            return []
        start, end = self.region(i)
        items = []
        for j in range(start, end):
            m = _KEY_LINE.match(self.lines[j])
            if m and m.group("dash"):
                items.append((j, m.group("key"), (m.group("rest") or "").split(" #")[0].strip().strip('"').strip("'")))
        return items

    def all_path_globs(self):
        """Every `- path:`/`- paths:` list item in the file, with the key path that holds it."""
        found = []

        def walk(start, end, indent, prefix):
            for key, i in self._keys_in(start, end, indent):
                s, e = self.region(i)
                for j in range(s, e):
                    m = _KEY_LINE.match(self.lines[j])
                    if m and m.group("dash") and len(m.group("ind")) == self._child_indent(s, e, indent) and m.group("key") in ("path", "paths"):
                        found.append((tuple(prefix + [key]), m.group("key"), (m.group("rest") or "").split(" #")[0].strip().strip('"').strip("'")))
                walk(s, e, self._child_indent(s, e, indent), prefix + [key])

        walk(0, len(self.lines), 0, [])
        return found

    def _commented(self, start: int, end: int, key: str, vind: int):
        pat = re.compile(r"^ *#( +)" + re.escape(key) + r":(\s|$)")
        for i in range(start, end):
            if pat.match(self.lines[i]) and _vindent(self.lines[i]) == vind:
                j = i + 1
                while j < end and self._is_comment(j) and _vindent(self.lines[j]) > vind:
                    j += 1
                return i, j
        return None

    # --- writing

    def set(self, path, value):
        """Set path to value: replaces a real key's value, replaces a commented-out example, or appends."""
        path = list(path)
        i = self.find(path)
        if i is not None:
            ind = self._indent(i)
            start, end = self.region(i)
            if isinstance(value, (dict, list)) and value:
                head = _KEY_LINE.match(self.lines[i])
                rest = head.group("rest") or ""
                comment = rest[rest.index("#"):] if rest.strip().startswith("#") else ""
                self.lines[i] = f"{' ' * ind}{_key(path[-1])}:" + (f" {comment}" if comment else "")
                self.lines[start:end] = emit_lines(value, ind + 2)
            else:
                self.lines[i:end] = dump_value_lines(path[-1], value, ind)
            return
        parent = path[:-1]
        if parent:
            pi = self.find(parent)
            if pi is None:
                self.set(parent, {path[-1]: value})
                return
            pstart, pend = self.region(pi)
            cind = self._child_indent(pstart, pend, self._indent(pi))
            m = _KEY_LINE.match(self.lines[pi])
            if m and (m.group("rest") or "").strip() and not (m.group("rest") or "").strip().startswith("#"):
                # Parent holds an inline scalar ([] or {} or a value): rewrite it as a mapping.
                self.lines[pi:pend] = [f"{' ' * self._indent(pi)}{_key(parent[-1])}:"]
                pstart = pend = pi + 1
        else:
            pstart, pend, cind = 0, len(self.lines), 0
        hit = self._commented(pstart, pend, path[-1], cind)
        new = dump_value_lines(path[-1], value, cind)
        if hit:
            self.lines[hit[0]:hit[1]] = new
            return
        if parent:
            self.lines[pend:pend] = new
        else:
            while self.lines and not self.lines[-1].strip():
                self.lines.pop()
            self.lines += [""] + new + [""]

    def merge(self, path, value: dict):
        """Set each key of value under path, keeping the comments around keys already there."""
        for k, v in value.items():
            p = list(path) + [k]
            i = self.find(p)
            if isinstance(v, dict) and v and i is not None:
                m = _KEY_LINE.match(self.lines[i])
                rest = (m.group("rest") or "").strip()
                if not rest or rest.startswith("#"):
                    self.merge(p, v)
                    continue
            self.set(p, v)

    def insert_top(self, key: str, value, comment: list[str] | None = None, after: str | None = None):
        """Add a top-level key with an optional comment block, after another top-level key or at the end."""
        new = (comment or []) + dump_value_lines(key, value, 0)
        if after is not None:
            ai = self.find([after])
            if ai is not None:
                _, end = self.region(ai)
                self.lines[end:end] = [""] + new
                return
        while self.lines and not self.lines[-1].strip():
            self.lines.pop()
        self.lines += [""] + new + [""]

    def remove(self, path):
        i = self.find(list(path))
        if i is None:
            return False
        _, end = self.region(i)
        del self.lines[i:end]
        return True

    def comment_out(self, path):
        i = self.find(list(path))
        if i is None:
            return False
        _, end = self.region(i)
        ind = self._indent(i)
        for j in range(i, end):
            ln = self.lines[j]
            self.lines[j] = (" " * ind + "# " + ln[ind:]) if ln.strip() else ln
        return True
