import importlib.util
import json
import sys
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from build_wheel import build_wheel, PLATFORMS, ROOT
from check_artifacts import check_wheel

spec = importlib.util.spec_from_file_location("atc_launcher", ROOT / "build/python/agents_toolchain/_cli.py")
launcher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(launcher)


class PackagingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        (ROOT / ".tmp/packaging-tests").mkdir(parents=True, exist_ok=True)

    def test_wheel_targets_and_records(self):
        version = "0.1.2"
        with tempfile.TemporaryDirectory(dir=ROOT / ".tmp/packaging-tests") as directory:
            binary = Path(directory) / "atc"
            binary.write_bytes(b"fake native binary")
            for goos, goarch in PLATFORMS:
                if goos == "linux":
                    # Minimal ELF64 with the matching machine and no program headers.
                    data = bytearray(64)
                    data[:6] = b"\x7fELF\x02\x01"
                    data[18:20] = {"amd64": 62, "arm64": 183}[goarch].to_bytes(2, "little")
                    binary.write_bytes(data)
                wheel = build_wheel(version, binary, goos, goarch, directory)
                check_wheel(wheel, goos, goarch, version)

    def test_missing_binary(self):
        with patch.object(launcher, "Path") as path, patch.object(launcher.sys, "stderr"):
            path.return_value.parent.__truediv__.return_value.__truediv__.return_value.is_file.return_value = False
            self.assertEqual(launcher.main(), 1)

    def test_posix_exec_preserves_arguments(self):
        with patch.object(launcher, "Path") as path, patch.object(launcher, "os") as os_mock, patch.object(launcher.sys, "argv", ["atc", "a b", "кириллица", '"quote"', "$(literal)"]):
            binary = path.return_value.parent.__truediv__.return_value.__truediv__.return_value
            binary.__str__.return_value = "/bundled/atc"
            os_mock.name = "posix"
            os_mock.execv.side_effect = SystemExit(37)
            with self.assertRaises(SystemExit) as result:
                launcher.main()
            self.assertEqual(result.exception.code, 37)
            os_mock.execv.assert_called_once_with("/bundled/atc", ["/bundled/atc", "a b", "кириллица", '"quote"', "$(literal)"])

    def test_windows_waits_after_ctrl_c_and_returns_child_status(self):
        with patch.object(launcher, "Path"), patch.object(launcher, "os") as os_mock, patch.object(launcher.subprocess, "Popen") as popen:
            os_mock.name = "nt"
            popen.return_value.wait.side_effect = [KeyboardInterrupt, 130]
            self.assertEqual(launcher.main(), 130)
            self.assertEqual(popen.return_value.wait.call_count, 2)

    def test_start_failure(self):
        with patch.object(launcher, "Path"), patch.object(launcher, "os") as os_mock, patch.object(launcher.sys, "stderr"):
            os_mock.name = "posix"
            os_mock.execv.side_effect = OSError("permission denied")
            self.assertEqual(launcher.main(), 1)


if __name__ == "__main__":
    unittest.main()
