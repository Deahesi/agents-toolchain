"""Validate the release's file lists, wheel records, platforms, and checksums."""

import base64
import csv
from email.parser import BytesParser
import hashlib
import io
import json
import sys
from pathlib import Path
import struct
import tarfile
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from build_wheel import DIST, PLATFORMS, ROOT


def check_wheel(filename, goos, goarch, version):
    info = "agents_toolchain-{}.dist-info/".format(version)
    binary = "agents_toolchain/bin/" + ("atc.exe" if goos == "windows" else "atc")
    expected = {"agents_toolchain/" + item for item in ("__init__.py", "__main__.py", "_cli.py")}
    expected.update({binary, *(info + item for item in ("METADATA", "WHEEL", "entry_points.txt", "licenses/LICENSE", "RECORD"))})
    with zipfile.ZipFile(filename) as wheel:
        assert len(wheel.namelist()) == len(expected) and set(wheel.namelist()) == expected, filename
        assert wheel.getinfo(binary).external_attr >> 16 & 0o111, filename
        metadata = BytesParser().parsebytes(wheel.read(info + "METADATA"))
        assert metadata["Name"] == "agents-toolchain" and metadata["Version"] == version
        assert metadata["Requires-Python"] == ">=3.10"
        for license_file in metadata.get_all("License-File", []):
            assert info + "licenses/" + license_file in expected
        tags = BytesParser().parsebytes(wheel.read(info + "WHEEL")).get_all("Tag")
        assert set(tags) == {"py3-none-" + tag for tag in PLATFORMS[(goos, goarch)].split(".")}
        records = list(csv.reader(io.StringIO(wheel.read(info + "RECORD").decode())))
        assert len(records) == len(expected) and {row[0] for row in records} == expected
        for name, digest, size in records:
            if name == info + "RECORD":
                assert digest == size == ""
                continue
            content = wheel.read(name)
            actual = base64.urlsafe_b64encode(hashlib.sha256(content).digest()).rstrip(b"=").decode()
            assert digest == "sha256=" + actual and size == str(len(content)), name
        if goos == "linux":
            content = wheel.read(binary)
            assert content[:6] == b"\x7fELF\x02\x01", "Expected a little-endian ELF64 binary"
            assert struct.unpack_from("<H", content, 18)[0] == {"amd64": 62, "arm64": 183}[goarch]
            offset = struct.unpack_from("<Q", content, 32)[0]
            entry_size, count = struct.unpack_from("<HH", content, 54)
            types = [struct.unpack_from("<I", content, offset + i * entry_size)[0] for i in range(count)]
            assert 2 not in types and 3 not in types, "Linux wheels must contain static binaries"


def deb_members(filename):
    with open(filename, "rb") as package:
        assert package.read(8) == b"!<arch>\n"
        result = {}
        while True:
            header = package.read(60)
            if not header:
                return result
            assert len(header) == 60 and header[-2:] == b"`\n"
            size = int(header[48:58])
            result[header[:16].decode().strip().rstrip("/")] = package.read(size)
            if size % 2:
                package.read(1)


def check_deb(filename, arch, version):
    members = deb_members(filename)
    with tarfile.open(fileobj=io.BytesIO(members["control.tar.gz"]), mode="r:gz") as archive:
        control_file = next(item for item in archive.getmembers() if item.name.lstrip("./") == "control")
        control = BytesParser().parsebytes(archive.extractfile(control_file).read())
        assert control["Package"] == "agents-toolchain" and control["Version"] == version
        assert control["Architecture"] == arch
    with tarfile.open(fileobj=io.BytesIO(members["data.tar.gz"]), mode="r:gz") as archive:
        files = {item.name.lstrip("./"): item for item in archive.getmembers() if item.isfile()}
        assert set(files) == {"usr/bin/atc", "usr/share/licenses/agents-toolchain/LICENSE", "usr/share/doc/agents-toolchain/README.md"}
        assert files["usr/bin/atc"].mode & 0o111


