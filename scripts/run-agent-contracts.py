#!/usr/bin/env python3
"""Run named Agent contracts and fail if the selected graph omits their tests."""

import argparse
import os
from pathlib import Path
import subprocess
import sys

MODULE = "github.com/yurika0211/lucky-agent/agent"
TESTS = [
    "TestEnsureRequestFitsEmptyPartsCannotHidePlainContent",
    "TestCompactionCannotBypassRequestBudget",
    "TestModelRequestBudgetFitsHistoryAndPreservesStructuredPayload",
    "TestModelRequestBudgetRejectsOversizedSchemasBeforeDispatch",
    "TestModelRequestBudgetPinsStructuredOnlyCurrentInput",
    "TestModelRequestBudgetSnapshotIsStableForTurn",
    "TestModelRequestBudgetSelectsSchemasAndRetainsTurnRequiredTools",
    "TestCompletionReviewRetainsEarlyFailureAndRejectsUnsupportedComplete",
    "TestCompletionFailureRequiresEquivalentLaterSuccess",
    "TestCompletionProofValidatesCriteriaAndFailureReferences",
    "TestInjectedMinimalToolLoopNeedsNoMemoryOrSchedulers",
    "TestCheckpointLegacyResultMigrationPreservesEquality",
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("graph", choices=["leaf", "workspace", "published"])
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    directory, target = (root / "agent", ".") if args.graph == "leaf" else (root, MODULE)
    env = dict(os.environ, GOWORK=str(root / "go.work") if args.graph == "workspace" else "off")
    pattern = "^(" + "|".join(TESTS) + ")$"
    result = subprocess.run(["go", "test", "-mod=readonly", "-list", pattern, target], cwd=directory, env=env, capture_output=True, text=True, check=True)
    missing = set(TESTS) - set(result.stdout.splitlines())
    if missing:
        print("Selected Agent graph is missing required contracts: " + ", ".join(sorted(missing)), file=sys.stderr)
        return 1
    print(f"Running {len(TESTS)} contracts in {args.graph} graph", flush=True)
    subprocess.run(["go", "test", "-mod=readonly", "-race", "-count=1", "-timeout", "10m", "-run", pattern, target], cwd=directory, env=env, check=True)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as error:
        if error.stdout:
            print(error.stdout, file=sys.stderr)
        if error.stderr:
            print(error.stderr, file=sys.stderr)
        sys.exit(error.returncode)
