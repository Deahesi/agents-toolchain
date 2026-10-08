import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { EventEmitter, once } from 'node:events';
import { copyFileSync, mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import test, { after } from 'node:test';
import { metadata, output, removeOutputDirectory, root } from '../npm.mjs';

const require = createRequire(import.meta.url);
// Load the launcher in the same layout as the installed npm package.
mkdirSync(output, { recursive: true });
const fixturePackage = mkdtempSync(path.join(output, 'launcher package '));
const fixtureBin = path.join(fixturePackage, 'bin');
mkdirSync(fixtureBin);
writeFileSync(path.join(fixturePackage, 'package.json'), JSON.stringify({ ...metadata, version: '0.1.2' }));
for (const file of ['atc.cjs', 'platforms.cjs']) copyFileSync(path.join(root, 'build', 'npm', file), path.join(fixtureBin, file));
after(() => removeOutputDirectory(fixturePackage));
const launcher = path.join(fixtureBin, 'atc.cjs');
const { resolveBinary, runBinary } = require(launcher);
const runner = 'require(process.argv[1]).runBinary(process.execPath, process.argv.slice(2))';

test('resolves the native binary for all supported Node platforms', () => {
  for (const platform of ['win32', 'linux', 'darwin']) {
    for (const arch of ['x64', 'arm64']) {
      const expected = `${metadata.name}-${platform}-${arch}/bin/atc${platform === 'win32' ? '.exe' : ''}`;
      assert.equal(resolveBinary(platform, arch, value => value), expected);
    }
  }
  assert.throws(() => resolveBinary('freebsd', 'x64'), /Unsupported platform: freebsd\/x64/);
  assert.throws(() => resolveBinary('linux', 'ia32'), /Unsupported platform: linux\/ia32/);
});

test('reports missing optional packages with reinstall instructions', () => {
  assert.throws(() => resolveBinary('linux', 'arm64', () => {
    throw Object.assign(new Error('not found'), { code: 'MODULE_NOT_FOUND' });
  }), /Missing platform package .*linux-arm64.*optional dependencies.*--omit=optional/);
});

test('preserves arguments, working directory, environment and stdin/stdout/stderr', t => {
  mkdirSync(output, { recursive: true });
  const cwd = mkdtempSync(path.join(output, 'launcher test '));
  t.after(() => removeOutputDirectory(cwd));
  const args = ['space in argument', 'текст 🙂', '"quotes"', '$(literal)&value', '--flag=value'];
  const fixture = 'process.stdout.write(JSON.stringify({args:process.argv.slice(1),cwd:process.cwd(),env:process.env.ATC_LAUNCHER_TEST,stdin:require("node:fs").readFileSync(0,"utf8")}));process.stderr.write("fixture stderr");';
  const result = spawnSync(process.execPath, ['--eval', runner, '--', launcher, '--eval', fixture, '--', ...args], {
    cwd, env: { ...process.env, ATC_LAUNCHER_TEST: 'inherited value' }, input: 'fixture stdin\n', encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout), { args, cwd, env: 'inherited value', stdin: 'fixture stdin\n' });
  assert.equal(result.stderr, 'fixture stderr');
});

test('preserves a nonzero child exit code', () => {
  const result = spawnSync(process.execPath, ['--eval', runner, '--', launcher, '--eval', 'process.exit(37)'], { encoding: 'utf8' });
  assert.equal(result.status, 37, result.stderr);
});

test('reports an unlaunchable executable and exits with failure', () => {
  const code = 'require(process.argv[1]).runBinary(process.argv[2], [])';
  const result = spawnSync(process.execPath, ['--eval', code, '--', launcher, path.join(output, 'does-not-exist', 'atc')], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /Could not start the CLI/);
});

test('lets the Windows CLI handle console Ctrl+C and releases its signal handlers', () => {
  const host = Object.assign(new EventEmitter(), { platform: 'win32' });
  const child = new EventEmitter();
  const signals = [];
  child.kill = signal => signals.push(signal);
  runBinary('atc.exe', [], { spawnImpl: () => child, processImpl: host });
  host.emit('SIGINT');
  assert.deepEqual(signals, []);
  host.emit('SIGTERM');
  assert.deepEqual(signals, ['SIGTERM']);
  child.emit('exit', null, 'SIGINT');
  assert.equal(host.exitCode, 130);
  assert.equal(host.listenerCount('SIGINT'), 0);
  assert.equal(host.listenerCount('SIGTERM'), 0);
});

test('forwards and preserves Unix signal termination', { skip: process.platform === 'win32', timeout: 10000 }, async t => {
  const wrapper = spawn(process.execPath, ['--eval', runner, '--', launcher, '--eval', 'process.stdout.write("ready");setInterval(()=>{},1000)'], { stdio: ['ignore', 'pipe', 'pipe'] });
  t.after(() => { if (wrapper.exitCode === null && wrapper.signalCode === null) wrapper.kill('SIGKILL'); });
  const exited = once(wrapper, 'exit');
  await once(wrapper.stdout, 'data');
  wrapper.kill('SIGTERM');
  assert.deepEqual(await exited, [null, 'SIGTERM']);
});
