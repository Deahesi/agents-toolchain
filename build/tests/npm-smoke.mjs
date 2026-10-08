import assert from 'node:assert/strict';
import { existsSync, mkdirSync, mkdtempSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { getTarget, metadata, npm, output, run, readArtifacts, releaseVersion } from '../npm.mjs';

const version = releaseVersion();

const artifacts = readArtifacts();
const target = getTarget(process.platform, process.arch);
const main = artifacts.find(item => item.name === metadata.name);
const native = artifacts.find(item => item.target === target.id);
assert.ok(main && native, `Build the main package and ${target.id} before running the smoke test.`);
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'smoke test '));
const prefix = path.join(directory, 'global prefix');
const project = path.join(directory, 'project with spaces');
const local = path.join(directory, 'local install');
for (const dir of [prefix, project, local]) mkdirSync(dir, { recursive: true });
// An empty, isolated cache makes this work before the first npm publication.
const env = { ...process.env, npm_config_cache: path.join(directory, 'cache') };
const tarballs = [main.path, native.path];

npm(['install', '--global', '--prefix', prefix, '--offline', '--ignore-scripts', '--no-audit', '--no-fund', ...tarballs], { env });
const shim = process.platform === 'win32' ? path.join(prefix, 'atc.cmd') : path.join(prefix, 'bin', 'atc');
assert.ok(existsSync(shim), 'npm did not create the global atc command.');
function atc(args) {
  const options = { cwd: project, env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] };
  if (process.platform === 'win32') {
    // Pass values through environment variables, never interpolate shell code.
    return run('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', '$taskArguments = ConvertFrom-Json $env:ATC_SMOKE_ARGS; & $env:ATC_SMOKE_SHIM @taskArguments; exit $LASTEXITCODE'], {
      ...options, env: { ...env, ATC_SMOKE_SHIM: shim, ATC_SMOKE_ARGS: JSON.stringify(args) },
    });
  }
  return run(shim, args, options);
}
assert.match(atc(['--help']), /Manage and run YAML-configured agents/);
assert.equal(atc(['--version']).trim(), `atc version ${version}`);
let rejected = false;
try { atc(['--not-an-atc-flag']); } catch (error) {
  rejected = true;
  assert.equal(error.status, 1);
  assert.match(error.stderr.toString(), /unknown flag/);
}
assert.ok(rejected, 'An invalid flag must return a nonzero exit code.');
atc(['init', '--agents-dir', 'agent configs']);
assert.match(readFileSync(path.join(project, 'agents-toolchain.yml'), 'utf8'), /agents_dir: agent configs/);
assert.ok(existsSync(path.join(project, 'agent configs')));

npm(['install', '--prefix', local, '--offline', '--no-package-lock', '--no-save', '--no-audit', '--no-fund', ...tarballs], { env });
const result = npm(['exec', '--offline', '--package', metadata.name, '--', 'atc', '--version'], { cwd: local, env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] });
assert.equal(result.trim(), `atc version ${version}`);
// This is npx's default package-to-bin inference, without an explicit atc command.
const inferred = npm(['exec', '--offline', '--', metadata.name, '--help'], { cwd: local, env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] });
assert.match(inferred, /Manage and run YAML-configured agents/);
console.log(`Global install (--ignore-scripts), npm exec/npx, help, version, errors and init passed on ${target.id}.`);
console.log(`Smoke test files: ${directory}`);
