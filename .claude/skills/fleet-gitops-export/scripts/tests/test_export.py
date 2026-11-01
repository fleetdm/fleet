"""Offline tests for the export scripts. Run: python3 -m unittest discover -s scripts/tests"""

import argparse
import re
import shutil
import stat
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

import gitops_export  # noqa: E402
from checks import PLACEHOLDER, Checker, load_yaml, read_env_file, yaml_backend  # noqa: E402
from collect import Collector  # noqa: E402
from fleet_api import Fleetctl, FleetctlError  # noqa: E402
from fleet_defaults import is_default_script  # noqa: E402
from layout import Layout  # noqa: E402
from placer import Placer, glob_match  # noqa: E402
from planner import ALIASES, Planner, _fma_script_state, _script_platform, _url_names_version, slugify  # noqa: E402
from yamlio import Doc, EnvRef, _shell_name, dump, env_escape  # noqa: E402

REPO = HERE.parents[4]
TEMPLATES = REPO / "cmd/fleetctl/fleetctl/templates/new"
DEPRECATIONS = REPO / "pkg/spec/gitops_deprecations.go"
POINTER = REPO / ".agents/skills/fleet-gitops-export/SKILL.md"
FLEET_SCRIPTS = REPO / "pkg/file"
DEFAULT_SCRIPTS = HERE.parent / "fleet_default_scripts"


def default_script(name: str, package_id: str = "") -> str:
    """Fleet's default script as Fleet stores it, with the package ID filled in."""
    text = (DEFAULT_SCRIPTS / name).read_text(encoding="utf-8")
    return text.replace("$PACKAGE_ID", f"'{package_id}'") if package_id else text


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
        for text in self.CASES + ["multi\nline $X\n", "quote ' and \" $Y", "  leading space $Z", "tab\there $X", "bell\x07 ${Y} \\$Z"]:
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


class CommandTest(unittest.TestCase):
    def test_all_returns_validation_failure(self):
        calls = []
        saved = {n: getattr(gitops_export, n) for n in ("cmd_preflight", "cmd_scaffold", "cmd_export", "cmd_place", "cmd_validate", "cmd_report")}
        try:
            for n in saved:
                setattr(gitops_export, n, (lambda name: lambda args: calls.append(name) or (1 if name == "cmd_validate" else 0))(n))
            self.assertEqual(gitops_export.cmd_all(None), 1)
            self.assertEqual(calls[-1], "cmd_report")
        finally:
            for n, f in saved.items():
                setattr(gitops_export, n, f)

    def test_work_dir_is_private(self):
        tmp = Path(tempfile.mkdtemp())
        try:
            w = gitops_export.work_dir(tmp / "repo", str(tmp / "work"))
            self.assertEqual(stat.S_IMODE(w.stat().st_mode), 0o700)
            (tmp / "elsewhere").mkdir()
            (tmp / "link").symlink_to(tmp / "elsewhere")
            with self.assertRaises(SystemExit):
                gitops_export.work_dir(tmp / "repo", str(tmp / "link"))
        finally:
            shutil.rmtree(tmp)

    def test_missing_work_files_fail_clearly(self):
        tmp = Path(tempfile.mkdtemp())
        try:
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl=None, config=None, context=None)
            for cmd, step in ((gitops_export.cmd_export, "scaffold"), (gitops_export.cmd_place, "scaffold"),
                              (gitops_export.cmd_validate, "export"), (gitops_export.cmd_report, "export")):
                with self.subTest(cmd=cmd.__name__), self.assertRaises(SystemExit) as cm:
                    cmd(args)
                self.assertIn(f"run `{step}` first", str(cm.exception.code))
        finally:
            shutil.rmtree(tmp)


