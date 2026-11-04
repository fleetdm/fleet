"""Offline tests for the export scripts. Run: python3 -m unittest discover -s scripts/tests"""

import argparse
import json
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
from planner import ALIASES, Planner, _fma_script_state, _script_platform, _url_names_version, exact_name_ok, slugify  # noqa: E402
from yamlio import Doc, EnvRef, ServerText, _shell_name, dump, env_escape  # noqa: E402

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

    def test_fleet_secrets_are_escaped_outside_the_name_template(self):
        self.assertIn("\\$FLEET_SECRET_WIFI", dump({"description": "uses $FLEET_SECRET_WIFI"}))
        template = dump({"name_template": ServerText("mac-$FLEET_SECRET_SITE")})
        self.assertIn("$FLEET_SECRET_SITE", template)
        self.assertNotIn("\\$", template)

    def test_keys_that_read_as_numbers_are_quoted(self):
        backend = yaml_backend()
        if not backend:
            self.skipTest("no YAML parser")
        from checks import load_yaml
        for k in ("1.10", "010", "0x1F", "1e3", "1_000", "$HOME", "true", "pack-1"):
            with self.subTest(key=k):
                self.assertEqual(list(load_yaml(fleetctl_expand(dump({k: 1}), {}), backend)), [k])

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

    def test_session_pins_relative_paths(self):
        tmp = Path(tempfile.mkdtemp())
        try:
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl="bin/fleetctl", config="fleetctl.yml", context=None)
            gitops_export.session(args)
            saved = json.loads((tmp / "work/session.json").read_text())
            self.assertEqual(saved["fleetctl"], str(Path("bin/fleetctl").resolve()))
            self.assertEqual(saved["config"], str(Path("fleetctl.yml").resolve()))
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl="fleetctl", config=None, context=None)
            gitops_export.session(args)
            saved = json.loads((tmp / "work/session.json").read_text())
            self.assertEqual((saved["fleetctl"], saved["config"]), ("fleetctl", str(Path("fleetctl.yml").resolve())))
        finally:
            shutil.rmtree(tmp)

    def test_failed_export_leaves_no_stale_plan(self):
        tmp = Path(tempfile.mkdtemp())
        saved = gitops_export.Collector, gitops_export.Planner, gitops_export.Layout, gitops_export.say

        class Boom:
            fail_collect = False

            def __init__(self, *a, **kw):
                pass

            def collect(self):
                if Boom.fail_collect:
                    raise RuntimeError("boom")
                return {}

            def build(self):
                raise RuntimeError("boom")

        try:
            gitops_export.Collector = gitops_export.Planner = Boom
            gitops_export.Layout = type("L", (), {"load": staticmethod(lambda p: None)})
            gitops_export.say = lambda *a: None
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl=None, config=None, context=None,
                                      keep_fma_scripts=None)
            work = gitops_export.work_dir(tmp / "repo", args.work)
            (work / "layout.json").write_text("{}")
            for fail_collect in (False, True):
                Boom.fail_collect = fail_collect
                (work / "plan.json").write_text("{}")
                with self.subTest(fail_collect=fail_collect), self.assertRaises(RuntimeError):
                    gitops_export.cmd_export(args)
                self.assertFalse((work / "plan.json").exists())
        finally:
            gitops_export.Collector, gitops_export.Planner, gitops_export.Layout, gitops_export.say = saved
            shutil.rmtree(tmp)

    def test_preflight_names_the_missing_install(self):
        tmp, out, saved = Path(tempfile.mkdtemp()), [], gitops_export.say
        gitops_export.say = out.append
        try:
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl=str(tmp / "no-fleetctl"), config=None, context=None)
            self.assertEqual(gitops_export.cmd_preflight(args), 1)
            self.assertTrue(out[0].startswith("FAIL install fleetctl:"), out)
        finally:
            gitops_export.say = saved
            shutil.rmtree(tmp)

    def test_validation_results_are_tied_to_their_plan(self):
        tmp, saved = Path(tempfile.mkdtemp()), gitops_export.say
        gitops_export.say = lambda *a: None
        try:
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl=None, config=None, context=None)
            work = gitops_export.work_dir(tmp / "repo", args.work)
            plan = {"server": {"version": "4.99.0", "premium": True, "user": {}}, "fleets": [], "counts": {}, "notes": [], "files": {}, "env": []}
            (work / "plan.json").write_text(json.dumps(plan))
            (work / "validation.json").write_text(json.dumps({"passed": True, "plan_sha256": "0" * 64, "static": [], "references": [], "reconcile": []}))
            gitops_export.cmd_report(args)
            report = (work / "report.md").read_text()
            self.assertIn("Not run for this plan", report)
            self.assertNotIn("PASS", report)
        finally:
            gitops_export.say = saved
            shutil.rmtree(tmp)

    def test_validate_writes_results_when_a_step_raises(self):
        tmp, saved = Path(tempfile.mkdtemp()), (gitops_export.Checker, gitops_export.say)

        class Stops(Checker):
            def run_validator(self, offline):
                pass

            def reconcile(self):
                raise RuntimeError("boom")

        gitops_export.Checker, gitops_export.say = Stops, lambda *a: None
        try:
            (tmp / "repo").mkdir()
            args = argparse.Namespace(dir=str(tmp / "repo"), work=str(tmp / "work"), fleetctl=None, config=None, context=None,
                                      no_dry_run=True, offline=False, env_file=None)
            work = gitops_export.work_dir(tmp / "repo", args.work)
            (work / "plan.json").write_text(json.dumps({"fleets": [], "env": [], "files": {}}))
            self.assertEqual(gitops_export.cmd_validate(args), 1)
            res = json.loads((work / "validation.json").read_text())
            self.assertIn("validate stopped before it finished: RuntimeError: boom", [r["message"] for r in res["reconcile"]])
        finally:
            gitops_export.Checker, gitops_export.say = saved
            shutil.rmtree(tmp)

    def test_nothing_compared_fails(self):
        tmp = Path(tempfile.mkdtemp())
        try:
            ch = Checker(tmp, {"env": [], "files": {}})
            ch.backend = None
            self.assertFalse(ch.load_repo())
            self.assertFalse(ch.summary()["passed"])
        finally:
            shutil.rmtree(tmp)

    @unittest.skipUnless(TEMPLATES.exists(), "not inside the Fleet repository")
    def test_scaffold_rerun_keeps_pristine_clean(self):
        tmp, saved = Path(tempfile.mkdtemp()), (gitops_export.session, gitops_export.say)

        class FakeFleetctl:
            def new(self, directory, org_name, force=False):
                if Path(directory).exists():
                    raise AssertionError("scaffold should run fleetctl new into an empty folder")
                render_templates(Path(directory))
                return f"Created new Fleet GitOps repository at {directory}\nNext steps"

        target, work = tmp / "repo", tmp / "work"
        work.mkdir(mode=0o700)
        gitops_export.session = lambda args: (FakeFleetctl(), target, work)
        gitops_export.say = lambda *a: None
        try:
            args = argparse.Namespace(org_name="Test Org", force=True)
            self.assertEqual(gitops_export.cmd_scaffold(args), 0)
            (target / "fleets/servers.yml").write_text("name: Servers\n")
            (target / ".env").write_text("FLEET_X=secret\n")
            self.assertEqual(gitops_export.cmd_scaffold(args), 0)
            self.assertFalse((work / "pristine/fleets/servers.yml").exists())
            self.assertFalse((work / "pristine/.env").exists())
            self.assertTrue((work / "pristine/default.yml").exists())
            self.assertFalse((work / "new").exists())
        finally:
            gitops_export.session, gitops_export.say = saved
            shutil.rmtree(tmp)

    def test_collector_tolerates_endpoints_older_servers_lack(self):
        class API:
            def paged(self, path, key, params=None):
                raise FleetctlError("not found", status=404)

        self.assertEqual(Collector(api=API())._paged_if_present("/api/latest/fleet/custom_host_vitals", "custom_host_vitals"), [])

    def test_setup_step(self):
        # Error text as fleetctl 4.94 prints it.
        cases = {
            "Error: set the Fleet API address with: fleetctl config set --address https://fleet.example.com": "address",
            "couldn't read the address from fleetctl's config; run `fleetctl login`": "address",
            "GET /api/latest/fleet/me: Token missing. SSO is enabled for this Fleet instance.": "login",
            "GET /api/latest/fleet/me: Token invalid or session expired.\nError: unauthenticated, or invalid token": "login",
            'Error: GET /api/latest/fleet/version: do request: Get "https://x.invalid/api": dial tcp: lookup x.invalid: no such host': "network",
            "GET /api/latest/fleet/config: non 2XX return of 500": "other",
        }
        for error, step in cases.items():
            with self.subTest(error=error):
                self.assertEqual(gitops_export.setup_step(error), step)


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

    def test_values_yaml_would_misread(self):
        if not yaml_backend():
            self.skipTest("no YAML parser")

        class FakeAPI:
            def json(self, path, params=None):
                return {"ca": {"password": "p@ss: word #1"}, "url": "https://fleet.example.test"}

        f = self.tmp / ".env"
        f.write_text("FLEET_EXPORT_TEST_TOKEN=true\nFLEET_EXPORT_TEST_QUOTED='\"a: b\"'\n")
        plan = {"env": [{"name": "FLEET_EXPORT_TEST_PASSWORD", "source": {"type": "config", "pointer": ["ca", "password"]}, "used_in": ["default.yml"]},
                        {"name": "FLEET_EXPORT_TEST_URL", "source": {"type": "config", "pointer": ["url"]}, "used_in": ["default.yml"]},
                        {"name": "FLEET_EXPORT_TEST_TOKEN", "source": {"type": "mask"}, "used_in": ["default.yml"]},
                        {"name": "FLEET_EXPORT_TEST_QUOTED", "source": {"type": "mask"}, "used_in": ["default.yml"]},
                        {"name": "FLEET_EXPORT_TEST_KEY", "source": {"type": "mask", "object": True}, "used_in": ["default.yml"]}]}
        ch = Checker(self.tmp, plan)
        env, _ = ch.dry_run_env(FakeAPI(), f)
        doc = fleetctl_expand("ca:\n  password: $FLEET_EXPORT_TEST_PASSWORD\n", env)
        self.assertEqual(load_yaml(doc, yaml_backend())["ca"]["password"], "p@ss: word #1")
        self.assertEqual(env["FLEET_EXPORT_TEST_URL"], "https://fleet.example.test")
        results = [(r["level"], r["message"].split(":")[0]) for r in ch.results["reconcile"]]
        self.assertEqual(results, [("WARN", "FLEET_EXPORT_TEST_PASSWORD"), ("FAIL", "FLEET_EXPORT_TEST_TOKEN")])
        self.assertNotIn("p@ss", repr(ch.results))

    def test_no_quotes_for_values_profiles_also_use(self):
        if not yaml_backend():
            self.skipTest("no YAML parser")

        class FakeAPI:
            def json(self, path, params=None):
                return {"spec": {"secrets": [{"secret": "12345678"}]}}

        plan = {"env": [{"name": "FLEET_GLOBAL_ENROLL_SECRET", "source": {"type": "enroll_secret", "team": None, "index": 0},
                         "used_in": ["default.yml", "platforms/macos/configuration-profiles/fleetd.mobileconfig"]}]}
        ch = Checker(self.tmp, plan)
        env, _ = ch.dry_run_env(FakeAPI(), None)
        self.assertEqual(env["FLEET_GLOBAL_ENROLL_SECRET"], "12345678")
        self.assertEqual([r["level"] for r in ch.results["reconcile"]], ["FAIL"])
        self.assertIn("fleetd.mobileconfig", ch.results["reconcile"][0]["message"])
        self.assertNotIn("12345678", repr(ch.results))

    def test_api_source_and_leaked_values(self):
        class FakeAPI:
            def json(self, path, params=None):
                assert path == "/api/latest/fleet/software/titles/7" and params == {"fleet_id": 2}
                return {"software_title": {"software_package": {"url": "https://x.test/app.pkg?token=abcdef123456"}}}

        plan = {"env": [{"name": "FLEET_APP_URL", "used_in": ["platforms/macos/software/app.package.yml"],
                         "source": {"type": "api", "path": "/api/latest/fleet/software/titles/7", "params": {"fleet_id": 2},
                                    "pointer": ["software_title", "software_package", "url"]}}]}
        ch = Checker(self.tmp, plan)
        env, _ = ch.dry_run_env(FakeAPI(), None)
        self.assertEqual(env["FLEET_APP_URL"], "https://x.test/app.pkg?token=abcdef123456")
        (self.tmp / "notes.md").write_text("see https://x.test/app.pkg?token=abcdef123456\n")
        (self.tmp / ".env").write_text("FLEET_APP_URL=https://x.test/app.pkg?token=abcdef123456\n")
        ch.check_leaks()
        messages = [r["message"] for r in ch.results["reconcile"] if r["level"] == "FAIL"]
        self.assertEqual(len(messages), 1)
        self.assertIn("notes.md holds the value of $FLEET_APP_URL", messages[0])
        self.assertNotIn("abcdef123456", repr(ch.results))

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
        scopes = [{"profiles": [{"content": b"<string>wk-s3cr3t-value</string> and wk-s3cr3t-value"}, {"content": b"shortcake"}],
                   "scripts": [{"content": b"echo short"}], "assets": [{"content": b"\xff\xfe"}]}]
        c._mark_secrets(scopes)
        refs = scopes[0]["profiles"][0]["env_refs"]
        self.assertEqual([(r["start"], r["end"]) for r in refs], [(8, 23), (37, 52)])
        self.assertEqual(refs[0]["source"], {"type": "enroll_secret", "team": 1, "index": 0})
        self.assertNotIn("env_refs", scopes[0]["scripts"][0])
        self.assertNotIn("wk-s3cr3t-value", repr(refs))
        self.assertEqual(scopes[0]["scripts"][0]["short_secret_refs"], [{"type": "enroll_secret", "team": None, "index": 0}])
        self.assertNotIn("short_secret_refs", scopes[0]["profiles"][1])
        self.assertTrue(scopes[0]["profiles"][1]["short_secret_in_word"])
        self.assertEqual(c.warnings, [])
        self.assertNotIn("short", repr(scopes[0]["scripts"][0]["short_secret_refs"]))

    def test_marks_secrets_where_fleetctl_sends_files_as_is(self):
        c = Collector(api=None)
        long_src, short_src = {"type": "enroll_secret", "team": 1, "index": 0}, {"type": "enroll_secret", "team": None, "index": 0}
        c._secret_values = [("wk-s3cr3t-value", long_src), ("short", short_src)]
        profile = {"name": "Passcode", "content": b'{"Type": "x"}', "activation": b'{"Note": "wk-s3cr3t-value"}'}
        script = {"name": "enroll.sh", "content": b"fleetctl package --enroll-secret wk-s3cr3t-value"}
        setup = {"name": "setup.sh", "content": b"echo short"}
        title = {"name": "Acme", "packages": [{"install_script": "true"}, {"post_install_script": "echo wk-s3cr3t-value"}]}
        c._mark_secrets([{"profiles": [profile], "scripts": [script], "setup": {"script": setup}, "software": {"titles": [title]}}])
        self.assertNotIn("env_refs", profile)
        self.assertEqual(profile["plain_secret_refs"], [long_src])
        self.assertEqual(script["plain_secret_refs"], [long_src])
        self.assertNotIn("env_refs", script)
        self.assertEqual(setup["short_secret_refs"], [short_src])
        self.assertEqual(title["plain_secret_refs"], [long_src])
        self.assertNotIn("wk-s3cr3t-value", repr([profile["plain_secret_refs"], script["plain_secret_refs"], title["plain_secret_refs"]]))

    def test_marks_encoded_secrets_and_yaml_items(self):
        import base64 as b64
        c = Collector(api=None)
        secret = "wk+s3cr3t/value=="
        src = {"type": "enroll_secret", "team": 1, "index": 0}
        c._secret_values = [(secret, src)]
        json_escaped = {"name": "decl", "content": b'{"Secret": "wk+s3cr3t\\/value=="}'}
        url = {"name": "url", "content": b"<string>https://x.test/?s=wk%2Bs3cr3t%2Fvalue%3D%3D</string>"}
        data = {"name": "data", "content": b"<data>" + b64.b64encode(b"prefix " + secret.encode() + b" suffix") + b"</data>"}
        plain = {"name": "plain", "content": ("<string>" + secret + "</string>").encode(), "description": "uses " + secret}
        policy = {"name": "p", "query": f"SELECT 1 WHERE value = '{secret}'"}
        report, label = {"name": "r", "query": "SELECT 1", "description": secret}, {"name": "l", "query": secret}
        rule, eula = {"name": "r.yar", "contents": secret}, {"name": "terms.md", "content": secret.encode()}
        snap = {"config": {"agent_options": {"config": {"options": {"x": "a " + secret}}}, "yara_rules": [rule]},
                "eula": {"windows": eula}}
        scopes = [{"key": "global", "profiles": [json_escaped, url, data, plain], "policies": [policy], "reports": [report], "labels": [label]}]
        c._mark_secrets(scopes, snap)
        for item in (json_escaped, url, data, plain, policy, report, label, rule, eula):
            self.assertEqual(item.get("plain_secret_refs"), [src], item["name"])
        self.assertEqual(snap["setting_secret_paths"], {"global": [["agent_options", "config", "options", "x"]]})

    def test_marks_secrets_in_configurations_and_undecodable_bodies(self):
        c = Collector(api=None)
        src = {"type": "enroll_secret", "team": 1, "index": 0}
        c._secret_values = [("wk-s3cr3t-value", src)]
        utf16 = {"name": "UTF-16", "content": "\ufeff<string>wk-s3cr3t-value</string>".encode("utf-16-le")}
        binary = {"name": "Signed", "content": b"\x30\x82\xff<string>wk-s3cr3t-value</string>"}
        app = {"name": "Slack", "app_store_app": {"configuration": "<string>wk-s3cr3t-value</string>"}}
        ipa = {"name": "App", "software_package": {"name": "app.ipa", "configuration": "<string>wk-s3cr3t-value</string>"},
               "icon": b"wk-s3cr3t-value"}
        icon_only = {"name": "Icon", "software_package": {"name": "x.pkg"}, "icon": b"wk-s3cr3t-value"}
        dep = {"name": "ade.json", "enrollment_profile": {"configuration_web_url": "https://x.test/?s=wk-s3cr3t-value"}}
        c._mark_secrets([{"profiles": [utf16, binary], "software": {"titles": [app, ipa, icon_only]}, "setup": {"enrollment_profile": dep}}])
        for item in (utf16, binary, app, ipa, dep):
            self.assertEqual(item.get("plain_secret_refs"), [src], item["name"])
            self.assertNotIn("env_refs", item)
        self.assertNotIn("plain_secret_refs", icon_only)


