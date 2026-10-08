"""GoReleaser post-build hook. Create a wheel using only Python's standard library."""

import base64
import csv
import hashlib
import io
from pathlib import Path
import re
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]
DIST = ROOT / "dist"
PLATFORMS = {
    ("windows", "amd64"): "win_amd64",
    ("windows", "arm64"): "win_arm64",
    ("darwin", "amd64"): "macosx_12_0_x86_64",
    ("darwin", "arm64"): "macosx_12_0_arm64",
    # CGO_ENABLED=0 produces static Linux executables with no libc dependency.
    ("linux", "amd64"): "manylinux_2_17_x86_64.musllinux_1_2_x86_64",
    ("linux", "arm64"): "manylinux_2_17_aarch64.musllinux_1_2_aarch64",
}


def build_wheel(version, binary, goos, goarch, destination=DIST / "wheels"):
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", version):
        raise ValueError("GoReleaser must supply a stable X.Y.Z version")
    platform_tag = PLATFORMS[(goos, goarch)]
    info_dir = "agents_toolchain-{}.dist-info".format(version)
    executable = "atc.exe" if goos == "windows" else "atc"
    files = {}
    for name in ("__init__.py", "__main__.py", "_cli.py"):
        files["agents_toolchain/" + name] = (ROOT / "build/python/agents_toolchain" / name).read_text(encoding="utf-8").replace("\r\n", "\n").encode()
    binary_name = "agents_toolchain/bin/" + executable
    files[binary_name] = Path(binary).read_bytes()
    readme = (ROOT / "README.md").read_text(encoding="utf-8").replace("\r\n", "\n")
    files[info_dir + "/METADATA"] = ("Metadata-Version: 2.4\nName: agents-toolchain\nVersion: {}\n"
        "Summary: {}\nAuthor: {}\nLicense-Expression: MIT\nLicense-File: LICENSE\n"
        "Requires-Python: >=3.10\nProject-URL: Source, https://github.com/Deahesi/agents-toolchain\n"
        "Description-Content-Type: text/markdown\n\n{}".format(version, "CLI for creating and running YAML-configured AI agents", "Deahesi", readme)).encode()
    tags = "".join("Tag: py3-none-{}\n".format(tag) for tag in platform_tag.split("."))
    files[info_dir + "/WHEEL"] = ("Wheel-Version: 1.0\nGenerator: agents-toolchain-goreleaser\nRoot-Is-Purelib: false\n" + tags).encode()
    files[info_dir + "/entry_points.txt"] = b"[console_scripts]\natc = agents_toolchain._cli:main\n"
    files[info_dir + "/licenses/LICENSE"] = (ROOT / "LICENSE").read_text(encoding="utf-8").replace("\r\n", "\n").encode()
    record = io.StringIO(newline="")
    writer = csv.writer(record, lineterminator="\n")
    for name, content in sorted(files.items()):
        digest = base64.urlsafe_b64encode(hashlib.sha256(content).digest()).rstrip(b"=").decode("ascii")
        writer.writerow((name, "sha256=" + digest, str(len(content))))
    writer.writerow((info_dir + "/RECORD", "", ""))
    files[info_dir + "/RECORD"] = record.getvalue().encode()
    destination = Path(destination)
    destination.mkdir(parents=True, exist_ok=True)
    wheel = destination / "agents_toolchain-{}-py3-none-{}.whl".format(version, platform_tag)
    with zipfile.ZipFile(wheel, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name, content in sorted(files.items()):
            entry = zipfile.ZipInfo(name, date_time=(1985, 10, 26, 8, 15, 0))
            entry.create_system = 3
            entry.compress_type = zipfile.ZIP_DEFLATED
            entry.external_attr = (0o100755 if name == binary_name else 0o100644) << 16
            archive.writestr(entry, content)
    return wheel


if __name__ == "__main__":
    if len(sys.argv) != 5:
        raise SystemExit("Usage: build_wheel.py VERSION BINARY GOOS GOARCH")
    print(build_wheel(*sys.argv[1:]))