class DryRunEnvTest(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def test_read_env_file(self):
        f = self.tmp / ".env"
        f.write_text("# comment\nFLEET_A=plain  # note\nexport FLEET_B='single $x'\nFLEET_C=\"multi\nline\"\nFLEET_EMPTY=\n\nnot an entry\n")
        self.assertEqual(read_env_file(f), {"FLEET_A": "plain", "FLEET_B": "single $x", "FLEET_C": "multi\nline"})

    def test_env_file_fills_values_fleet_cant_return(self):
        f = self.tmp / ".env"
        f.write_text("FLEET_SECRET_EXPORT_TEST_A=s3cr3t-from-file\nFLEET_EXPORT_TEST_TOKEN=s3cr3t-token\nFLEET_SECRET_EXPORT_TEST_B=\n")
        plan = {"env": [{"name": "FLEET_SECRET_EXPORT_TEST_A", "source": {"type": "fleet_secret"}},
                        {"name": "FLEET_EXPORT_TEST_TOKEN", "source": {"type": "mask"}},
                        {"name": "FLEET_SECRET_EXPORT_TEST_B", "source": {"type": "fleet_secret"}}]}
        env, notes = Checker(self.tmp, plan).dry_run_env(None, f)
        self.assertEqual(env["FLEET_SECRET_EXPORT_TEST_A"], "s3cr3t-from-file")
        self.assertEqual(env["FLEET_EXPORT_TEST_TOKEN"], "s3cr3t-token")
        self.assertEqual(env["FLEET_SECRET_EXPORT_TEST_B"], PLACEHOLDER)
        self.assertEqual(len(notes), 1)
        self.assertNotIn("s3cr3t", " ".join(notes))

    def test_mask_stays_a_yaml_string(self):
        backend = yaml_backend()
        if not backend:
            self.skipTest("no YAML parser")
        plan = {"env": [{"name": "FLEET_EXPORT_TEST_MASKED", "source": {"type": "mask"}}]}
        env, _ = Checker(self.tmp, plan).dry_run_env(None, None)
        doc = fleetctl_expand("integrations:\n  api_key_json: $FLEET_EXPORT_TEST_MASKED\n", env)
        self.assertEqual(load_yaml(doc, backend)["integrations"]["api_key_json"], "********")


class SecretSpansTest(unittest.TestCase):
    def test_marks_enroll_secrets_without_keeping_values(self):
        c = Collector(api=None)
        c._secret_values = [("wk-s3cr3t-value", {"type": "enroll_secret", "team": 1, "index": 0}),
                            ("short", {"type": "enroll_secret", "team": None, "index": 0})]
        scopes = [{"profiles": [{"content": b"<string>wk-s3cr3t-value</string> and wk-s3cr3t-value"}],
                   "scripts": [{"content": b"echo short"}], "assets": [{"content": b"\xff\xfe"}]}]
        c._mark_secrets(scopes)
        refs = scopes[0]["profiles"][0]["env_refs"]
        self.assertEqual([(r["start"], r["end"]) for r in refs], [(8, 23), (37, 52)])
        self.assertEqual(refs[0]["source"], {"type": "enroll_secret", "team": 1, "index": 0})
        self.assertNotIn("env_refs", scopes[0]["scripts"][0])
        self.assertNotIn("wk-s3cr3t-value", repr(refs))


class FleetDefaultsTest(unittest.TestCase):
    def test_recognizes_defaults_with_ids_filled_in(self):
        self.assertTrue(is_default_script("install_script", ".deb", default_script("install_deb.sh")))
        self.assertTrue(is_default_script("uninstall_script", "deb", default_script("uninstall_deb.sh", "slack-desktop")))
        pkg = default_script("uninstall_pkg.sh").replace("$PACKAGE_ID", "(\n  'com.example.a'\n  'com.example.b'\n)")
        self.assertTrue(is_default_script("uninstall_script", "pkg", pkg))
        upgrade = default_script("uninstall_msi_with_upgrade_code.ps1").replace('"$UPGRADE_CODE"', "'{ABC-123}'")
        self.assertTrue(is_default_script("uninstall_script", "msi", upgrade))

    def test_custom_scripts_are_not_defaults(self):
        self.assertFalse(is_default_script("install_script", "deb", default_script("install_deb.sh") + "systemctl restart x\n"))
        self.assertFalse(is_default_script("install_script", "exe", "Start-Process setup.exe"))
        self.assertFalse(is_default_script("post_install_script", "deb", default_script("install_deb.sh")))

    @unittest.skipUnless(FLEET_SCRIPTS.exists(), "not inside the Fleet repository")
    def test_copies_match_fleet(self):
        management = (FLEET_SCRIPTS / "management.go").read_text()
        embedded = set(re.findall(r"//go:embed scripts/((?:install|uninstall)_\w+\.(?:sh|ps1))", management))
        self.assertEqual(embedded, {p.name for p in DEFAULT_SCRIPTS.iterdir()})
        for name in embedded:
            self.assertEqual((DEFAULT_SCRIPTS / name).read_bytes(), (FLEET_SCRIPTS / "scripts" / name).read_bytes(), name)


class HelpersTest(unittest.TestCase):
    def test_slugify(self):
        self.assertEqual(slugify("💻 Workstations"), "workstations")
        self.assertEqual(slugify("📱🔐 Personal mobile devices"), "personal-mobile-devices")
        self.assertEqual(slugify("macOS - FileVault: on"), "macos-filevault-on")

    def test_script_platform_reads_paths(self):
        self.assertEqual(_script_platform("ro.sh", 'RULE="/etc/udev/rules.d/99.rules"\nudevadm trigger\n')[0], "linux")
        self.assertEqual(_script_platform("x.sh", 'rm -rf "/Library/Foo"\n'), ("macos", None))
        self.assertIsNotNone(_script_platform("certs.sh", 'BUNDLE="${1:-/opt/orbit/certs.pem}"\n')[1])

    def test_url_names_version(self):
        self.assertTrue(_url_names_version("https://x.test/prod/7.1.5.4332/zoom_amd64.deb", "7.1.5.4332"))
        self.assertTrue(_url_names_version("https://x.test/slack-4.52.155-0.1.el8.x86_64.rpm", "4.52.155-0.1.el8"))
        self.assertTrue(_url_names_version("https://x.test/app_2.0.deb", "1:2.0-3"))
        self.assertFalse(_url_names_version("https://x.test/stable/1password-latest.deb", "8.11.16"))
        self.assertTrue(_url_names_version("https://x.test/anything.pkg", None))

    def test_fma_script_state(self):
        catalog = "".join(f"line {i}\n" for i in range(20))
        older = catalog.replace("line 3\n", "old line 3\n").replace("line 7\n", "old line 7\n")
        fma = {"uninstall_script": catalog}
        self.assertEqual(_fma_script_state({"uninstall_script": catalog + "\n"}, fma, "uninstall_script"), "default")
        self.assertEqual(_fma_script_state({"uninstall_script": older}, fma, "uninstall_script"), "older")
        self.assertEqual(_fma_script_state({"uninstall_script": "#!/bin/sh\nrm -rf /Applications/X.app\n"}, fma, "uninstall_script"), "custom")
        self.assertEqual(_fma_script_state({"uninstall_script": older, "uninstall_script_edited": True}, fma, "uninstall_script"), "custom")
        self.assertEqual(_fma_script_state({"uninstall_script": "x", "uninstall_script_edited": False}, fma, "uninstall_script"), "default")

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

    @unittest.skipUnless(POINTER.exists(), "not inside the Fleet repository")
    def test_agents_pointer_matches_skill(self):
        def frontmatter(path):
            return dict(re.findall(r"(?m)^([\w-]+): (.*)$", path.read_text(encoding="utf-8").split("---")[1]))
        skill, pointer = frontmatter(HERE.parents[1] / "SKILL.md"), frontmatter(POINTER)
        for key in ("name", "description", "compatibility"):
            self.assertEqual(pointer.get(key), skill.get(key), key)


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


SECRET = "wk-s3cr3t-value-1234"
SERVER_SECRET = "srv-s3cr3t-value-99"
FLEETD_PROFILE = (b'<?xml version="1.0" encoding="UTF-8"?>\n<plist version="1.0"><dict><key>PayloadDisplayName</key>'
                  b"<string>Fleetd config</string><key>EnrollSecret</key><string>" + SECRET.encode() + b"</string>"
                  b"<key>Note</key><string>costs $5</string></dict></plist>\n")
ENROLL_SCRIPT = f"#!/bin/sh\napt-get install -y fleet-osquery --enroll-secret {SERVER_SECRET}\n".encode()
FMA_CATALOG = "".join(f"line {i}\n" for i in range(20))
FMA_STORED = FMA_CATALOG.replace("line 3\n", "old line 3\n").replace("line 7\n", "old line 7\n")


def secret_span(content: bytes, secret: str, team_id) -> list:
    start = content.decode().index(secret)
    return [{"start": start, "end": start + len(secret), "source": {"type": "enroll_secret", "team": team_id, "index": 0}}]


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
                          {"profile_uuid": "b", "name": "Min password", "platform": "windows", "content": b"<Replace><Data>$8</Data></Replace>\n"},
                          {"profile_uuid": "c", "name": "Fleetd config", "platform": "darwin", "content": FLEETD_PROFILE,
                           "env_refs": secret_span(FLEETD_PROFILE, SECRET, 1)}],
             "assets": [], "android_certificates": [], "setup": {},
             "software": {"titles": [{"id": 10, "name": "Maccy", "fma": {"slug": "maccy/darwin", "install_script": "i", "uninstall_script": "u"},
                                      "software_package": {"name": "Maccy.zip", "platform": "darwin", "fleet_maintained_app_id": 5,
                                                           "self_service": True, "categories": ["Productivity"],
                                                           "install_script": "i", "uninstall_script": "u"}},
                                     {"id": 11, "name": "Rectangle", "fma": {"slug": "rectangle/darwin", "install_script": "i", "uninstall_script": FMA_CATALOG},
                                      "software_package": {"name": "Rectangle.dmg", "platform": "darwin", "fleet_maintained_app_id": 6,
                                                           "install_script": "i", "uninstall_script": FMA_STORED}}],
                          "setup": [], "setup_macos": []}},
            {"key": "2", "id": 2, "name": "Servers", "team": team("Servers"), "enroll_secret_count": 1, "labels": [],
             "policies": [{"id": 4, "name": "curl installed", "query": "SELECT 1;", "platform": "linux"},
                          {"id": 5, "name": "Zoom installed", "query": "SELECT 1;", "platform": "linux",
                           "install_software": {"software_title_id": 21, "name": "Zoom"}}],
             "reports": [],
             "scripts": [{"id": 4, "name": "update.sh", "content": b"#!/bin/sh\napt-get update\n"},
                         {"id": 5, "name": "enroll.sh", "content": ENROLL_SCRIPT, "env_refs": secret_span(ENROLL_SCRIPT, SERVER_SECRET, 2)}],
             "profiles": [], "assets": [], "android_certificates": [], "setup": {},
             "software": {"titles": [{"id": 20, "name": "Zoom", "software_package": {
                                          "name": "zoom_amd64.deb", "platform": "linux", "version": "7.1.5.4332",
                                          "url": "https://cdn.example.test/prod/7.1.5.4332/zoom_amd64.deb", "hash_sha256": "a" * 64,
                                          "install_script": default_script("install_deb.sh"), "uninstall_script": default_script("uninstall_deb.sh", "zoom")}},
                                     {"id": 21, "name": "Zoom", "software_package": {
                                          "name": "zoom_x86_64.rpm", "platform": "linux", "version": "7.1.5.4332",
                                          "url": "https://cdn.example.test/zoom-latest.rpm", "hash_sha256": "b" * 64,
                                          "install_script": "#!/bin/sh\nrpm -i --nodeps \"$INSTALLER_PATH\"\n",
                                          "uninstall_script": default_script("uninstall_rpm.sh", "zoom")}}],
                          "setup": [], "setup_macos": []}},
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
        ch.env_values = {"FLEET_WORKSTATIONS_ENROLL_SECRET": SECRET}
        ch.references()
        ch.reconcile()
        self.assertEqual([r for k in ("references", "reconcile") for r in ch.results[k]], [])

    def test_enroll_secret_in_profile_becomes_variable(self):
        rel = "platforms/macos/configuration-profiles/fleetd-config.mobileconfig"
        text = self.read(rel)
        self.assertNotIn(SECRET, text)
        self.assertIn("<string>$FLEET_WORKSTATIONS_ENROLL_SECRET</string>", text)
        self.assertIn(r"costs \$5", text)
        self.assertEqual(fleetctl_expand(text, {"FLEET_WORKSTATIONS_ENROLL_SECRET": SECRET}).encode(), FLEETD_PROFILE)
        self.assertEqual(next(f for f in self.plan["files"].values() if f["path"] == rel)["env_refs"], ["FLEET_WORKSTATIONS_ENROLL_SECRET"])
        self.assertNotIn(SECRET, repr({k: v for k, v in self.plan.items() if k != "inventory"}))

    def test_enroll_secret_in_script_is_flagged(self):
        notes = [n for n in self.plan["notes"] if n["level"] == "review" and n["item"].endswith("/enroll.sh")]
        self.assertEqual(len(notes), 1)
        self.assertIn("FLEET_SERVERS_ENROLL_SECRET", notes[0]["message"])
        self.assertNotIn(SERVER_SECRET, notes[0]["message"])

    def test_reconcile_fills_in_enroll_secret_variables(self):
        ch = Checker(self.root, self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        ch.reconcile()
        self.assertEqual([r["level"] for r in ch.results["reconcile"]], ["WARN"])
        ch = Checker(self.root, self.plan)
        ch.load_repo()
        ch.env_values = {"FLEET_WORKSTATIONS_ENROLL_SECRET": "a-different-secret"}
        ch.reconcile()
        self.assertIn("profile 'Fleetd config': sha256", "\n".join(r["message"] for r in ch.results["reconcile"]))

    def test_older_fma_scripts_left_out_and_flagged(self):
        self.assertNotIn("rectangle-uninstall.sh", self.read("fleets/workstations.yml"))
        self.assertTrue(any(n["level"] == "review" and "rectangle/darwin (uninstall)" in n["message"] for n in self.plan["notes"]))
        kept = Planner(snapshot(), self.layout, self.work, keep_fma_scripts=["rectangle/darwin"]).build()
        self.assertTrue(any(f["path"].endswith("/rectangle-uninstall.sh") for f in kept["files"].values()))

    def test_packages_skip_fleet_defaults_and_unpinnable_hashes(self):
        deb = self.read("platforms/linux/software/zoom-deb.package.yml")
        rpm = self.read("platforms/linux/software/zoom-rpm.package.yml")
        self.assertIn("hash_sha256", deb)
        self.assertNotIn("install_script", deb)
        self.assertNotIn("uninstall_script", deb)
        self.assertNotIn("hash_sha256", rpm)
        self.assertIn("install_script", rpm)
        self.assertNotIn("uninstall_script", rpm)
        self.assertIn("package_path: ../software/zoom-rpm.package.yml", self.read("platforms/linux/policies/zoom-installed.yml"))
        self.assertFalse((self.root / "platforms/linux/software/servers").exists())

    def test_profile_validation_only_when_fleet_reports_it(self):
        snap = snapshot()
        planner = Planner(snap, self.layout, self.work)
        self.assertIs(planner._global_controls(snap["config"]["mdm"])["enable_profile_validation"], True)
        del snap["config"]["mdm"]["enable_profile_validation"]
        self.assertNotIn("enable_profile_validation", planner._global_controls(snap["config"]["mdm"]))

    def test_plan_paths_stay_in_repo(self):
        placer = Placer(self.plan, self.layout, self.work)
        for bad in ("../outside.yml", "/etc/passwd", "platforms/../../x"):
            with self.assertRaises(ValueError):
                placer.target(bad)
        outside = self.tmp / "outside"
        outside.mkdir()
        (self.root / "escape").symlink_to(outside)
        with self.assertRaises(ValueError):
            placer.target("escape/file.yml")

    def test_reconcile_catches_content_changes(self):
        ch = Checker(self.root, self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        policy = self.root / "platforms/macos/policies/collect-logs.yml"
        policy.write_text(policy.read_text().replace("SELECT 1;", "SELECT 2;"))
        profile = self.root / "platforms/windows/configuration-profiles/Min password.xml"
        profile.write_text(profile.read_text().replace("8", "9"))
        ws = self.root / "fleets/workstations.yml"
        ws.write_text(ws.read_text().replace("slug: maccy/darwin", "slug: other/darwin"))
        ch.load_repo()
        ch.reconcile()
        messages = "\n".join(r["message"] for r in ch.results["reconcile"])
        self.assertIn("policy 'Collect logs': query", messages)
        self.assertIn("profile 'Min password': sha256", messages)
        self.assertIn("software 'fma:maccy/darwin' is in Fleet but not in the repo", messages)

    def test_place_is_repeatable(self):
        before = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        Placer(self.plan, self.layout, self.work).place()
        after = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)


if __name__ == "__main__":
    unittest.main()