class FleetDefaultsTest(unittest.TestCase):
    def test_recognizes_defaults_with_ids_filled_in(self):
        self.assertTrue(is_default_script("install_script", ".deb", default_script("install_deb.sh")))
        self.assertTrue(is_default_script("uninstall_script", "deb", default_script("uninstall_deb.sh", "slack-desktop")))
        pkg = default_script("uninstall_pkg.sh").replace("$PACKAGE_ID", "(\n  'com.example.a'\n  'com.example.b'\n)")
        self.assertTrue(is_default_script("uninstall_script", "pkg", pkg))
        upgrade = default_script("uninstall_msi_with_upgrade_code.ps1").replace('"$UPGRADE_CODE"', "'{ABC-123}'")
        self.assertTrue(is_default_script("uninstall_script", "msi", upgrade, {"UPGRADE_CODE": ["{ABC-123}"]}))

    def test_msi_default_follows_the_upgrade_code(self):
        product = default_script("uninstall_msi.ps1").replace("$PACKAGE_ID", "'{PROD-1}'")
        upgrade = default_script("uninstall_msi_with_upgrade_code.ps1").replace('"$UPGRADE_CODE"', "'{ABC-123}'")
        self.assertFalse(is_default_script("uninstall_script", "msi", product, {"UPGRADE_CODE": ["{ABC-123}"]}))
        self.assertTrue(is_default_script("uninstall_script", "msi", product))
        self.assertFalse(is_default_script("uninstall_script", "msi", upgrade))

    def test_retargeted_defaults_are_not_defaults(self):
        ids = {"PACKAGE_ID": ["zoom"]}
        self.assertTrue(is_default_script("uninstall_script", "deb", default_script("uninstall_deb.sh", "zoom"), ids))
        self.assertFalse(is_default_script("uninstall_script", "deb", default_script("uninstall_deb.sh", "zoom-beta"), ids))
        upgrade = default_script("uninstall_msi_with_upgrade_code.ps1").replace('"$UPGRADE_CODE"', "'{ABC-123}'")
        self.assertTrue(is_default_script("uninstall_script", "msi", upgrade, {"UPGRADE_CODE": ["{ABC-123}"]}))
        self.assertFalse(is_default_script("uninstall_script", "msi", upgrade, {"UPGRADE_CODE": ["{DEF-456}"]}))

    def test_glob_characters_can_be_file_names(self):
        self.assertTrue(exact_name_ok("[AllowSpotlightCollection].xml"))
        self.assertTrue(exact_name_ok("{braces}.sh"))
        self.assertFalse(exact_name_ok("star*.sh"))

    def test_fleetd_script_is_never_a_default(self):
        self.assertFalse(is_default_script("install_script", "pkg", default_script("install_pkg_fleetd.sh")))
        self.assertTrue(is_default_script("install_script", "pkg", default_script("install_pkg.sh")))

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
        self.assertTrue(_url_names_version("https://x.test/releases/v8.11.16/app.dmg", "8.11.16"))
        self.assertFalse(_url_names_version("https://8.11.16.x.test/latest.deb", "8.11.16"))
        self.assertFalse(_url_names_version("https://x.test/latest.deb?v=8.11.16", "8.11.16"))
        self.assertFalse(_url_names_version("https://x.test/app-18.1.deb", "8.1"))
        self.assertFalse(_url_names_version("https://x.test/app-8.1.2.deb", "8.1"))
        self.assertFalse(_url_names_version("https://x.test/8.1/latest.rpm", "8.1"))
        self.assertFalse(_url_names_version("https://x.test/app-current.pkg", None))
        self.assertFalse(_url_names_version("https://x.test/anything.pkg", None))
        self.assertFalse(_url_names_version("https://x.test/1.0/anything.pkg", ""))
        self.assertFalse(_url_names_version("https://vendor.example/latest/app.rpm", None))
        self.assertFalse(_url_names_version("https://x.test/stable/app.deb", "8.11.16"))
        self.assertTrue(_url_names_version("https://x.test/linux/stable/1password-8.11.16.deb", "8.11.16"))

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

    @unittest.skipUnless(DEPRECATIONS.exists(), "not inside the Fleet repository")
    def test_reconcile_knows_deprecated_paths(self):
        from checks import DEPRECATED_PATHS
        pairs = re.findall(r'\{"([\w.]+)", "([\w.]+)"\}', DEPRECATIONS.read_text())
        paths = {old: new for old, new in pairs if "." in old or old in ("team_settings", "queries")}
        self.assertEqual(DEPRECATED_PATHS, paths)

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


