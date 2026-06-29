#!/usr/bin/env python3
"""Summarize db-temp Terraform outputs in a human-friendly format."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict


def run_terraform_output(directory: Path) -> Dict[str, Any]:
    """Return the parsed JSON from `terraform output -json` in `directory`."""
    try:
        proc = subprocess.run(
            ["terraform", "output", "-json"],
            cwd=directory,
            check=True,
            capture_output=True,
            text=True,
        )
    except FileNotFoundError:
        print("terraform executable not found in PATH", file=sys.stderr)
        sys.exit(1)
    except subprocess.CalledProcessError as exc:
        stderr = (exc.stderr or "").strip()
        stdout = (exc.stdout or "").strip()
        message = stderr or stdout or "command failed"
        print(f"terraform output failed in {directory}:\n{message}", file=sys.stderr)
        sys.exit(exc.returncode or 1)

    try:
        return json.loads(proc.stdout or "{}")
    except json.JSONDecodeError as exc:
        print(f"failed to parse terraform output JSON in {directory}: {exc}", file=sys.stderr)
        sys.exit(1)


def summarize_databases(databases: Dict[str, Any]) -> str:
    """Format the database information."""
    lines: list[str] = []
    if not databases:
        lines.append("No databases found.")
        return "\n".join(lines)

    for name in sorted(databases):
        details = databases.get(name) or {}
        lines.append(f"Customer: {name}")
        cluster_endpoint = details.get("cluster_endpoint")
        reader_endpoint = details.get("cluster_reader_endpoint")
        engine = (
            details.get("cluster_engine_version_actual")
            or details.get("cluster_engine_version")
            or "unknown"
        )
        db_name = details.get("cluster_database_name") or "fleet"

        if cluster_endpoint:
            lines.append(f"  Cluster endpoint: {cluster_endpoint}")
        if reader_endpoint and reader_endpoint != cluster_endpoint:
            lines.append(f"  Reader endpoint:  {reader_endpoint}")
        lines.append(f"  Engine version:   {engine}")
        lines.append(f"  Database name:    {db_name}")

        instances = details.get("cluster_instances") or {}
        if instances:
            lines.append("  Instances:")
            for instance_name in sorted(instances):
                instance = instances.get(instance_name) or {}
                endpoint = instance.get("endpoint") or "n/a"
                instance_class = instance.get("instance_class") or "unknown"
                is_writer = instance.get("writer")
                role = "writer" if is_writer else "reader"
                lines.append(f"    - {instance_name} ({role}) @ {endpoint} [{instance_class}]")

        lines.append("")  # blank line between customers

    return "\n".join(lines).rstrip()


def summarize_developer_passwords(dev_passwords: Dict[str, Any]) -> str:
    """Format the developer credential information."""
    lines: list[str] = []
    if not dev_passwords:
        lines.append("No developer credentials found.")
        return "\n".join(lines)

    for name in sorted(dev_passwords):
        details = dev_passwords.get(name) or {}
        credentials = details.get("developer_passwords") or {}
        lines.append(f"Customer: {name}")
        if not credentials:
            lines.append("  (no developer credentials present)")
        else:
            for user, password in sorted(credentials.items()):
                lines.append(f"  - {user}: {password}")
        lines.append("")

    return "\n".join(lines).rstrip()


def main() -> None:
    script_path = Path(__file__).resolve()
    default_root = script_path.parent.parent

    parser = argparse.ArgumentParser(
        description="Display db-temp database details and developer credentials using terraform output -json.",
    )
    parser.add_argument(
        "--tf-dir",
        type=Path,
        default=default_root,
        help="Path to the db-temp Terraform root (default: %(default)s)",
    )
    parser.add_argument(
        "--module-dir",
        type=Path,
        default=default_root / "mysql_dev_access",
        help="Path to the mysql_dev_access module (used as a fallback for credentials).",
    )
    args = parser.parse_args()

    tf_output = run_terraform_output(args.tf_dir)

    databases = (tf_output.get("databases") or {}).get("value") or {}
    developer_passwords = (tf_output.get("developer_passwords") or {}).get("value")

    # Fallback: try pulling credentials directly from the module if not exposed at root.
    if developer_passwords is None and args.module_dir.exists():
        module_output = run_terraform_output(args.module_dir)
        developer_passwords = (module_output.get("developer_passwords") or {}).get("value") or {}
    else:
        developer_passwords = developer_passwords or {}

    print("=== Databases ===")
    print(summarize_databases(databases))
    print()
    print("=== Developer Credentials ===")
    print(summarize_developer_passwords(developer_passwords))


if __name__ == "__main__":
    main()
