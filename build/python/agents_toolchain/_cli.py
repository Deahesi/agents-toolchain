"""Run the bundled executable without changing its arguments or environment."""

import os
from pathlib import Path
import subprocess
import sys


def main():
    binary = Path(__file__).parent / "bin" / ("atc.exe" if os.name == "nt" else "atc")
    if not binary.is_file():
        print("atc: bundled binary is missing; reinstall agents-toolchain with pip.", file=sys.stderr)
        return 1
    args = [str(binary), *sys.argv[1:]]
    try:
        if os.name != "nt":
            os.execv(str(binary), args)
        child = subprocess.Popen(args)
        while True:
            try:
                return child.wait()
            except KeyboardInterrupt:
                # Windows sends console Ctrl+C to both processes. Let the CLI exit.
                continue
    except OSError as error:
        print("atc: could not start bundled binary: {}".format(error), file=sys.stderr)
        return 1
