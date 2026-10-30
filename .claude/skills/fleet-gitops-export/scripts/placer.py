"""Write an export plan into the scaffolded repo.

Fills the generated YAML in place (keeping its comments), writes every
artifact, wires fleet files to them, deletes template examples nothing uses,
and documents the environment variables the repo needs.
"""

from __future__ import annotations

import fnmatch
import json
import os
import re
import shutil
from pathlib import Path

from layout import Layout
from yamlio import Doc, EnvRef, dump, env_escape

UNASSIGNED_HEADER = """# # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
#
# unassigned.yml
#
# Use this file to configure policies and software for hosts
# that aren't in a fleet ("Unassigned").
#
# Controls for these hosts (configuration profiles, scripts,
# OS updates, setup experience) live under `controls:` in
# default.yml: fleetctl gitops accepts them in only one of
# the two files.
#
# To see all supported options, check out:
#   • https://fleetdm.com/docs/configuration/yaml-files
#
# # # # # # # # # # # # # # # # # # # # # # # # # # # # # #
name: Unassigned
"""

TOP_COMMENTS = {
    "agent_options": ["###########################################################",
                      "# Agent (osquery) options. See:",
                      "#   • https://fleetdm.com/docs/configuration/agent-configuration",
                      "###########################################################"],
    "policies": ["###########################################################",
                 "# Policies that apply to all fleets",
                 "###########################################################"],
    "reports": ["###########################################################",
                "# Reports that apply to all fleets",
                "###########################################################"],
    "custom_host_vitals": ["###########################################################",
                           "# Custom host vitals (names only; values are set per host)",
                           "###########################################################"],
}

# Order of top-level keys, used to pick where an added key goes.
TOP_ORDER = ["name", "settings", "org_settings", "agent_options", "controls", "labels", "custom_host_vitals",
             "reports", "policies", "software"]


def glob_match(pattern: str, path: str) -> bool:
    """Match like fleetctl's doublestar glob: `*` stays within one path segment."""
    pp, ps = pattern.split("/"), path.split("/")
    if "**" in pp:
        return fnmatch.fnmatchcase(path, pattern.replace("**/", "*"))
    return len(pp) == len(ps) and all(fnmatch.fnmatchcase(s, p) for p, s in zip(pp, ps))


def relref(target: str, from_dir: str) -> str:
    rel = os.path.relpath(target, from_dir or ".").replace(os.sep, "/")
    return rel if rel.startswith(".") else "./" + rel


