"""Install a host wheel offline into an isolated venv and exercise the command."""

import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
import tempfile
import venv

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from build_wheel import DIST, PLATFORMS, ROOT


def smoke():
    if sys.version_info < (3, 10):
        raise SystemExit("Wheel installation checks require Python 3.10 or later.")
    version = json.loads((DIST / "metadata.json").read_text(encoding="utf-8"))["version"]
    goos = {"win32": "windows", "darwin": "darwin", "linux": "linux"}[sys.platform]
    arch = {"amd64": "amd64", "x86_64": "amd64", "arm64": "arm64", "aarch64": "arm64"}[platform.machine().lower()]
    wheel = DIST / "wheels" / "agents_toolchain-{}-py3-none-{}.whl".format(version, PLATFORMS[(goos, arch)])
    (ROOT / ".tmp/packaging-tests").mkdir(parents=True, exist_ok=True)
    work = Path(tempfile.mkdtemp(prefix="pip smoke ", dir=ROOT / ".tmp/packaging-tests"))
    environment = work / "venv with spaces"
    venv.create(environment, with_pip=True)
    scripts = environment / ("Scripts" if os.name == "nt" else "bin")
    python = scripts / ("python.exe" if os.name == "nt" else "python")
    subprocess.run([str(python), "-m", "pip", "install", "--no-index", "--no-deps", "--disable-pip-version-check", str(wheel)], check=True)
    env = {**os.environ, "PATH": str(scripts) + os.pathsep + os.environ.get("PATH", "")}
    executable = shutil.which("atc", path=env["PATH"])
    assert executable and Path(executable).parent == scripts
    project = work / "temporary project"
    project.mkdir()

    def run(args, code=0):
        result = subprocess.run([executable, *args], cwd=project, env=env, capture_output=True, text=True)
        assert result.returncode == code, (args, result.returncode, result.stdout, result.stderr)
        return result.stdout + result.stderr

    assert version in run(["--version"])
    assert "Usage:" in run(["--help"])
    assert "unknown flag" in run(["--no-such-flag"], 1)
    run(["init", "--agents-dir", "agent configs"])
    assert (project / "agents-toolchain.yml").is_file() and (project / "agent configs").is_dir()
    result = subprocess.run([str(python), "-m", "agents_toolchain", "--version"], cwd=project, capture_output=True, text=True)
    assert result.returncode == 0 and version in result.stdout
    print("Offline pip installation, atc, python -m, help, version, errors and init passed.")
    print("Smoke test files:", work)


if __name__ == "__main__":
    smoke()
