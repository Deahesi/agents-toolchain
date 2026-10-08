"""Exercise the generated cask against local archives before a release exists."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from build_wheel import DIST, ROOT

if sys.platform not in ("darwin", "linux"):
    raise SystemExit("Homebrew cask installation checks run on macOS or Linux.")
env = {**os.environ, "HOMEBREW_NO_AUTO_UPDATE": "1", "HOMEBREW_NO_INSTALL_FROM_API": "1"}
tap = "deahesi/atc-packaging-test"
subprocess.run(["brew", "tap-new", tap], env=env, check=True)
repository = Path(subprocess.check_output(["brew", "--repository", tap], env=env, text=True).strip())
cask = (DIST / "homebrew/Casks/agents-toolchain.rb").read_text()
cask = re.sub(r'url "https://[^"\n]+/([^/"\n]+)"', lambda match: 'url "' + DIST.as_uri() + '/' + match[1] + '"', cask)
(repository / "Casks").mkdir(exist_ok=True)
(repository / "Casks/agents-toolchain.rb").write_text(cask)
subprocess.run(["brew", "install", "--cask", tap + "/agents-toolchain"], env=env, check=True)
version = json.loads((DIST / "metadata.json").read_text(encoding="utf-8"))["version"]
(ROOT / ".tmp/packaging-tests").mkdir(parents=True, exist_ok=True)
work = Path(tempfile.mkdtemp(prefix="brew smoke ", dir=ROOT / ".tmp/packaging-tests"))
for args, code in ((["--version"], 0), (["--help"], 0), (["--no-such-flag"], 1), (["init"], 0)):
    result = subprocess.run(["atc", *args], cwd=work, env=env, text=True, capture_output=True)
    assert result.returncode == code, result.stderr
    if args == ["--version"]:
        assert version in result.stdout
assert (work / "agents-toolchain.yml").is_file()
print("Homebrew cask installation, help, version, errors and init passed.")
