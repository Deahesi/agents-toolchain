"""Inspect both architectures without installing foreign-architecture packages."""

import json
from pathlib import Path
import sys
import subprocess

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from build_wheel import DIST, ROOT
from check_artifacts import check_deb

version = json.loads((DIST / "metadata.json").read_text(encoding="utf-8"))["version"]
expected = {"/usr/bin/atc", "/usr/share/licenses/agents-toolchain/LICENSE", "/usr/share/doc/agents-toolchain/README.md"}
for arch in ("amd64", "arm64"):
    check_deb(DIST / "agents-toolchain_{}_{}.deb".format(version, arch), arch, version)
for arch in ("x86_64", "aarch64"):
    package = DIST / "agents-toolchain-{}-1.{}.rpm".format(version, arch)
    files = subprocess.check_output(["rpm", "-qpl", str(package)], text=True).splitlines()
    assert set(files) == expected, files
    metadata = subprocess.check_output(["rpm", "-qp", "--queryformat", "%{VERSION}\n%{ARCH}\n", str(package)], text=True).splitlines()
    assert metadata == [version, arch], metadata
print("Both .deb and .rpm architectures have the expected metadata and file list.")
