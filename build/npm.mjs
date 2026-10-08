// GoReleaser adapter for npm's platform packages. No build orchestration.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmdirSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { tmpdir } from 'node:os';
import { setTimeout } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

export const root = fileURLToPath(new URL('../', import.meta.url));
export const dist = path.join(root, 'dist');
export const output = path.join(root, '.tmp', 'packaging-tests');
export const metadata = JSON.parse(readFileSync(new URL('./npm/package.json', import.meta.url), 'utf8'));
export const registry = 'https://registry.npmjs.org/';
export const githubRegistry = 'https://npm.pkg.github.com/';
export const { targets, getTarget } = createRequire(import.meta.url)('./npm/platforms.cjs');

export function checkVersion(version) {
  if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)) {
    throw new Error('GoReleaser must supply a stable X.Y.Z version.');
  }
  return version;
}

export function releaseVersion(directory = dist) {
  return checkVersion(JSON.parse(readFileSync(path.join(directory, 'metadata.json'), 'utf8')).version);
}

export function run(command, args, options = {}) {
  return execFileSync(command, args, { cwd: root, stdio: 'inherit', ...options });
}

function npmCommand(args) {
  if (process.platform !== 'win32') return ['npm', args];
  // npm.cmd cannot be passed to execFile on Windows. Use npm's JS entrypoint.
  const directories = [path.dirname(process.execPath), ...(process.env.PATH ?? '').split(path.delimiter)];
  const cli = [process.env.npm_execpath, ...directories.map(dir => path.join(dir, 'node_modules/npm/bin/npm-cli.js'))]
    .find(filename => filename && existsSync(filename));
  if (!cli) throw new Error('Install Node.js with npm and add it to PATH.');
  return [process.execPath, [cli, ...args]];
}

export function npm(args, options = {}) {
  const [command, commandArgs] = npmCommand(args);
  return run(command, commandArgs, options);
}

export function removeOutputDirectory(directory) {
  const relative = path.relative(realpathSync(path.join(root, '.tmp')), realpathSync(directory));
  if (!relative || relative.startsWith('..') || path.isAbsolute(relative)) throw new Error(`Unsafe temporary output directory: ${directory}`);
  rmSync(directory, { recursive: true, force: true });
}

export function packageSpecs() {
  return [...targets.map(target => ({ name: `${metadata.name}-${target.id}`, target })), { name: metadata.name, target: null }];
}

export function preparePackages(version) {
  checkVersion(version);
  const staging = path.join(root, '.tmp', 'packaging', 'npm');
  if (existsSync(staging)) removeOutputDirectory(staging);
  const { description, license, author, repository, homepage, bugs, keywords, engines } = metadata;
  const base = { version, description, license, author, repository, homepage, bugs, keywords, publishConfig: { access: 'public', registry }, files: ['bin'] };
  function copyText(source, destination) {
    writeFileSync(destination, readFileSync(source, 'utf8').replace(/\r\n/g, '\n'), { mode: 0o644 });
  }
  for (const { name, target } of packageSpecs()) {
    const directory = path.join(staging, name.split('/')[1]);
    mkdirSync(directory, { recursive: true });
    copyText(path.join(root, 'LICENSE'), path.join(directory, 'LICENSE'));
    let manifest;
    if (target) {
      manifest = { ...base, name, os: [target.platform], cpu: [target.arch] };
      writeFileSync(path.join(directory, 'README.md'), `# ${name}\n\nNative ${target.id} binary for ${metadata.name}. Install the main package to use atc.\n`);
    } else {
      manifest = { ...base, name, engines, bin: { atc: 'bin/atc.cjs' }, optionalDependencies: Object.fromEntries(targets.map(target => [`${name}-${target.id}`, version])) };
      copyText(path.join(root, 'README.md'), path.join(directory, 'README.md'));
      mkdirSync(path.join(directory, 'bin'), { recursive: true });
      for (const file of ['atc.cjs', 'platforms.cjs']) copyText(path.join(root, 'build/npm', file), path.join(directory, 'bin', file));
      chmodSync(path.join(directory, 'bin/atc.cjs'), 0o755);
    }
    writeFileSync(path.join(directory, 'package.json'), `${JSON.stringify(manifest, null, 2)}\n`);
  }
}

export function readArtifacts({ directory = dist, version = releaseVersion(directory) } = {}) {
  checkVersion(version);
  const checksums = new Map(readFileSync(path.join(directory, 'checksums.txt'), 'utf8').trim().split(/\r?\n/).map(line => line.trim().split(/\s+/, 2).reverse()));
  return packageSpecs().map(({ name, target }) => {
    const filename = `${name.replace('@', '').replace('/', '-')}-${version}.tar.gz`;
    const archive = path.join(directory, filename);
    const content = readFileSync(archive);
    if (createHash('sha256').update(content).digest('hex') !== checksums.get(filename)) throw new Error(`GoReleaser checksum mismatch: ${filename}`);
    return { name, version, target: target?.id ?? null, filename, path: archive, integrity: `sha512-${createHash('sha512').update(content).digest('base64')}`, sha1Integrity: `sha1-${createHash('sha1').update(content).digest('base64')}` };
  });
}

export function publicationOptions(channel = 'npm', env = process.env) {
  if (channel === 'npm') return { registry };
  if (channel !== 'github') throw new Error('Unknown npm publication channel; use npm or github.');
  if (!env.GITHUB_TOKEN) throw new Error('GitHub Packages requires GITHUB_TOKEN with packages: write.');
  return { registry: githubRegistry, token: env.GITHUB_TOKEN };
}

