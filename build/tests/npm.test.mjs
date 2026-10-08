import assert from 'node:assert/strict';
import test from 'node:test';
import { createHash } from 'node:crypto';
import { execFileSync, spawn, spawnSync } from 'node:child_process';
import { createServer } from 'node:http';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { checkVersion, lookupIntegrity, registry, publishPackages, publishNpm, packageSpecs, readArtifacts, root, output, removeOutputDirectory } from '../npm.mjs';

const main = { name: 'main', version: '0.1.2', target: null, integrity: 'sha512-main' };
const native = { name: 'native', version: '0.1.2', target: 'linux-x64', integrity: 'sha512-native' };

test('accepts GoReleaser release versions and rejects invalid npm/PyPI versions', () => {
  assert.equal(checkVersion('0.1.2'), '0.1.2');
  for (const version of ['v0.1.2', '0.1.2-dev', '01.2.3', 'invalid']) assert.throws(() => checkVersion(version), /stable X.Y.Z/);
});

test('recognizes an existing scoped version when package-wide metadata still returns 404', async () => {
  const artifact = { ...native, name: '@deahesi/agents-toolchain-win32-arm64' };
  const versionURL = `${registry}${encodeURIComponent(artifact.name)}/${artifact.version}`;
  const requests = [];
  const request = async (url, options) => {
    requests.push(String(url));
    assert.equal(options.cache, 'no-store');
    assert.equal(options.headers.accept, 'application/json');
    assert.ok(options.signal instanceof AbortSignal);
    if (String(url) !== versionURL) return new Response('{"error":"Not found"}', { status: 404 });
    return Response.json({ name: artifact.name, version: artifact.version, dist: { integrity: artifact.integrity } });
  };
  await publishPackages([artifact], {
    lookup: item => lookupIntegrity(item, request),
    publish: () => assert.fail('must not republish an existing version'),
    wait: () => assert.fail('version is already available'), log: () => {},
  });
  assert.deepEqual(requests, [versionURL]);
});

test('distinguishes a missing version from registry and authentication failures', async () => {
  assert.equal(await lookupIntegrity(native, async () => new Response('Not found', { status: 404 })), null);
  for (const status of [401, 403, 429, 500, 503]) {
    await assert.rejects(lookupIntegrity(native, async () => new Response('Registry error', { status })), new RegExp(`Cannot query native@0.1.2: registry returned HTTP ${status}`));
  }
  await assert.rejects(lookupIntegrity(native, async () => { throw new Error('network timeout'); }), /Cannot query native@0.1.2: network timeout/);
});

