"""Offline tests for the export scripts. Run: python3 -m unittest discover -s scripts/tests"""

import re
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from checks import Checker, yaml_backend  # noqa: E402
from fleet_api import Fleetctl, FleetctlError  # noqa: E402
from layout import Layout  # noqa: E402
from placer import Placer, glob_match  # noqa: E402
from planner import ALIASES, Planner, slugify  # noqa: E402
from yamlio import Doc, EnvRef, _shell_name, dump, env_escape  # noqa: E402

REPO = HERE.parents[4]
TEMPLATES = REPO / "cmd/fleetctl/fleetctl/templates/new"
DEPRECATIONS = REPO / "pkg/spec/gitops_deprecations.go"


def fleetctl_expand(s: str, env: dict) -> str:
    """Python port of pkg/spec expandEnv (escapes, $FLEET_VAR_/$FLEET_SECRET_ passthrough, unset vars fail)."""
    prefix = "PREVENT_ESCAPING_XYZ"
    s = re.sub(r"(\\+\$)", lambda m: m.group(0) if len(m.group(0)) % 2 else "\\" * (len(m.group(0)) // 2 - 1) + "$" + prefix, s)
    out, i = [], 0
    while i < len(s):
        if s[i] == "$" and i + 1 < len(s):
            rest = s[i + 1:]
            name = _shell_name(rest)
            if rest[0] == "{":
                end = rest.find("}")
                width = end + 1 if end > 0 else (2 if rest.startswith("{}") else 1)
            else:
                width = len(name)
            if not name:
                out.append("$")
                i += 1
                continue
            if name.startswith(prefix):
                out.append("$" + name[len(prefix):])
            elif name.upper().startswith("FLEET_VAR_") or name.startswith("FLEET_SECRET_"):
                out.append(s[i:i + 1 + width])
            elif name in env:
                out.append(env[name])
            else:
                raise KeyError(name)
            i += 1 + width
            continue
        out.append(s[i])
        i += 1
    return "".join(out)


class EnvEscapeTest(unittest.TestCase):
    CASES = [
        "SELECT '$HOME' AS h", "${PATH}", "cost $5", "trailing $", "$$", "$1 $? $@", "\\$X", "a\\\\$b", "\\\\\\$c",
        "$FLEET_VAR_HOST_UUID", "${FLEET_VAR_HOST_HARDWARE_SERIAL}", "$FLEET_SECRET_WIFI", "\\$FLEET_VAR_X", "${}", "${",
        'rule X { strings: $a = "hi" condition: $a }', "no dollars", "💻 $é",
    ]

    def test_round_trip(self):
        for text in self.CASES:
            with self.subTest(text=text):
                self.assertEqual(fleetctl_expand(env_escape(text), {}), text)

    def test_scalars_survive_expansion_and_yaml(self):
        backend = yaml_backend()
        if not backend:
            self.skipTest("no YAML parser")
        from checks import load_yaml
        for text in self.CASES + ["multi\nline $X\n", "quote ' and \" $Y", "  leading space $Z"]:
            with self.subTest(text=text):
                doc = dump({"v": text})
                self.assertEqual(load_yaml(fleetctl_expand(doc, {}), backend)["v"], text)

    def test_env_ref_is_expanded(self):
        self.assertEqual(dump({"secret": EnvRef("FLEET_X")}), "secret: $FLEET_X\n")
        self.assertEqual(fleetctl_expand(dump({"secret": EnvRef("FLEET_X")}), {"FLEET_X": "v"}), "secret: v\n")


class ReadOnlyTest(unittest.TestCase):
    def test_refuses_unlisted_paths(self):
        fc = Fleetctl("/nonexistent/fleetctl")
        for path in ("/api/latest/fleet/hosts", "/api/latest/fleet/scripts/run", "/api/v1/fleet/config"):
            with self.assertRaises(FleetctlError):
                fc.get(path)

    def test_gitops_is_always_dry_run(self):
        calls = []
        fc = Fleetctl("fleetctl")
        fc._run = lambda args, **kw: calls.append(args)
        fc.gitops_dry_run(["default.yml"], cwd=".", env={})
        self.assertEqual(calls[0][0], "gitops")
        self.assertEqual(calls[0][-1], "--dry-run")

    def test_api_never_sets_method(self):
        calls = []
        fc = Fleetctl("fleetctl")

        class R:
            returncode, stdout, stderr = 0, b"{}", b""

        fc._run = lambda args, **kw: calls.append(args) or R()
        fc.get("/api/latest/fleet/config", {"x": "1"})
        self.assertNotIn("-X", calls[0])


class HelpersTest(unittest.TestCase):
    def test_slugify(self):
        self.assertEqual(slugify("💻 Workstations"), "workstations")
        self.assertEqual(slugify("📱🔐 Personal mobile devices"), "personal-mobile-devices")
        self.assertEqual(slugify("macOS - FileVault: on"), "macos-filevault-on")

    def test_glob_match_stays_in_segment(self):
        self.assertTrue(glob_match("platforms/macos/scripts/*.sh", "platforms/macos/scripts/a.sh"))
        self.assertFalse(glob_match("platforms/macos/scripts/*.sh", "platforms/macos/scripts/x/a.sh"))

    def test_doc_replaces_commented_example(self):
        doc = Doc("controls:\n  # Uncomment to turn on\n  # windows_enabled_and_configured: true\n\nlabels:\n  - paths: ./labels/*.yml\n")
        doc.set(["controls", "windows_enabled_and_configured"], True)
        self.assertIn("  # Uncomment to turn on\n  windows_enabled_and_configured: true\n", doc.text())
        self.assertEqual(doc.all_path_globs(), [(("labels",), "paths", "./labels/*.yml")])

    @unittest.skipUnless(DEPRECATIONS.exists(), "not inside the Fleet repository")
    def test_aliases_cover_deprecated_keys(self):
        text = DEPRECATIONS.read_text()
        for parent in ("server_settings", "org_info"):
            for old, new in re.findall(r'\{"org_settings\.' + parent + r'\.(\w+)", "org_settings\.' + parent + r'\.(\w+)"\}', text):
                self.assertEqual(ALIASES[parent].get(old), new, f"{parent}.{old}")


def render_templates(dest: Path):
    for src in TEMPLATES.rglob("*"):
        rel = src.relative_to(TEMPLATES)
        out = dest / str(rel).replace(".template.", ".", 1)
        if src.is_dir():
            out.mkdir(parents=True, exist_ok=True)
        else:
            out.parent.mkdir(parents=True, exist_ok=True)
            out.write_bytes(src.read_bytes().replace(b"<%= .org_name %>", b"Test Org"))


def team(name, mdm=None):
    return {"name": name, "features": {"enable_host_users": True}, "host_expiry_settings": {"host_expiry_enabled": False, "host_expiry_window": 0},
            "webhook_settings": {}, "integrations": {}, "mdm": mdm or {}}


def snapshot():
    mobileconfig = (b'<?xml version="1.0" encoding="UTF-8"?>\n<plist version="1.0"><dict><key>PayloadDisplayName</key>'
                    b"<string>Firewall on</string><key>PayloadType</key><string>Configuration</string></dict></plist>\n")
    return {
        "me": {"email": "admin@example.test", "global_role": "admin"},
        "version": "4.99.0",
        "flags": {"premium": True, "apple_mdm": True, "windows_mdm": True, "android_mdm": False, "any_mdm": True},
        "gitops": {"exceptions": {"labels": False, "software": False, "secrets": False}},
        "config": {
            "org_info": {"org_name": "Test Org", "contact_url": "https://example.test/help"},
            "server_settings": {"server_url": "https://fleet.example.test", "live_query_disabled": False, "live_reporting_disabled": False},
            "features": {"enable_host_users": True}, "webhook_settings": {}, "integrations": {}, "sso_settings": {},
            "agent_options": {"config": {"options": {"distributed_interval": 10}}},
            "mdm": {"windows_enabled_and_configured": True, "enable_profile_validation": True,
                    "macos_updates": {"minimum_version": "latest", "deadline_days": 14}},
        },
        "enroll_secret_count": 1, "custom_host_vitals": [{"name": "Asset tag"}], "certificate_authorities": {},
        "graph_credentials": [], "vpp_tokens": [], "eula": {}, "logos": {}, "warnings": [],
        "scopes": [
            {"key": "global", "id": None, "name": "Global",
             "labels": [{"name": "Apple Silicon macOS hosts", "query": "SELECT 1", "label_membership_type": "dynamic", "platform": "darwin"},
                        {"name": "Arm Macs", "query": "SELECT 1", "label_membership_type": "dynamic"}],
             "policies": [{"id": 1, "name": "Global check", "query": "SELECT 1;", "description": "Costs $5", "platform": "darwin,linux"}],
             "reports": [{"id": 1, "name": "Uptime", "query": "SELECT * FROM uptime;", "platform": "", "interval": 60}]},
            {"key": "unassigned", "id": 0, "name": "Unassigned", "team": team("Unassigned"), "enroll_secret_count": 0,
             "policies": [{"id": 2, "name": "Unassigned check", "query": "SELECT 1;", "platform": "darwin"}],
             "scripts": [{"id": 1, "name": "hello.sh", "content": b"#!/bin/sh\nlaunchctl list\n"}], "profiles": [], "assets": [],
             "android_certificates": [], "software": {"titles": [], "setup": [], "setup_macos": []}, "setup": {}},
            {"key": "1", "id": 1, "name": "💻 Workstations", "team": team("💻 Workstations"), "enroll_secret_count": 1,
             "labels": [{"name": "Canary", "query": "SELECT 1", "label_membership_type": "dynamic", "fleet_id": 1}],
             "policies": [{"id": 3, "name": "Collect logs", "query": "SELECT 1;", "platform": "darwin", "run_script": {"id": 2, "name": "logs.sh"}}],
             "reports": [],
             "scripts": [{"id": 2, "name": "logs.sh", "content": b"#!/bin/sh\ndefaults read /Library/x\n"},
                         {"id": 3, "name": "hi.ps1", "content": b"Write-Host hi\n"}],
             "profiles": [{"profile_uuid": "a", "name": "Firewall on", "platform": "darwin", "content": mobileconfig,
                           "labels_include_any": [{"name": "Arm Macs"}]},
                          {"profile_uuid": "b", "name": "Min password", "platform": "windows", "content": b"<Replace><Data>$8</Data></Replace>\n"}],
             "assets": [], "android_certificates": [], "software": {"titles": [], "setup": [], "setup_macos": []}, "setup": {}},
            {"key": "2", "id": 2, "name": "Servers", "team": team("Servers"), "enroll_secret_count": 1, "labels": [],
             "policies": [{"id": 4, "name": "curl installed", "query": "SELECT 1;", "platform": "linux"}], "reports": [],
             "scripts": [{"id": 4, "name": "update.sh", "content": b"#!/bin/sh\napt-get update\n"}], "profiles": [], "assets": [],
             "android_certificates": [], "software": {"titles": [], "setup": [], "setup_macos": []}, "setup": {}},
        ],
    }


@unittest.skipUnless(TEMPLATES.exists(), "not inside the Fleet repository")
class EndToEndTest(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.root = self.tmp / "repo"
        render_templates(self.root)
        self.work = self.tmp / "work"
        shutil.copytree(self.root, self.work / "pristine")
        self.layout = Layout.scan(str(self.root))
        self.plan = Planner(snapshot(), self.layout, self.work).build()
        Placer(self.plan, self.layout, self.work).place()

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def read(self, rel):
        return (self.root / rel).read_text(encoding="utf-8")

    def test_layout(self):
        self.assertTrue((self.root / "fleets/workstations.yml").exists())
        self.assertTrue((self.root / "fleets/servers.yml").exists())
        self.assertTrue((self.root / "fleets/unassigned.yml").exists())
        self.assertFalse((self.root / "fleets/personal-mobile-devices.yml").exists())
        self.assertTrue((self.root / "labels/apple-silicon-macos-hosts.yml").exists())
        self.assertFalse((self.root / "labels/x86-based-windows-hosts.yml").exists())
        self.assertFalse((self.root / "platforms/macos/policies/all-software-updates-installed.yml").exists())
        self.assertTrue((self.root / "platforms/windows/configuration-profiles/Min password.xml").exists())
        self.assertTrue((self.root / "labels/workstations/canary.yml").exists())

    def test_globs_only_where_exact(self):
        ws, srv = self.read("fleets/workstations.yml"), self.read("fleets/servers.yml")
        self.assertIn("- paths: ../platforms/windows/scripts/*.ps1", ws)
        self.assertNotIn("../platforms/linux/scripts/*.sh", ws)
        self.assertIn("- paths: ../platforms/linux/scripts/*.sh", srv)
        self.assertNotIn("../platforms/macos/scripts/*.sh", srv)
        self.assertNotIn("configuration-profiles/*.mobileconfig", srv)
        self.assertIn("path: ../platforms/macos/configuration-profiles/firewall-on.mobileconfig", ws)

    def test_secrets_and_escaping(self):
        d = self.read("default.yml")
        self.assertIn("server_url: $FLEET_URL", d)
        self.assertIn("- secret: $FLEET_GLOBAL_ENROLL_SECRET", d)
        self.assertIn("$FLEET_WORKSTATIONS_ENROLL_SECRET", self.read("fleets/workstations.yml"))
        self.assertIn(r"Costs \$5", self.read("platforms/all/policies/global-check.yml"))
        self.assertIn(r"<Data>\$8</Data>", self.read("platforms/windows/configuration-profiles/Min password.xml"))
        self.assertIn("FLEET_SERVERS_ENROLL_SECRET=", self.read(".env.example"))
        self.assertNotIn("live_query_disabled", d)

    def test_unassigned_controls_in_default(self):
        d = self.read("default.yml")
        self.assertIn("path: ./platforms/macos/scripts/hello.sh", d)
        self.assertIn("windows_enabled_and_configured: true", d)
        self.assertNotRegex(self.read("fleets/unassigned.yml"), r"(?m)^controls:")

    def test_run_script_reference(self):
        self.assertIn("path: ../scripts/logs.sh", self.read("platforms/macos/policies/collect-logs.yml"))

    def test_checks_pass(self):
        ch = Checker(self.root, self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        ch.references()
        ch.reconcile()
        self.assertEqual([r for k in ("references", "reconcile") for r in ch.results[k]], [])

    def test_place_is_repeatable(self):
        before = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        Placer(self.plan, self.layout, self.work).place()
        after = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)


if __name__ == "__main__":
    unittest.main()
