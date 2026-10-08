import assert from 'node:assert/strict';
import test from 'node:test';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { checkVersion, parseRegistryResult, publishPackages, packageSpecs, readArtifacts, output, removeOutputDirectory } from '../npm.mjs';

const main = { name: 'main', version: '0.1.2', target: null, integrity: 'sha512-main' };
const native = { name: 'native', version: '0.1.2', target: 'linux-x64', integrity: 'sha512-native' };

test('accepts GoReleaser release versions and rejects invalid npm/PyPI versions', () => {
  assert.equal(checkVersion('0.1.2'), '0.1.2');
  for (const version of ['v0.1.2', '0.1.2-dev', '01.2.3', 'invalid']) assert.throws(() => checkVersion(version), /stable X.Y.Z/);
});

test('distinguishes missing versions from registry and authentication failures', () => {
  assert.equal(parseRegistryResult({ status: 0, stdout: '"sha512-native"' }, 'native'), 'sha512-native');
  assert.equal(parseRegistryResult({ status: 0, stdout: '["sha512-native"]' }, 'native'), 'sha512-native');
  assert.equal(parseRegistryResult({ status: 1, stdout: '{"error":{"code":"E404"}}' }, 'native'), null);
  assert.equal(parseRegistryResult({ status: 1, stderr: 'npm error code E404\nNot found' }, 'native'), null);
  for (const code of ['E401', 'E403', 'EACCES', 'ETIMEDOUT']) {
    assert.throws(() => parseRegistryResult({ status: 1, stdout: JSON.stringify({ error: { code } }) }, 'native'), /Cannot query/);
  }
});

test('publishes platform packages before the main package and waits for availability', async () => {
  const published = [];
  const available = new Map();
  let waits = 0;
  await publishPackages([main, native], {
    lookup: artifact => available.get(artifact.name) ?? null,
    publish: artifact => {
      if (artifact === main) assert.equal(available.get(native.name), native.integrity);
      published.push(artifact.name);
      if (artifact === main) available.set(main.name, main.integrity);
    },
    wait: () => { waits++; available.set(native.name, native.integrity); },
    log: () => {},
  });
  assert.deepEqual(published, ['native', 'main']);
  assert.equal(waits, 1);
});

test('resumes a partially published release without publishing existing versions again', async () => {
  const available = new Map([[native.name, native.integrity]]);
  const published = [];
  const options = {
    lookup: artifact => available.get(artifact.name) ?? null,
    publish: artifact => { published.push(artifact.name); available.set(artifact.name, artifact.integrity); },
    wait: () => {}, log: () => {},
  };
  await publishPackages([main, native], options);
  await publishPackages([main, native], options);
  assert.deepEqual(published, ['main']);
});

test('does not publish the main package when a platform package stays unavailable', async () => {
  const published = [];
  await assert.rejects(publishPackages([main, native], {
    lookup: () => null, publish: artifact => published.push(artifact.name), wait: () => {}, log: () => {}, attempts: 2,
  }), /not available/);
  assert.deepEqual(published, ['native']);
});

test('refuses to skip an existing version with different contents', async () => {
  await assert.rejects(publishPackages([main, native], {
    lookup: () => 'sha512-different', publish: () => assert.fail('must not publish'), wait: () => {}, log: () => {},
  }), /different contents/);
});

test('reads all seven GoReleaser archives and rejects modified or stale files before publication', t => {
  mkdirSync(output, { recursive: true });
  const directory = mkdtempSync(path.join(output, 'release checksums '));
  t.after(() => removeOutputDirectory(directory));
  const version = '0.1.2';
  writeFileSync(path.join(directory, 'metadata.json'), JSON.stringify({ version }));
  const checksums = packageSpecs().map(({ name }) => {
    const filename = `${name.replace('@', '').replace('/', '-')}-${version}.tar.gz`;
    const content = Buffer.from(name);
    writeFileSync(path.join(directory, filename), content);
    return `${createHash('sha256').update(content).digest('hex')}  ${filename}`;
  });
  writeFileSync(path.join(directory, 'checksums.txt'), checksums.join('\n'));
  const artifacts = readArtifacts({ directory });
  assert.equal(artifacts.length, 7);
  assert.equal(artifacts.filter(item => item.target).length, 6);
  // Custom publishers run before artifacts.json is written. No second index is needed.
  assert.equal(readArtifacts({ directory, version }).length, 7);
  writeFileSync(artifacts[0].path, 'modified');
  assert.throws(() => readArtifacts({ directory }), /checksum mismatch/);
  assert.throws(() => readArtifacts({ directory, version: '0.1.3' }), /ENOENT/);
});