test('rejects unexpected or invalid version metadata instead of treating it as unpublished', async () => {
  for (const manifest of [{}, { name: native.name, version: '0.1.3' }]) {
    await assert.rejects(lookupIntegrity(native, async () => Response.json(manifest)), /unexpected metadata/);
  }
  for (const dist of [undefined, {}, { integrity: 'sha1-invalid' }]) {
    await assert.rejects(lookupIntegrity(native, async () => Response.json({ name: native.name, version: native.version, dist })), /no SHA-512 integrity/);
  }
  await assert.rejects(lookupIntegrity(native, async () => new Response('not json')), SyntaxError);
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

test('reports a content conflict after publishing separately from a visibility timeout', async () => {
  const published = [];
  let queries = 0;
  await assert.rejects(publishPackages([main, native], {
    lookup: () => queries++ === 0 ? null : 'sha512-different',
    publish: artifact => published.push(artifact.name),
    wait: () => assert.fail('must not retry different contents'), log: () => {},
  }), /already published with different contents/);
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


test('rejects wheel paths at the npm entrypoint before reading checksums or publishing', async () => {
  await assert.rejects(publishNpm('0.1.2', path.join('dist', 'wheels', 'agents_toolchain-0.1.2-py3-none-win_amd64.whl')), /Expected npm launcher archive/);
  await assert.rejects(publishNpm('0.1.2', path.join('dist', 'deahesi-agents-toolchain-0.1.3.tar.gz')), /Expected npm launcher archive/);
});

test('GoReleaser publishes npm and PyPI once when GitHub release extra_files include all six wheels', { timeout: 90000 }, async t => {
  const localTool = path.join(root, '.tmp/tools/goreleaser', process.platform === 'win32' ? 'goreleaser.exe' : 'goreleaser');
  const goreleaser = existsSync(localTool) ? localTool : 'goreleaser';
  if (spawnSync(goreleaser, ['--version'], { stdio: 'ignore' }).status !== 0) {
    t.skip('GoReleaser is not installed');
    return;
  }
  mkdirSync(output, { recursive: true });
  const directory = mkdtempSync(path.join(output, 'publisher integration '));
  t.after(() => removeOutputDirectory(directory));
  const uploads = [];
  let release;
  const server = createServer(async (request, response) => {
    const url = new URL(request.url, 'http://localhost');
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    const reply = (status, body) => {
      response.writeHead(status, { 'Content-Type': 'application/json' });
      response.end(JSON.stringify(body));
    };
    if (request.method === 'POST' && url.pathname.endsWith('/assets')) {
      const name = url.searchParams.get('name');
      uploads.push(name);
      reply(201, { id: uploads.length, name, state: 'uploaded', size: Buffer.concat(chunks).length });
    } else if (request.method === 'POST' && url.pathname.endsWith('/releases')) {
      release = { id: 1, tag_name: 'v0.1.2', name: 'v0.1.2', draft: true, prerelease: false, html_url: 'https://example.invalid/release', upload_url: 'http://127.0.0.1:' + server.address().port + '/uploads/repos/fixture/agents-toolchain/releases/1/assets{?name,label}', assets: [] };
      reply(201, release);
    } else if (request.method === 'PATCH' && url.pathname.endsWith('/releases/1')) {
      Object.assign(release, JSON.parse(Buffer.concat(chunks).toString()));
      reply(200, release);
    } else if (request.method === 'GET' && url.pathname.endsWith('/assets')) {
      reply(200, []);
    } else if (request.method === 'GET' && url.pathname.endsWith('/releases')) {
      reply(200, release ? [release] : []);
    } else if (request.method === 'GET' && /\/releases\/(?:1|tags\/v0\.1\.2)$/.test(url.pathname) && release) {
      reply(200, release);
    } else {
      reply(404, { message: 'Not Found' });
    }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  t.after(() => {
    server.closeAllConnections();
    return new Promise(resolve => server.close(resolve));
  });
  const endpoint = 'http://127.0.0.1:' + server.address().port;
  const production = readFileSync(path.join(root, 'build/goreleaser.yaml'), 'utf8').replace(/\r\n/g, '\n');
  const publisherStart = production.indexOf('\npublishers:\n');
  assert.notEqual(publisherStart, -1);
  // Keep the production selectors and templates. Replace only the upload
  // implementation with local recorders; no npm or PyPI request is possible.
  const publishers = production.slice(publisherStart).replace('uv publish ', 'node build/pypi-test.mjs publish ');
  const config = 'version: 2\nproject_name: agents-toolchain\nbuilds:\n  - skip: true\narchives:\n  - id: npm-launcher\n    meta: true\n    formats: [tar.gz]\n    name_template: "deahesi-agents-toolchain-{{ .Version }}"\n    files: [README.md]\nchangelog:\n  disable: true\ngithub_urls:\n  api: ' + endpoint + '/api/v3/\n  upload: ' + endpoint + '/uploads/\n  download: ' + endpoint + '/\nrelease:\n  github:\n    owner: fixture\n    name: agents-toolchain\n  extra_files:\n    - glob: wheels/*.whl\n' + publishers;
  writeFileSync(path.join(directory, 'goreleaser.yaml'), config);
  writeFileSync(path.join(directory, 'README.md'), 'Publisher integration fixture\n');
  writeFileSync(path.join(directory, 'go.mod'), 'module example.invalid/publisher-test\n\ngo 1.26.5\n');
  writeFileSync(path.join(directory, '.gitignore'), 'dist/\n.tmp/\n');
  mkdirSync(path.join(directory, 'build'));
  mkdirSync(path.join(directory, '.tmp'));
  mkdirSync(path.join(directory, 'wheels'));
  const record = publisher => "import { appendFileSync } from 'node:fs';\nappendFileSync('.tmp/calls.jsonl', JSON.stringify({ publisher: '" + publisher + "', args: process.argv.slice(2), oidc: { githubActions: process.env.GITHUB_ACTIONS, requestUrl: process.env.ACTIONS_ID_TOKEN_REQUEST_URL, requestToken: process.env.ACTIONS_ID_TOKEN_REQUEST_TOKEN } }) + '\\n');\n";
  writeFileSync(path.join(directory, 'build/npm.mjs'), record('npm'));
  writeFileSync(path.join(directory, 'build/pypi-test.mjs'), record('pypi'));
  const wheels = ['win_amd64', 'win_arm64', 'macosx_12_0_x86_64', 'macosx_12_0_arm64', 'manylinux_2_17_x86_64', 'manylinux_2_17_aarch64'].map(platform => 'agents_toolchain-0.1.2-py3-none-' + platform + '.whl');
  for (const wheel of wheels) writeFileSync(path.join(directory, 'wheels', wheel), 'fixture wheel\n');
  const env = { ...process.env, GITHUB_ACTIONS: 'true', ACTIONS_ID_TOKEN_REQUEST_URL: endpoint + '/oidc?fixture=1&run=2', ACTIONS_ID_TOKEN_REQUEST_TOKEN: 'fake-oidc-fixture-token', GITHUB_TOKEN: 'local-integration-fixture', GITHUB_REPOSITORY: 'fixture/agents-toolchain', GITHUB_REF: 'refs/tags/v0.1.2', GITHUB_EVENT_NAME: 'push', GIT_CONFIG_COUNT: '1', GIT_CONFIG_KEY_0: 'safe.directory', GIT_CONFIG_VALUE_0: directory };
  const git = args => execFileSync('git', args, { cwd: directory, env, stdio: 'pipe', encoding: 'utf8' });
  git(['init', '-b', 'main']);
  git(['add', '.']);
  git(['-c', 'user.name=Publisher test', '-c', 'user.email=test@example.invalid', 'commit', '-m', 'Publisher integration fixture']);
  git(['remote', 'add', 'origin', 'https://github.com/fixture/agents-toolchain.git']);
  git(['tag', 'v0.1.2']);
  env.GITHUB_SHA = git(['rev-parse', 'HEAD']).trim();
  let processOutput = '';
  const code = await new Promise((resolve, reject) => {
    const child = spawn(goreleaser, ['release', '--config', 'goreleaser.yaml', '--clean'], { cwd: directory, env, timeout: 60000, stdio: ['ignore', 'pipe', 'pipe'] });
    child.stdout.on('data', data => { processOutput += data; });
    child.stderr.on('data', data => { processOutput += data; });
    child.on('error', reject);
    child.on('close', resolve);
  });
  assert.equal(code, 0, processOutput.slice(-12000));
  for (const wheel of wheels) assert.ok(uploads.includes(wheel), 'Wheel must pass through the real release.extra_files pipeline: ' + wheel);
  const calls = readFileSync(path.join(directory, '.tmp/calls.jsonl'), 'utf8').trim().split('\n').map(line => JSON.parse(line));
  assert.deepEqual(calls.map(call => call.publisher), ['npm', 'pypi']);
  for (const call of calls) assert.deepEqual(call.oidc, { githubActions: 'true', requestUrl: endpoint + '/oidc?fixture=1&run=2', requestToken: 'fake-oidc-fixture-token' }, call.publisher + ' must receive the GitHub OIDC environment unchanged');
  assert.equal(calls[0].args[0], 'publish');
  assert.equal(calls[0].args[1], '0.1.2');
  assert.equal(path.basename(calls[0].args[2]), 'deahesi-agents-toolchain-0.1.2.tar.gz');
  assert.deepEqual(calls[1].args, ['publish', '--trusted-publishing', 'always', '--check-url', 'https://pypi.org/simple/', 'dist/wheels/*.whl']);
});
