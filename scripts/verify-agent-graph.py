#!/usr/bin/env python3
"""Compare the checkout's Agent with the module actually selected by the launcher."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

MODULE = "github.com/yurika0211/lucky-agent/agent"
IGNORED_DIRS = {".git", "docs", "testdata", "__pycache__"}


def production_tree(directory):
    files = {}
    for path in directory.rglob("*"):
        relative = path.relative_to(directory)
        if not path.is_file() or any(part in IGNORED_DIRS or part.startswith(".") for part in relative.parts):
            continue
        if path.name.endswith("_test.go") or path.name.startswith(("README", "LICENSE", "go.work", "go.local.work")):
            continue
        files[relative.as_posix()] = hashlib.sha256(path.read_bytes()).hexdigest()
    if "go.mod" not in files or not any(name.endswith(".go") for name in files):
        raise ValueError(f"not an Agent source module: {directory}")
    return files


def compare(source, selected):
    local, published = production_tree(source), production_tree(selected)
    return [name for name in sorted(local.keys() | published.keys()) if local.get(name) != published.get(name)]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report-only", action="store_true", help="report development differences without failing")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    env = dict(os.environ, GOWORK="off")
    result = subprocess.run(["go", "list", "-mod=readonly", "-m", "-json", MODULE], cwd=root, env=env, check=True, capture_output=True, text=True)
    module = json.loads(result.stdout)
    if module.get("Replace"):
        raise ValueError("published graph must not replace the Agent module")
    directory = module.get("Dir")
    if not directory:
        result = subprocess.run(["go", "mod", "download", "-json", MODULE + "@" + module["Version"]], cwd=root, env=env, check=True, capture_output=True, text=True)
        directory = json.loads(result.stdout)["Dir"]
    differences = compare(root / "agent", Path(directory))
    report = {"module": MODULE, "selected_version": module["Version"], "equal": not differences, "different_files": differences}
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    if differences and not args.report_only:
        print("Publish the Agent source commit and update the launcher's selected version before release.", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"Agent graph verification failed: {error}", file=sys.stderr)
        sys.exit(1)