def check_npm(version):
    targets = [(platform, arch) for platform in ("win32", "linux", "darwin") for arch in ("x64", "arm64")]
    dependencies = {"@deahesi/agents-toolchain-{}-{}".format(*target): version for target in targets}
    specs = [("@deahesi/agents-toolchain", None), *((name, target) for name, target in zip(dependencies, targets))]
    names = set()
    for name, target in specs:
        filename = name.replace("@", "").replace("/", "-") + "-" + version + ".tar.gz"
        names.add(filename)
        executable = "atc.exe" if target and target[0] == "win32" else "atc" if target else "atc.cjs"
        expected = {"LICENSE", "README.md", "package.json", "bin/" + executable}
        if target is None:
            expected.add("bin/platforms.cjs")
        with tarfile.open(DIST / filename) as archive:
            files = {item.name.removeprefix("package/"): item for item in archive.getmembers() if item.isfile()}
            assert set(files) == expected, filename
            assert files["bin/" + executable].mode & 0o111, filename
            manifest = json.load(archive.extractfile(files["package.json"]))
            assert manifest["name"] == name and manifest["version"] == version
            assert "scripts" not in manifest and "private" not in manifest
            assert manifest["publishConfig"]["access"] == "public"
            if target:
                assert manifest["os"] == [target[0]] and manifest["cpu"] == [target[1]]
                assert "optionalDependencies" not in manifest and "bin" not in manifest
            else:
                assert manifest["optionalDependencies"] == dependencies
                assert manifest["bin"] == {"atc": "bin/atc.cjs"} and manifest["engines"]["node"] == ">=22"
    assert {p.name for p in DIST.glob("deahesi-agents-toolchain-*.tar.gz")} == names
    return names


def check_release():
    version = json.loads((DIST / "metadata.json").read_text(encoding="utf-8"))["version"]
    npm_names = check_npm(version)
    wheels = list((DIST / "wheels").glob("*.whl"))
    assert len(wheels) == 6, "Expected six platform wheels"
    for (goos, goarch), platform in PLATFORMS.items():
        check_wheel(DIST / "wheels" / "agents_toolchain-{}-py3-none-{}.whl".format(version, platform), goos, goarch, version)
        binary = "atc.exe" if goos == "windows" else "atc"
        with tarfile.open(DIST / "agents-toolchain_{}_{}_{}.tar.gz".format(version, goos, goarch)) as archive:
            files = {item.name: item for item in archive.getmembers() if item.isfile()}
            assert set(files) == {"LICENSE", "README.md", binary}
            assert files[binary].mode & 0o111
    for arch in ("amd64", "arm64"):
        check_deb(DIST / "agents-toolchain_{}_{}.deb".format(version, arch), arch, version)
    for arch in ("x86_64", "aarch64"):
        assert (DIST / "agents-toolchain-{}-1.{}.rpm".format(version, arch)).is_file()
    checksum_names = set()
    for line in (DIST / "checksums.txt").read_text().splitlines():
        digest, name = line.split(None, 1)
        checksum_names.add(name)
        filename = DIST / name
        # GoReleaser uses base names for extra files in checksums.txt.
        if not filename.is_file():
            filename = DIST / "wheels" / name
        assert filename.is_file() and hashlib.sha256(filename.read_bytes()).hexdigest() == digest, name
    assert all(wheel.name in checksum_names or "wheels/" + wheel.name in checksum_names for wheel in wheels)
    expected_names = npm_names | {wheel.name for wheel in wheels}
    expected_names |= {p.name for p in DIST.glob("agents-toolchain_*.tar.gz")}
    expected_names |= {p.name for pattern in ("*.deb", "*.rpm") for p in DIST.glob(pattern)}
    assert checksum_names == expected_names, (checksum_names, expected_names)
    cask = (DIST / "homebrew/Casks/agents-toolchain.rb").read_text()
    assert 'version "{}"'.format(version) in cask and 'binary "atc"' in cask
    for goos in ("darwin", "linux"):
        for arch in ("amd64", "arm64"):
            filename = DIST / "agents-toolchain_{}_{}_{}.tar.gz".format(version, goos, arch)
            assert hashlib.sha256(filename.read_bytes()).hexdigest() in cask
    print("Seven npm archives, six wheels, six native archives, four Linux packages, cask hashes and release checksums passed.")


if __name__ == "__main__":
    check_release()