def fleet_scope(fleet_id, name, **kw):
    sc = {"key": str(fleet_id), "id": fleet_id, "name": name, "team": team(name), "enroll_secret_count": 0, "labels": [],
          "policies": [], "reports": [], "scripts": [], "profiles": [], "assets": [], "android_certificates": [], "setup": {},
          "software": {"titles": [], "setup": [], "setup_macos": []}}
    sc.update(kw)
    return sc


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
            "features": {"enable_host_users": True}, "webhook_settings": {}, "sso_settings": {},
            "integrations": {"jira": [{"url": "https://jira.example.test", "username": "fleet", "api_token": "********",
                                       "project_key": "IT", "enable_failing_policies": True}]},
            "agent_options": {"config": {"options": {"distributed_interval": 10}}},
            "mdm": {"windows_enabled_and_configured": True, "enable_profile_validation": True,
                    "macos_updates": {"minimum_version": "latest", "deadline_days": 14}},
        },
        "enroll_secret_count": 1, "custom_host_vitals": [{"name": "Asset tag"}],
        "certificate_authorities": {"digicert": [{"name": "WIFI", "url": "https://one.digicert.test", "api_token": "********",
                                                  "profile_id": "7", "certificate_common_name": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}]},
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
             "scripts": [{"id": 4, "name": "update.sh", "content": b"#!/bin/sh\napt-get update\n"}],
             "profiles": [], "assets": [], "android_certificates": [], "setup": {},
             "software": {"titles": [{"id": 20, "name": "zoom", "software_package": {
                                          "name": "zoom_amd64.deb", "platform": "linux", "version": "7.1.5.4332",
                                          "url": "https://cdn.example.test/prod/7.1.5.4332/zoom_amd64.deb", "hash_sha256": "a" * 64,
                                          "install_script": default_script("install_deb.sh"), "uninstall_script": default_script("uninstall_deb.sh", "zoom")}},
                                     {"id": 21, "name": "zoom", "software_package": {
                                          "name": "zoom_x86_64.rpm", "platform": "linux", "version": "7.1.5.4332",
                                          "url": "https://cdn.example.test/zoom-latest.rpm", "hash_sha256": "b" * 64,
                                          "install_script": "#!/bin/sh\nrpm -i --nodeps \"$INSTALLER_PATH\"\n",
                                          "uninstall_script": default_script("uninstall_rpm.sh", "zoom")}}],
                          "setup": [], "setup_macos": []}},
        ],
    }


