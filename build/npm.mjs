// GoReleaser adapter for npm's platform packages. No build orchestration.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, existsSync, mkdirSync, readFileSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { setTimeout } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

export const root = fileURLToPath(new URL('../', import.meta.url));
export const dist = path.join(root, 'dist');
export const output = path.join(root, '.tmp', 'packaging-tests');
export const metadata = JSON.parse(readFileSync(new URL('./npm/package.json', import.meta.url), 'utf8'));
export const registry = 'https://registry.npmjs.org/';
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

export function npmResult(args, options = {}) {
  const [command, commandArgs] = npmCommand(args);
  return spawnSync(command, commandArgs, { cwd: root, encoding: 'utf8', ...options });
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
    return { name, version, target: target?.id ?? null, filename, path: archive, integrity: `sha512-${createHash('sha512').update(content).digest('base64')}` };
  });
}

export function parseRegistryResult(result, name) {
  if (result.error) throw result.error;
  if (result.status === 0) {
    const parsed = JSON.parse(result.stdout);
    // npm 12 wraps even a single queried field in an array; npm <=11 unwraps it.
    const value = Array.isArray(parsed) && parsed.length === 1 ? parsed[0] : parsed;
    if (typeof value !== 'string' || !value.startsWith('sha512-')) throw new Error(`Registry returned no SHA-512 integrity for ${name}.`);
    return value;
  }
  let code;
  try { code = JSON.parse(result.stdout).error?.code; } catch { /* npm may report errors only on stderr. */ }
  if (code === 'E404' || /\bnpm (?:error|ERR!) code E404\b/.test(result.stderr || '')) return null;
  throw new Error(`Cannot query ${name}: ${result.stderr || result.stdout || `exit ${result.status}`}`);
}

export async function publishPackages(artifacts, { lookup, publish, wait, log = console.log, attempts = 12 }) {
  const ordered = [...artifacts.filter(item => item.target), ...artifacts.filter(item => !item.target)];
  for (const artifact of ordered) {
    let found = await lookup(artifact);
    if (found) {
      if (found !== artifact.integrity) throw new Error(`${artifact.name}@${artifact.version} is already published with different contents.`);
      log(`Already published: ${artifact.name}@${artifact.version}`);
      continue;
    }
    await publish(artifact);
    for (let attempt = 0; attempt < attempts; attempt++) {
      found = await lookup(artifact);
      if (found) break;
      if (attempt + 1 < attempts) await wait();
    }
    if (found !== artifact.integrity) throw new Error(`Published package ${artifact.name}@${artifact.version} is not available with the expected integrity; rerun after checking the registry.`);
    log(`Published: ${artifact.name}@${artifact.version}`);
  }
}

export async function publishNpm(version = releaseVersion(), mainArchive = path.join(dist, `deahesi-agents-toolchain-${version}.tar.gz`)) {
  const directory = path.dirname(path.resolve(mainArchive));
  const artifacts = readArtifacts({ directory, version });
  await publishPackages(artifacts, {
    lookup: artifact => parseRegistryResult(npmResult(['view', `${artifact.name}@${artifact.version}`, 'dist.integrity', '--json', '--registry', registry, '--fetch-retries=0', '--fetch-timeout=15000']), artifact.name),
    publish: artifact => npm(['publish', artifact.path, '--access', 'public', '--tag', 'latest', '--registry', registry]),
    wait: () => setTimeout(5000),
  });
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [command, ...args] = process.argv.slice(2);
  if (command === 'prepare' && args.length === 1) preparePackages(args[0]);
  else if (command === 'publish' && [0, 2].includes(args.length)) await publishNpm(...args);
  else throw new Error('Usage: node build/npm.mjs prepare VERSION | publish [VERSION MAIN_ARCHIVE]');
}