class Placer:
    def __init__(self, plan: dict, layout: Layout, work: Path, log=print):
        self.plan = plan
        self.layout = layout
        self.root = layout.root
        self.work = work
        self.log = log
        self.files = plan["files"]
        self.written: list[str] = []

    def target(self, rel: str) -> Path:
        """The repo path for a plan path, refusing any that would land outside the repo.

        Plan paths come from Fleet data and from a plan.json a person or agent may edit.
        """
        p = Path(rel)
        if not rel or p.is_absolute() or ".." in p.parts:
            raise ValueError(f"refusing {rel!r}: plan paths must be relative and stay inside {self.root}")
        out = self.root / p
        if not out.resolve().is_relative_to(self.root.resolve()):
            raise ValueError(f"refusing {rel!r}: it resolves outside {self.root} through a symlink")
        return out

    def resolve(self, data, base_dir: str):
        if isinstance(data, dict):
            if set(data) == {"$file"}:
                return relref(self.files[data["$file"]]["path"], base_dir)
            if set(data) == {"$env"}:
                return EnvRef(data["$env"])
            return {k: self.resolve(v, base_dir) for k, v in data.items()}
        if isinstance(data, list):
            return [self.resolve(v, base_dir) for v in data]
        return data

    # ------------------------------------------------------------ main

    def place(self):
        for rel in [f["path"] for f in self.files.values()] + [f["file"] for f in self.plan["fleets"]] + self.plan["remove"]:
            self.target(rel)
        self._restore()
        for f in self.files.values():
            self._write_file(f)
        for fleet in self.plan["fleets"]:
            self._write_fleet(fleet)
        self._remove()
        if self.plan["exceptions"].get("labels"):
            doc = Doc((self.root / "default.yml").read_text(encoding="utf-8"))
            doc.comment_out(["labels"])
            (self.root / "default.yml").write_text(doc.text(), encoding="utf-8")
        self._env_docs()
        (self.work / "placed.json").write_text(json.dumps(sorted(set(self.written)), indent=2), encoding="utf-8")

    def _restore(self):
        """Put the scaffold back the way fleetctl new left it, so placing twice gives the same result."""
        placed = self.work / "placed.json"
        if placed.exists():
            for rel in json.loads(placed.read_text()):
                p = self.target(rel)
                if p.is_file():
                    p.unlink()
        pristine = self.work / "pristine"
        for dirpath, _, filenames in os.walk(pristine):
            for name in filenames:
                src = Path(dirpath) / name
                dst = self.root / src.relative_to(pristine)
                dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src, dst)
        for dirpath, dirnames, filenames in sorted(os.walk(self.root), reverse=True):
            if ".git" in Path(dirpath).parts:
                continue
            if not dirnames and not filenames and Path(dirpath) != self.root:
                os.rmdir(dirpath)

    def _write_file(self, f: dict):
        path = self.target(f["path"])
        path.parent.mkdir(parents=True, exist_ok=True)
        if "blob" in f:
            content = (self.work / "blobs" / f["blob"]).read_bytes()
            if f.get("escape"):
                try:
                    text = content.decode("utf-8")
                except UnicodeDecodeError:
                    # Not text (the plan already warned about it), so it's written byte for byte.
                    text = None
                if text is not None:
                    content = env_escape(text).encode("utf-8")
        else:
            content = dump(self.resolve(f["data"], os.path.dirname(f["path"]))).encode("utf-8")
        path.write_bytes(content)
        self.written.append(f["path"])

    # ------------------------------------------------------------ fleet files

    def _write_fleet(self, fleet: dict):
        rel = fleet["file"]
        base = os.path.dirname(rel)
        if fleet["action"] == "fill":
            text = self.target(rel).read_text(encoding="utf-8")
        elif fleet["action"] == "clone":
            text = self._clone_text(fleet["template"], rel)
        else:
            text = UNASSIGNED_HEADER
        doc = Doc(text)
        sections = self.resolve(fleet["sections"], base)
        for key in TOP_ORDER:
            if key not in sections:
                continue
            value = sections[key]
            if key in ("org_settings", "controls", "settings", "software") and isinstance(value, dict):
                if doc.find([key]) is None:
                    if value:
                        self._insert(doc, key, value)
                    continue
                doc.merge([key], value)
            elif doc.find([key]) is not None:
                doc.set([key], value)
            else:
                self._insert(doc, key, value)
        sections = dict(fleet["entries"])
        if fleet["action"] in ("fill", "clone"):
            # A template glob in a section this fleet has nothing for still has to be checked:
            # left alone, it would pull in another fleet's files from the shared folder.
            for g in self.layout.globs:
                if g["file"] == fleet["template"]:
                    sections.setdefault(".".join(g["keypath"]), [])
        for section, entries in sections.items():
            if section == "labels" and fleet["file"] == "default.yml":
                continue
            self._apply_entries(doc, fleet, section.split("."), entries, base)
        if self.plan["exceptions"].get("software") and doc.find(["software"]) is not None:
            doc.comment_out(["software"])
        out = self.target(rel)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(doc.text(), encoding="utf-8")
        self.written.append(rel)

    def _insert(self, doc: Doc, key: str, value):
        after = None
        for k in TOP_ORDER[:TOP_ORDER.index(key)] if key in TOP_ORDER else TOP_ORDER:
            if doc.find([k]) is not None:
                after = k
        doc.insert_top(key, value, TOP_COMMENTS.get(key) if doc.find(["org_settings"]) is not None else None, after=after)

    def _clone_text(self, template: str, new_rel: str) -> str:
        pristine = self.work / "pristine" / template
        text = pristine.read_text(encoding="utf-8")
        lines, out, in_header = text.split("\n"), [], True
        old_name, new_name = os.path.basename(template), os.path.basename(new_rel)
        for ln in lines:
            if in_header and not ln.startswith("#"):
                in_header = False
            if in_header:
                if ln.startswith("# > "):
                    continue
                if ln.strip() == f"# {old_name}":
                    ln = f"# {new_name}"
                if ln == "#" and out and out[-1] == "#":
                    continue
            out.append(ln)
        return "\n".join(out)

    def _apply_entries(self, doc: Doc, fleet: dict, keypath: list, entries: list, base: str):
        template = fleet["template"] if fleet["action"] in ("fill", "clone") else None
        globs = self.layout.fleet_globs(template, keypath) if template else []
        all_paths = {fid: f["path"] for fid, f in self.files.items()}
        kept, covered = [], set()
        for g in globs:
            in_folder = {fid for fid, p in all_paths.items() if glob_match(g["root_pattern"], p)}
            plain = {e["file"] for e in entries if "file" in e and not e["opts"] and e["file"] in in_folder}
            if in_folder == plain:
                kept.append(g)
                covered |= plain
        items = [{g["key"]: g["pattern"]} for g in kept]
        for e in entries:
            if "inline" in e:
                items.append(self.resolve(e["inline"], base))
            elif e["file"] not in covered:
                item = {"path": relref(self.files[e["file"]]["path"], base)}
                item.update(self.resolve(e["opts"], base))
                items.append(item)
        if len(kept) == len(globs) and len(items) == len(kept) and (globs or doc.find(keypath) is None):
            return
        if not items and doc.find(keypath) is None:
            return
        if len(keypath) == 1 and doc.find(keypath) is None:
            self._insert(doc, keypath[0], items)
        else:
            doc.set(keypath, items if items else None)

    # ------------------------------------------------------------ cleanup and docs

    def _remove(self):
        written = set(self.written)
        for rel in self.plan["remove"]:
            if rel in written:
                continue
            p = self.target(rel)
            if p.is_file():
                p.unlink()
        if self.plan["exceptions"].get("labels"):
            labels = self.root / self.layout.labels_dir
            if labels.is_dir():
                shutil.rmtree(labels)
        for f in self.files.values():
            for parent in Path(f["path"]).parents:
                keep = self.root / parent / ".gitkeep"
                if keep.exists():
                    keep.unlink()

    def _env_docs(self):
        env = self.plan["env"]
        lines = ["# Environment variables this GitOps repo uses. Set each one as a",
                 "# repository secret (GitHub) or CI/CD variable (GitLab). For a local",
                 "# dry run, copy this file to .env (git ignores it) and fill in values.",
                 "# Never commit real values.",
                 "# FLEET_API_TOKEN isn't referenced in YAML but CI needs it to reach Fleet.", ""]
        for e in env:
            lines.append(f"# {e['description']}")
            lines.append(f"# Used in: {', '.join(e['used_in'])}")
            lines.append(f"{e['name']}=")
            lines.append("")
        (self.root / ".env.example").write_text("\n".join(lines), encoding="utf-8")
        self.written.append(".env.example")

        gi = self.root / ".gitignore"
        if gi.exists() and not re.search(r"^\.env$", gi.read_text(encoding="utf-8"), re.M):
            gi.write_text(gi.read_text(encoding="utf-8").rstrip("\n") + "\n\n# Local environment variables (see .env.example)\n.env\n", encoding="utf-8")

        readme = self.root / "README.md"
        if readme.exists() and env:
            intro = ("These YAML files reference the variables below instead of storing secrets. Add each one as a "
                     "repository secret (GitHub) or CI/CD variable (GitLab). `.env.example` lists the same variables "
                     "for local dry runs.")
            rows = ["", "## Environment variables", "", intro, "",
                    "| Variable | Used in | What it is |", "|---|---|---|"]
            for e in env:
                rows.append(f"| `{e['name']}` | {', '.join(f'`{u}`' for u in e['used_in'])} | {e['description']} |")
            readme.write_text(readme.read_text(encoding="utf-8").rstrip("\n") + "\n" + "\n".join(rows) + "\n", encoding="utf-8")

        wf = self.root / ".github/workflows/workflow.yml"
        extra = [e["name"] for e in env if e["name"] not in ("FLEET_URL", "FLEET_API_TOKEN")]
        if wf.exists() and extra:
            text = wf.read_text(encoding="utf-8")
            m = re.search(r"^( *)FLEET_API_TOKEN: \$\{\{ secrets\.FLEET_API_TOKEN \}\}\n", text, re.M)
            if m:
                ind = m.group(1)
                add = "".join(f"{ind}{n}: ${{{{ secrets.{n} }}}}\n" for n in extra if f"{n}:" not in text)
                text = text[:m.end()] + add + text[m.end():]
                wf.write_text(text, encoding="utf-8")

        gs = self.root / ".github/fleet-gitops/gitops.sh"
        names = {e["name"] for e in env}
        if gs.exists() and names & {"FLEET_SSO_METADATA", "FLEET_MDM_SSO_METADATA"}:
            text = gs.read_text(encoding="utf-8")
            if "FLEET_SSO_METADATA" in names:
                text = text.replace("# FLEET_SSO_METADATA=$( sed '2,$s/^/      /' <<<  \"${FLEET_MDM_SSO_METADATA}\")",
                                    "FLEET_SSO_METADATA=$( sed '2,$s/^/      /' <<<  \"${FLEET_SSO_METADATA}\")")
            if "FLEET_MDM_SSO_METADATA" in names:
                text = text.replace("# FLEET_MDM_SSO_METADATA=$( sed '2,$s/^/        /' <<<  \"${FLEET_MDM_SSO_METADATA}\")",
                                    "FLEET_MDM_SSO_METADATA=$( sed '2,$s/^/        /' <<<  \"${FLEET_MDM_SSO_METADATA}\")")
            gs.write_text(text, encoding="utf-8")