def rich_snapshot():
    """The test snapshot with most of what an export can carry turned on."""
    PNG = b"\x89PNG\r\n\x1a\n" + b"0" * 20
    snap = snapshot()
    cfg = snap["config"]
    cfg["org_info"].update({"org_logo_url_dark_mode": "https://fleet.example.test/api/latest/fleet/logo?mode=dark", "org_logo_url_light_mode": "https://cdn.test/l.png"})
    snap["logos"] = {"dark": PNG}
    cfg["features"] = {"enable_host_users": True, "enable_software_inventory": False, "additional_queries": {"q": "SELECT 1"},
                       "historical_data": {"uptime": True, "vulnerabilities": False}}
    cfg["fleet_desktop"] = {"transparency_url": "https://t.test"}
    cfg["host_expiry_settings"] = {"host_expiry_enabled": True, "host_expiry_window": 30}
    cfg["webhook_settings"] = {"interval": "24h", "host_status_webhook": {"enable_host_status_webhook": True, "destination_url": "https://hook.test/x",
                                                                           "host_percentage": 5, "days_count": 2}}
    cfg["sso_settings"] = {"enable_sso": True, "entity_id": "fleet", "idp_name": "Okta", "metadata": "<xml/>", "enable_jit_provisioning": True}
    cfg["yara_rules"] = [{"name": "r1.yar", "contents": "rule x { condition: true }"}]
    cfg["mdm"].update({"apple_server_url": "https://mdm.test", "end_user_authentication": {"entity_id": "e", "idp_name": "Okta", "metadata_url": "https://m.test"},
                       "apple_business": [{"organization_name": "Org", "macos_fleet": "💻 Workstations", "ios_fleet": "", "ipados_fleet": ""}],
                       "windows_automatic_enrollment": {"default_team": "Servers"},
                       "windows_migration_enabled": True, "macos_migration": {"enable": True, "mode": "voluntary", "webhook_url": "https://mig.test"},
                       "apple_account_provisioning": {"oauth_idp_token_url": "https://idp.test/token", "oauth_idp_client_id": "cid", "oauth_idp_client_secret": "********"},
                       "macos_updates": {"minimum_version": "", "update_new_hosts": True}, "windows_updates": {"deadline_days": 0, "grace_period_days": 2},
                       "setup_experience": {"enable_end_user_authentication": True, "require_all_software_macos": True},
                       "enable_recovery_lock_password": True, "name_template": "mac-$FLEET_VAR_HOST_HARDWARE_SERIAL"})
    snap["vpp_tokens"] = [{"location": "Loc", "fleets": []}]
    snap["eula"] = {"macos": {"name": "eula.pdf", "content": b"%PDF-1.4"}}
    snap["graph_credentials"] = [{"tenant_id": "t", "client_id": "c"}]
    snap["custom_host_vitals"] = [{"name": "Asset tag"}]
    un = next(s for s in snap["scopes"] if s["key"] == "unassigned")
    un["setup"] = {"script": {"name": "setup.sh", "content": b"#!/bin/sh\necho hi\n"},
                   "enrollment_profile": {"name": "ade.json", "enrollment_profile": {"profile_name": "ADE", "skip_setup_items": ["Location"]}}}
    un["android_certificates"] = [{"name": "wifi", "certificate_authority_name": "WIFI", "subject_name": "CN=x"}]
    ws = next(s for s in snap["scopes"] if s["key"] == "1")
    ws["team"]["webhook_settings"] = {"failing_policies_webhook": {"enable_failing_policies_webhook": False, "destination_url": "", "policy_ids": [], "host_batch_size": 0}}
    ws["team"]["integrations"] = {"google_calendar": {"enable_calendar_events": True, "webhook_url": "https://cal.test"}}
    ws["team"]["features"] = {"enable_host_users": False, "enable_software_inventory": True}
    ws["software"]["titles"].append({"id": 12, "name": "Slack", "icon": PNG, "app_store_app": {"app_store_id": "803453959", "platform": "ios",
                                     "self_service": True, "auto_update_enabled": True, "auto_update_window_start": "02:00", "auto_update_window_end": "04:00"}})
    ws["software"]["setup"] = [{"id": 10, "software_package": {"install_during_setup": True, "name": "Maccy.zip"}}]
    ws["profiles"].append({"profile_uuid": "d", "name": "Passcode", "platform": "darwin", "content": b'{"Type": "com.apple.configuration.passcode.settings"}',
                           "activation": b'{"Type": "com.apple.activation.simple"}'})
    return snap


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

    def replan(self, snap, **kw):
        shutil.rmtree(self.work / "blobs")
        plan = Planner(snap, self.layout, self.work, **kw).build()
        Placer(plan, self.layout, self.work).place()
        return plan

    def reconcile(self, plan=None):
        ch = Checker(self.root, plan or self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        ch.env_values = {"FLEET_WORKSTATIONS_ENROLL_SECRET": SECRET}
        ch.reconcile()
        return ch.results["reconcile"]

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

    def test_script_with_enroll_secret_isnt_exported(self):
        snap = snapshot()
        servers = next(s for s in snap["scopes"] if s["key"] == "2")
        servers["scripts"].append({"id": 5, "name": "enroll.sh", "content": ENROLL_SCRIPT,
                                   "plain_secret_refs": [{"type": "enroll_secret", "team": 2, "index": 0}]})
        plan = self.replan(snap)
        notes = [n for n in plan["notes"] if n["level"] == "gap" and n["item"] == "enroll.sh"]
        self.assertEqual(len(notes), 1)
        self.assertIn("FLEET_SERVERS_ENROLL_SECRET", notes[0]["message"])
        self.assertNotIn(SERVER_SECRET, notes[0]["message"])
        self.assertFalse(list(self.root.rglob("enroll.sh")))
        self.assertFalse(any(SERVER_SECRET.encode() in p.read_bytes() for p in (self.work / "blobs").iterdir()))
        self.assertIn("script 'enroll.sh' is in Fleet but not in the repo", "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_enroll_secret_in_activation_setup_script_or_software_isnt_exported(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["profiles"].append({"profile_uuid": "d", "name": "Passcode", "platform": "darwin", "content": b'{"Type": "com.apple.configuration.passcode.settings"}',
                               "activation": f'{{"Type": "com.apple.activation.simple", "Note": "{SECRET}"}}'.encode()})
        ws["setup"] = {"script": {"name": "setup.sh", "content": f"#!/bin/sh\necho {SECRET}\n".encode()}}
        ws["software"]["titles"].append({"id": 12, "name": "Acme", "software_package": {
            "name": "acme.pkg", "platform": "darwin", "version": "1.0", "url": "https://x.test/1.0/acme.pkg", "hash_sha256": "c" * 64,
            "install_script": f"installer -pkg \"$INSTALLER_PATH\" -target / && echo {SECRET}\n"}})
        c = Collector(api=None)
        c._secret_values = [(SECRET, {"type": "enroll_secret", "team": 1, "index": 0})]
        c._mark_secrets(snap["scopes"])
        plan = self.replan(snap)
        gaps = {n["item"] for n in plan["notes"] if n["level"] == "gap" and "FLEET_WORKSTATIONS_ENROLL_SECRET" in n["message"]}
        self.assertEqual(gaps, {"Passcode", "setup_experience.macos_script (setup.sh)", "Acme"})
        written = [p for p in list(self.root.rglob("*")) + list((self.work / "blobs").iterdir()) if p.is_file()]
        self.assertFalse([p for p in written if SECRET.encode() in p.read_bytes()])
        self.assertNotIn("macos_script", self.read("fleets/workstations.yml"))
        messages = "\n".join(r["message"] for r in self.reconcile(plan))
        self.assertIn("profile 'Passcode' is in Fleet but not in the repo", messages)
        self.assertIn("software 'package:" + "c" * 64 + "' is in Fleet but not in the repo", messages)
        self.assertNotIn(SECRET, json.dumps(plan))

    def test_enroll_secret_in_app_configuration_asset_or_ade_profile_isnt_exported(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        config = f"<dict><key>secret</key><string>{SECRET}</string></dict>"
        ws["software"]["titles"].append({"id": 12, "name": "Slack", "app_store_app": {
            "app_store_id": "803453959", "platform": "ios", "configuration": config}})
        ws["assets"] = [{"name": "enroll", "content": b"\xff\xfe" + SECRET.encode("utf-16-le")}]
        ws["setup"] = {"enrollment_profile": {"name": "ade.json", "enrollment_profile": {"configuration_web_url": f"https://x.test/?s={SECRET}"}}}
        ws["profiles"][0]["content"] += b"<!-- abc12345x -->"
        c = Collector(api=None)
        c._secret_values = [(SECRET, {"type": "enroll_secret", "team": 1, "index": 0}), ("abc12", {"type": "enroll_secret", "team": 1, "index": 1})]
        c._mark_secrets(snap["scopes"])
        plan = self.replan(snap)
        gaps = {n["item"] for n in plan["notes"] if n["level"] == "gap" and "FLEET_WORKSTATIONS_ENROLL_SECRET" in n["message"]}
        self.assertEqual(gaps, {"Slack", "enroll", "setup_experience.apple_setup_assistant (ade.json)"})
        warns = [n for n in plan["notes"] if n["level"] == "warn"]
        self.assertEqual([n["item"] for n in warns], ["Firewall on"])
        written = [p for p in list(self.root.rglob("*")) + list((self.work / "blobs").iterdir()) if p.is_file()]
        for needle in (SECRET.encode(), SECRET.encode("utf-16-le")):
            self.assertFalse([p for p in written if needle in p.read_bytes()])
        messages = "\n".join(r["message"] for r in self.reconcile(plan))
        self.assertIn("software 'app:803453959:ios' is in Fleet but not in the repo", messages)
        self.assertIn("DDM asset 'enroll' is in Fleet but not in the repo", messages)

    def test_reconcile_reads_sso_keys_fleet_leaves_out(self):
        snap = snapshot()
        snap["flags"]["premium"] = False
        snap["scopes"] = [snap["scopes"][0]]
        snap["config"]["sso_settings"] = {"enable_sso": True, "entity_id": "fleet", "idp_name": "Okta",
                                          "metadata_url": "https://idp.example.test/metadata", "enable_jit_provisioning": False}
        plan = self.replan(snap)
        default = self.root / "default.yml"
        text = default.read_text()
        self.assertNotIn("enable_jit_provisioning", text)
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])
        default.write_text(text.replace("    enable_sso: true\n", "    enable_sso: true\n    enable_jit_provisioning: true\n", 1))
        self.assertIn("settings 'org_settings.sso_settings': enable_jit_provisioning is in the repo but not in Fleet",
                      "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_unassigned_policy_scripts_put_controls_in_unassigned_file(self):
        snap = snapshot()
        un = next(s for s in snap["scopes"] if s["key"] == "unassigned")
        un["policies"][0]["run_script"] = {"id": 1, "name": "hello.sh"}
        plan = self.replan(snap)
        self.assertEqual(plan["controls_file"], "fleets/unassigned.yml")
        unassigned = self.read("fleets/unassigned.yml")
        self.assertIn("path: ../platforms/macos/scripts/hello.sh", unassigned)
        self.assertIn("windows_enabled_and_configured: true", unassigned)
        self.assertNotRegex(self.read("default.yml"), r"(?m)^controls:")
        self.assertIn("run_script:\n    path: ../scripts/hello.sh", self.read("platforms/macos/policies/unassigned-check.yml"))
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])

    def test_credentials_in_settings_and_urls_become_variables(self):
        snap = snapshot()
        snap["config"]["agent_options"]["config"]["options"]["aws_secret_access_key"] = "AKIAEXAMPLESECRET0"
        snap["config"]["agent_options"]["config"]["options"]["enroll_secret_path"] = "/etc/secret"
        srv = next(s for s in snap["scopes"] if s["key"] == "2")
        token_url = "https://cdn.example.test/prod/7.1.5.4332/zoom_amd64.deb?token=tok-123456789"
        srv["software"]["titles"][0]["software_package"]["url"] = token_url
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["profiles"].append({"profile_uuid": "w", "name": "Wi-Fi", "platform": "darwin",
                               "content": b"<plist><dict><key>PayloadDisplayName</key><string>Wi-Fi</string>"
                                          b"<key>Password</key><string>hunter22</string></dict></plist>\n"})
        ws["policies"].append({"id": 8, "name": "Secret check", "query": f"SELECT 1 WHERE s = '{SECRET}';", "platform": "darwin"})
        c = Collector(api=None)
        c._secret_values = [(SECRET, {"type": "enroll_secret", "team": 1, "index": 0})]
        c._mark_secrets(snap["scopes"], snap)
        plan = self.replan(snap)
        default = self.read("default.yml")
        self.assertIn("aws_secret_access_key: $FLEET_AGENT_OPTIONS_AWS_SECRET_ACCESS_KEY", default)
        self.assertIn("enroll_secret_path: /etc/secret", default)
        self.assertIn("url: $FLEET_ZOOM_URL", self.read("platforms/linux/software/zoom-deb.package.yml"))
        env = {e["name"]: e["source"] for e in plan["env"]}
        self.assertEqual(env["FLEET_AGENT_OPTIONS_AWS_SECRET_ACCESS_KEY"],
                         {"type": "config", "pointer": ["agent_options", "config", "options", "aws_secret_access_key"]})
        self.assertEqual(env["FLEET_ZOOM_URL"]["pointer"], ["software_title", "software_package", "url"])
        dump_ = json.dumps(plan)
        for needle in ("AKIAEXAMPLESECRET0", "tok-123456789", SECRET):
            self.assertNotIn(needle, dump_)
        self.assertTrue(any(n["level"] == "gap" and n["item"] == "Secret check" for n in plan["notes"]))
        self.assertTrue(any(n["level"] == "warn" and n["item"] == "Wi-Fi" and "Password" in n["message"] for n in plan["notes"]))
        messages = "\n".join(r["message"] for r in self.reconcile(plan) if r["level"] == "FAIL")
        self.assertEqual(messages, "fleets/workstations.yml: policy 'Secret check' is in Fleet but not in the repo (applying would delete it)")

    def test_names_fleetctl_rejects_arent_exported(self):
        snap = snapshot()
        g = snap["scopes"][0]
        g["labels"].append({"name": "Équipe Macs", "query": "SELECT 1", "label_membership_type": "dynamic"})
        g["reports"].append({"id": 2, "name": "Users’ apps", "query": "SELECT 1;", "platform": "", "interval": 60})
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["name"] = ws["team"]["name"] = "No-Team"
        plan = self.replan(snap)
        gaps = {n["item"] for n in plan["notes"] if n["level"] == "gap" and "ASCII" in n["message"]}
        self.assertEqual(gaps, {"Équipe Macs", "Users’ apps"})
        self.assertNotIn("fleets/no-team.yml", [f["file"] for f in plan["fleets"]])

    def test_policy_installs_the_package_it_picked(self):
        snap = snapshot()
        srv = next(s for s in snap["scopes"] if s["key"] == "2")
        base = {"platform": "darwin", "install_script": "installer -pkg \"$INSTALLER_PATH\" -target /\n"}
        first = dict(base, installer_id=31, name="acme-arm.pkg", version="2.0", url="https://x.test/latest/acme-arm.pkg", hash_sha256="d" * 64)
        second = dict(base, installer_id=32, name="acme-intel.pkg", version="2.0", url="https://x.test/latest/acme-intel.pkg", hash_sha256="e" * 64)
        srv["software"]["titles"].append({"id": 30, "name": "Acme", "software_package": first, "packages": [first, second]})
        srv["policies"].append({"id": 9, "name": "Acme installed", "query": "SELECT 1;", "platform": "darwin",
                                "install_software": {"software_title_id": 30, "software_package_id": 32, "name": "Acme"}})
        plan = self.replan(snap)
        policy = self.read("platforms/macos/policies/acme-installed.yml")
        self.assertIn("hash_sha256: " + "e" * 64, policy)
        package = self.read("platforms/macos/software/acme.package.yml")
        self.assertIn("e" * 64, package)
        self.assertNotIn("d" * 64, package)
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])

    def test_fleetd_install_script_is_kept(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        fleetd = default_script("install_pkg_fleetd.sh")
        for tid, name, bundle in ((13, "Acme", "com.acme.app"), (14, "Fleet osquery", "com.fleetdm.orbit")):
            slug = slugify(name)
            ws["software"]["titles"].append({"id": tid, "name": name, "bundle_identifier": bundle, "software_package": {
                "name": f"{slug}.pkg", "platform": "darwin", "version": "1.0", "url": f"https://x.test/1.0/{slug}.pkg",
                "hash_sha256": str(tid) * 32, "install_script": fleetd}})
        plan = self.replan(snap)
        self.assertTrue(list(self.root.rglob("acme-install.sh")))
        self.assertTrue(list(self.root.rglob("fleet-osquery-install.sh")))
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])

    def test_missing_referenced_files_fail_without_stopping_the_check(self):
        hello = self.root / "platforms/macos/scripts/hello.sh"
        hello.unlink()
        hello.mkdir()
        (self.root / "platforms/macos/configuration-profiles/firewall-on.mobileconfig").unlink()
        ch = Checker(self.root, self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        ch.env_values = {"FLEET_WORKSTATIONS_ENROLL_SECRET": SECRET}
        ch.references()
        ch.reconcile()
        refs = "\n".join(r["message"] for r in ch.results["references"])
        self.assertIn("points at platforms/macos/scripts/hello.sh, which isn't a file", refs)
        self.assertIn("points at platforms/macos/configuration-profiles/firewall-on.mobileconfig, which doesn't exist", refs)
        messages = "\n".join(r["message"] for r in ch.results["reconcile"])
        self.assertIn("script 'hello.sh': its file is missing from the repo", messages)
        self.assertIn("profile 'Firewall on' is in Fleet but not in the repo", messages)

    def test_report_lists_exported_warnings_apart_from_gaps(self):
        notes = [{"level": "warn", "scope": "Servers", "item": "enroll.sh", "message": "check it"},
                 {"level": "warn", "scope": "Servers", "item": "Acme", "message": "check that too"},
                 {"level": "gap", "scope": "Servers", "item": "Acme", "message": "left out"}]
        report = gitops_export.render_report(self.root, dict(self.plan, notes=notes), None)
        exported, missing = report.split("## Exported, check before committing")[1].split("## Not represented in GitOps")
        self.assertIn("enroll.sh: check it", exported)
        self.assertNotIn("Acme", exported)
        self.assertIn("Acme: left out", missing)
        self.assertNotIn("enroll.sh", missing)

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
        self.assertIn("profile 'Fleetd config': content differs", "\n".join(r["message"] for r in ch.results["reconcile"]))

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
        self.assertIn("profile 'Min password': content differs", messages)
        self.assertIn("software 'fma:maccy/darwin' is in Fleet but not in the repo", messages)

    def test_malformed_yaml_fails_without_stopping_the_check(self):
        servers = self.root / "fleets/servers.yml"
        servers.write_text(servers.read_text().replace("software:\n", "software: []\nx_software:\n", 1))
        ws = self.root / "fleets/workstations.yml"
        ws.write_text(ws.read_text().replace("    - path: ../platforms/macos/scripts/logs.sh\n",
                                             "    - path: ../platforms/macos/scripts/logs.sh\n    - paths: /etc/*.conf\n    - path: null\n", 1))
        deb = self.root / "platforms/linux/software/zoom-deb.package.yml"
        deb.write_text(deb.read_text().replace("- url:", "url:", 1).replace("\n  ", "\n"))
        (self.root / ".DS_Store").write_bytes(b"\x00")
        (self.root / ".gitignore").write_text((self.root / ".gitignore").read_text() + "local-notes.txt\n")
        (self.root / "local-notes.txt").write_text("mine")
        ch = Checker(self.root, self.plan)
        if not ch.load_repo():
            self.skipTest("no YAML parser")
        ch.env_values = {"FLEET_WORKSTATIONS_ENROLL_SECRET": SECRET}
        ch.references()
        ch.reconcile()
        refs = "\n".join(r["message"] for r in ch.results["references"])
        self.assertIn("points at /etc/*.conf, which is outside the repo", refs)
        self.assertNotIn(".DS_Store", refs)
        self.assertNotIn("local-notes.txt", refs)
        messages = "\n".join(r["message"] for r in ch.results["reconcile"])
        self.assertNotIn("couldn't be compared", messages)
        self.assertIn("fleets/servers.yml: software 'package:https://cdn.example.test/prod/7.1.5.4332/zoom_amd64.deb' is in Fleet", messages)

    def test_example_left_behind_by_a_plan_edit_is_removed(self):
        plan = json.loads(json.dumps(self.plan))
        label = next(f for f in plan["files"].values() if f["path"] == "labels/apple-silicon-macos-hosts.yml")
        label["path"] = "labels/apple-silicon.yml"
        Placer(plan, self.layout, self.work).place()
        self.assertFalse((self.root / "labels/apple-silicon-macos-hosts.yml").exists())
        self.assertTrue((self.root / "labels/apple-silicon.yml").exists())

    def test_failed_place_still_records_what_it_wrote(self):
        placer = Placer(self.plan, self.layout, self.work)

        def boom(fleet):
            raise OSError("disk full")
        placer._write_fleet = boom
        with self.assertRaises(OSError):
            placer.place()
        placed = json.loads((self.work / "placed.json").read_text())
        self.assertIn("platforms/macos/scripts/logs.sh", placed)
        self.assertIn("fleets/workstations.yml", placed)

    def test_rich_export_reconciles_cleanly(self):
        plan = self.replan(rich_snapshot())
        self.assertEqual([r["message"] for r in self.reconcile(plan) if r["level"] != "WARN"], [])

    def test_reconcile_fails_on_keys_it_cant_compare(self):
        default = self.root / "default.yml"
        default.write_text(default.read_text().replace("org_settings:\n", "org_settings:\n  smtp_settings:\n    enable_smtp: false\n"
                                                                              "  gitops:\n    gitops_mode_enabled: true\n", 1))
        servers = self.root / "fleets/servers.yml"
        servers.write_text(servers.read_text() + "queries:\n  - name: Extra\n    query: SELECT 1;\n")
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("default.yml: org_settings.smtp_settings is in the repo but isn't compared with Fleet", messages)
        self.assertIn("default.yml: org_settings.gitops is in the repo but isn't compared", messages)
        self.assertIn("fleets/servers.yml: queries is in the repo but isn't compared with Fleet (applying may change Fleet); "
                      "it's the old name of reports", messages)

    def test_variables_must_sit_where_planned(self):
        servers = self.root / "fleets/servers.yml"
        servers.write_text(servers.read_text().replace("secret: $FLEET_SERVERS_ENROLL_SECRET", "secret: $FLEET_WORKSTATIONS_ENROLL_SECRET"))
        default = self.root / "default.yml"
        default.write_text(default.read_text().replace("api_token: $FLEET_CA_WIFI_API_TOKEN", "api_token: written-out-token"))
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("fleets/servers.yml: settings.secrets[0].secret should be $FLEET_SERVERS_ENROLL_SECRET, "
                      "but the repo has $FLEET_WORKSTATIONS_ENROLL_SECRET there", messages)
        self.assertIn("default.yml: org_settings.certificate_authorities.digicert[0].api_token should be $FLEET_CA_WIFI_API_TOKEN, "
                      "but the repo has no variable there", messages)
        self.assertNotIn("written-out-token", messages)

    def test_strict_settings_and_omitted_defaults(self):
        plan = self.replan(rich_snapshot())
        default = self.root / "default.yml"
        text = default.read_text()
        self.assertIn("deadline_days: 0", text)
        default.write_text(text.replace("distributed_interval: 10", "distributed_interval: 10\n      utc: false")
                           .replace("    enable_software_inventory: false\n", "").replace("      vulnerabilities: false\n", "")
                           .replace("deadline_days: 0", "deadline_days:"))
        messages = "\n".join(r["message"] for r in self.reconcile(plan))
        self.assertIn("settings '(top level)': agent_options", messages)
        self.assertIn("settings 'org_settings.features': enable_software_inventory is true in the repo but false in Fleet", messages)
        self.assertIn("settings 'org_settings.features': historical_data", messages)
        self.assertIn("settings 'controls.windows_updates': deadline_days is null in the repo but 0 in Fleet", messages)

    def test_reconcile_compares_mdm_setup_and_certificates(self):
        plan = self.replan(rich_snapshot())
        default = self.root / "default.yml"
        default.write_text(default.read_text().replace('macos_fleet: "💻 Workstations"', "macos_fleet: Servers")
                           .replace("  setup_experience:\n", "  setup_experience:\n    macos_bootstrap_package: https://x.test/b.pkg\n", 1)
                           .replace("client_id: c\n", "client_id: d\n"))
        next(self.root.rglob("setup.sh")).write_text("#!/bin/sh\necho changed\n")
        (self.root / "platforms/all/org-logos/dark.png").write_bytes(b"changed")
        messages = "\n".join(r["message"] for r in self.reconcile(plan))
        for expected in ("settings 'org_settings.mdm': apple_business", "settings 'controls.setup_experience': macos_bootstrap_package",
                         "settings 'controls.setup_experience': macos_script", "settings 'org_settings': microsoft_graph_credentials",
                         "settings 'org_settings.org_info': org_logo_path_dark_mode"):
            self.assertIn(expected, messages)

    def test_reconcile_compares_item_fields_the_repo_adds(self):
        policy = self.root / "platforms/macos/policies/collect-logs.yml"
        policy.write_text(policy.read_text() + "  resend_configuration_profile: Firewall on\n")
        ws = self.root / "fleets/workstations.yml"
        ws.write_text(ws.read_text().replace("      - path: ../platforms/macos/configuration-profiles/firewall-on.mobileconfig\n",
                                             "      - path: ../platforms/macos/configuration-profiles/firewall-on.mobileconfig\n"
                                             "        activation: ../platforms/macos/scripts/logs.sh\n", 1)
                      .replace("controls:\n", "controls:\n  android_settings:\n    certificates:\n      - name: extra\n"
                                              "        certificate_authority_name: WIFI\n        subject_name: CN=x\n", 1))
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("policy 'Collect logs': resend_configuration_profile", messages)
        self.assertIn("profile 'Firewall on': activation differs", messages)
        self.assertIn("Android certificate 'extra' is in the repo but not in Fleet", messages)

    def test_place_is_repeatable(self):
        before = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        Placer(self.plan, self.layout, self.work).place()
        after = {p: p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)

    def test_restore_stays_in_repo(self):
        outside = self.tmp / "outside"
        outside.mkdir()
        shutil.rmtree(self.root / ".github")
        (self.root / ".github").symlink_to(outside)
        with self.assertRaises(ValueError):
            Placer(self.plan, self.layout, self.work).place()
        self.assertEqual(list(outside.iterdir()), [])

    def test_fleet_files_never_collide_or_linger(self):
        snap = snapshot()
        snap["scopes"].insert(2, fleet_scope(3, "Workstations"))
        files = {f.get("name"): f["file"] for f in self.replan(snap)["fleets"]}
        self.assertEqual(files["💻 Workstations"], "fleets/workstations.yml")
        self.assertEqual(files["Workstations"], "fleets/workstations-2.yml")

        snap = snapshot()
        snap["scopes"] = [s for s in snap["scopes"] if s["key"] != "1"]
        plan = self.replan(snap)
        self.assertFalse((self.root / "fleets/workstations.yml").exists())
        self.assertTrue((self.root / "fleets/servers.yml").exists())
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])
        (self.root / "fleets/extra.yml").write_text("name: Extra\n")
        self.assertIn("fleets/extra.yml isn't a fleet in Fleet", "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_label_folders_follow_fleet_files(self):
        snap = snapshot()
        label = {"name": "Ops", "query": "SELECT 1", "label_membership_type": "dynamic"}
        snap["scopes"] += [fleet_scope(5, "EMEA Ops", labels=[dict(label, fleet_id=5)]),
                           fleet_scope(6, "EMEA+Ops", labels=[dict(label, fleet_id=6)])]
        plan = self.replan(snap)
        files = {f.get("name"): f["file"] for f in plan["fleets"]}
        self.assertEqual((files["EMEA Ops"], files["EMEA+Ops"]), ("fleets/emea-ops.yml", "fleets/emea-ops-2.yml"))
        self.assertIn("paths: ../labels/emea-ops/*.yml", self.read("fleets/emea-ops.yml"))
        self.assertIn("paths: ../labels/emea-ops-2/*.yml", self.read("fleets/emea-ops-2.yml"))
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])

    def test_setup_experience_flags(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["team"]["mdm"] = {"setup_experience": {"macos_enable_bootstrap_package_during_manual_enrollment": True}}
        self.replan(snap)
        self.assertIn("macos_enable_bootstrap_package_during_manual_enrollment: true", self.read("fleets/workstations.yml"))

        snap["flags"]["apple_mdm"] = False
        srv = next(s for s in snap["scopes"] if s["key"] == "2")
        srv["team"]["mdm"] = {"macos_setup": {"require_all_software_windows": True}}
        plan = self.replan(snap)
        self.assertIn("require_all_software_windows: true", self.read("fleets/servers.yml"))
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])
        servers = self.root / "fleets/servers.yml"
        servers.write_text(servers.read_text().replace("require_all_software_windows: true", "require_all_software_windows: false"))
        self.assertIn("settings 'controls.setup_experience': require_all_software_windows",
                      "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_reconcile_compares_every_kind_and_setting(self):
        servers = self.root / "fleets/servers.yml"
        servers.write_text(servers.read_text().replace(
            "  windows_settings:\n    configuration_profiles:\n", "  windows_settings:\n    configuration_profiles:\n"
                                                                "      - path: ../platforms/windows/configuration-profiles/Min password.xml\n", 1))
        default = self.root / "default.yml"
        default.write_text(default.read_text().replace("distributed_interval: 10", "distributed_interval: 20")
                           .replace("deadline_days: 14", "deadline_days: 7"))
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("fleets/servers.yml: profile 'Min password' is in the repo but not in Fleet", messages)
        self.assertIn("default.yml: settings '(top level)': agent_options", messages)
        self.assertIn("default.yml: settings 'controls.macos_updates': deadline_days", messages)

    def test_profile_validation_left_out_means_true(self):
        default = self.root / "default.yml"
        text = default.read_text()
        self.assertIn("enable_profile_validation: true", text)
        default.write_text(text.replace("  enable_profile_validation: true\n", ""))
        self.assertEqual([r for r in self.reconcile() if r["level"] == "FAIL"], [])
        default.write_text(text.replace("enable_profile_validation: true", "enable_profile_validation: false"))
        self.assertIn("enable_profile_validation", "\n".join(r["message"] for r in self.reconcile()))

        snap = snapshot()
        snap["config"]["mdm"]["enable_profile_validation"] = False
        plan = self.replan(snap)
        text = default.read_text()
        self.assertIn("enable_profile_validation: false", text)
        default.write_text(text.replace("  enable_profile_validation: false\n", ""))
        self.assertIn("enable_profile_validation is true in the repo but false in Fleet", "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_reconcile_compares_integrations_and_certificate_authorities(self):
        default = self.root / "default.yml"
        text = default.read_text()
        self.assertIn("api_token: $FLEET_JIRA_API_TOKEN", text)
        default.write_text(text.replace("https://jira.example.test", "https://jira2.example.test")
                           .replace("https://one.digicert.test", "https://two.digicert.test"))
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("settings 'org_settings.integrations': jira", messages)
        self.assertIn("settings 'org_settings.certificate_authorities': digicert", messages)

        default.write_text(text.replace("api_token: $FLEET_CA_WIFI_API_TOKEN", "api_token: $FLEET_URL"))
        self.assertIn("settings 'org_settings.certificate_authorities': digicert", "\n".join(r["message"] for r in self.reconcile()))
        default.write_text(text)
        self.assertEqual([r for r in self.reconcile() if r["level"] == "FAIL"], [])

        snap = snapshot()
        del snap["config"]["integrations"]["jira"]
        snap["certificate_authorities"] = {"digicert": None, "hydrant": []}
        plan = self.replan(snap)
        text = default.read_text()
        self.assertIn("  integrations: {}\n", text)
        self.assertNotIn("certificate_authorities:", text)
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])
        default.write_text(text.replace("  integrations: {}\n", "  integrations:\n    zendesk:\n      - url: https://z.example.test\n"
                                                              "        api_token: $FLEET_URL\n"
                                                              "  certificate_authorities:\n    hydrant:\n      - name: H\n"
                                                              "        url: https://h.example.test\n"))
        messages = "\n".join(r["message"] for r in self.reconcile(plan))
        self.assertIn("settings 'org_settings.integrations': zendesk is in the repo but not in Fleet", messages)
        self.assertIn("settings 'org_settings.certificate_authorities': hydrant is in the repo but not in Fleet", messages)

    def test_short_enroll_secret_keeps_item_out(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["profiles"][0]["short_secret_refs"] = [{"type": "enroll_secret", "team": 1, "index": 0}]
        plan = self.replan(snap)
        notes = [n for n in plan["notes"] if n["level"] == "gap" and n["item"] == "Firewall on"]
        self.assertEqual(len(notes), 1)
        self.assertIn("FLEET_WORKSTATIONS_ENROLL_SECRET", notes[0]["message"])
        self.assertFalse((self.root / "platforms/macos/configuration-profiles/firewall-on.mobileconfig").exists())
        self.assertIn("profile 'Firewall on' is in Fleet but not in the repo", "\n".join(r["message"] for r in self.reconcile(plan)))

    def test_reconcile_compares_software_contents(self):
        policy = self.root / "platforms/linux/policies/zoom-installed.yml"
        policy.write_text(policy.read_text().replace("zoom-rpm.package.yml", "zoom-deb.package.yml"))
        (self.root / "platforms/linux/software/zoom-rpm-install.sh").write_text("#!/bin/sh\nrpm -i \"$INSTALLER_PATH\"\n")
        deb = self.root / "platforms/linux/software/zoom-deb.package.yml"
        (deb.parent / "zoom-deb-uninstall.sh").write_text(default_script("uninstall_deb.sh", "zoom"))
        deb.write_text(deb.read_text() + "  uninstall_script:\n    path: ./zoom-deb-uninstall.sh\n")
        messages = "\n".join(r["message"] for r in self.reconcile())
        self.assertIn("policy 'Zoom installed': install_software", messages)
        self.assertIn("install_script is", messages)
        self.assertNotIn("uninstall_script", messages)

    def test_reconcile_compares_app_configurations(self):
        snap = snapshot()
        ws = next(s for s in snap["scopes"] if s["key"] == "1")
        ws["software"]["titles"].append({"id": 12, "name": "Slack", "app_store_app": {
            "app_store_id": "803453959", "platform": "ios", "configuration": "<dict><key>k</key><string>v</string></dict>"}})
        plan = self.replan(snap)
        self.assertEqual([r for r in self.reconcile(plan) if r["level"] == "FAIL"], [])
        config = next(self.root.rglob("slack-ios-config.xml"))
        config.write_text(config.read_text().replace(">v<", ">w<"))
        self.assertIn("software 'app:803453959:ios': configuration", "\n".join(r["message"] for r in self.reconcile(plan)))


if __name__ == "__main__":
    unittest.main()