export async function lookupIntegrity({ name, version }, request = fetch, { registry: destination = registry, token } = {}) {
  const id = `${name}@${version}`;
  // npm view reads the package-wide document, which can still return 404
  // after the exact version has been published. Query the version directly.
  // GitHub exposes versions through the authenticated package document.
  const github = destination === githubRegistry;
  if (github && !token) throw new Error('GitHub Packages requires GITHUB_TOKEN with packages: write.');
  const url = new URL(github ? encodeURIComponent(name) : `${encodeURIComponent(name)}/${encodeURIComponent(version)}`, destination);
  const headers = { accept: 'application/json' };
  if (github) headers.authorization = `Bearer ${token}`;
  let response;
  try {
    response = await request(url, { cache: 'no-store', headers, signal: AbortSignal.timeout(15000) });
  } catch (error) {
    throw new Error(`Cannot query ${id}: ${error.message}`, { cause: error });
  }
  if (!response.ok) {
    await response.body?.cancel();
    if (response.status === 404) return null;
    throw new Error(`Cannot query ${id}: registry returned HTTP ${response.status}.`);
  }
  let manifest = await response.json();
  if (github) {
    if (manifest.name !== name || !manifest.versions || typeof manifest.versions !== 'object') throw new Error(`Registry returned unexpected metadata for ${id}.`);
    if (!Object.hasOwn(manifest.versions, version)) return null;
    manifest = manifest.versions[version];
  }
  if (manifest.name !== name || manifest.version !== version) throw new Error(`Registry returned unexpected metadata for ${id}.`);
  const integrity = manifest.dist?.integrity;
  // Some npm-compatible registries expose only the standard SHA-1 shasum.
  if (github && !integrity && /^[a-f0-9]{40}$/i.test(manifest.dist?.shasum ?? '')) {
    return `sha1-${Buffer.from(manifest.dist.shasum, 'hex').toString('base64')}`;
  }
  if (typeof integrity !== 'string' || !integrity.startsWith('sha512-')) throw new Error(`Registry returned no SHA-512 integrity for ${id}.`);
  return integrity;
}

export async function publishPackages(artifacts, { lookup, publish, wait, log = console.log, attempts = 60, retryCommand = 'node build/npm.mjs publish' }) {
  const matches = (artifact, integrity) => integrity === artifact.integrity || integrity === artifact.sha1Integrity;
  const ordered = [...artifacts.filter(item => item.target), ...artifacts.filter(item => !item.target)];
  for (const artifact of ordered) {
    let found = await lookup(artifact);
    if (found) {
      if (!matches(artifact, found)) throw new Error(`${artifact.name}@${artifact.version} is already published with different contents.`);
      log(`Already published: ${artifact.name}@${artifact.version}`);
      continue;
    }
    await publish(artifact);
    for (let attempt = 0; attempt < attempts; attempt++) {
      found = await lookup(artifact);
      if (found) break;
      if (attempt === 0) log(`Waiting for registry visibility: ${artifact.name}@${artifact.version}`);
      if (attempt + 1 < attempts) await wait();
    }
    if (!found) throw new Error(`Published package ${artifact.name}@${artifact.version} is not available in the registry yet after ${attempts} checks. Keep dist/ unchanged and rerun "${retryCommand}".`);
    if (!matches(artifact, found)) throw new Error(`${artifact.name}@${artifact.version} is already published with different contents.`);
    log(`Published: ${artifact.name}@${artifact.version}`);
  }
}

export async function publishNpm(version = releaseVersion(), mainArchive = path.join(dist, `deahesi-agents-toolchain-${version}.tar.gz`), channel = 'npm', {
  request = fetch, npmImpl = npm, env = process.env, wait = () => setTimeout(5000), log = console.log,
} = {}) {
  checkVersion(version);
  const expected = `deahesi-agents-toolchain-${version}.tar.gz`;
  if (path.basename(mainArchive) !== expected) throw new Error(`Expected npm launcher archive ${expected}, received ${mainArchive}.`);
  const destination = publicationOptions(channel, env);
  const directory = path.dirname(path.resolve(mainArchive));
  const artifacts = readArtifacts({ directory, version });
  let configDirectory;
  let configFile;
  let npmEnv = env;
  try {
    if (channel === 'github') {
      configDirectory = mkdtempSync(path.join(tmpdir(), 'atc-github-npm-'));
      configFile = path.join(configDirectory, 'npmrc');
      // Store a variable reference, never the credential itself. Keep scope
      // routing and authentication isolated from the user's npm configuration.
      writeFileSync(configFile, `${metadata.name.split('/')[0]}:registry=${githubRegistry}\n//npm.pkg.github.com/:_authToken=\${GITHUB_TOKEN}\n`, { mode: 0o600 });
      npmEnv = { ...env, NPM_CONFIG_USERCONFIG: configFile };
    }
    log(`Publishing npm packages to ${destination.registry}`);
    await publishPackages(artifacts, {
      lookup: artifact => lookupIntegrity(artifact, request, destination),
      publish: artifact => npmImpl(['publish', artifact.path, '--access', 'public', '--tag', 'latest', '--registry', destination.registry, '--loglevel=verbose'], { env: npmEnv }),
      wait, log,
      retryCommand: `node build/npm.mjs publish ${version} "${mainArchive}" ${channel}`,
    });
  } finally {
    if (configFile) rmSync(configFile, { force: true });
    if (configDirectory) rmdirSync(configDirectory);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [command, ...args] = process.argv.slice(2);
  if (command === 'prepare' && args.length === 1) preparePackages(args[0]);
  else if (command === 'publish' && [0, 2, 3].includes(args.length)) await publishNpm(...args);
  else throw new Error('Usage: node build/npm.mjs prepare VERSION | publish [VERSION MAIN_ARCHIVE [npm|github]]');
}
