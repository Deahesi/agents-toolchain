#!/usr/bin/env node
'use strict';

const { spawn } = require('node:child_process');
const { constants } = require('node:os');
const { getTarget } = require('./platforms.cjs');
const { name: packageName, version } = require('../package.json');

function resolveBinary(platform = process.platform, arch = process.arch, resolve = require.resolve) {
  const target = getTarget(platform, arch);
  const dependency = `${packageName}-${target.id}`;
  try {
    return resolve(`${dependency}/bin/${target.executable}`);
  } catch (error) {
    if (error.code !== 'MODULE_NOT_FOUND') throw error;
    throw new Error(`Missing platform package ${dependency}@${version}. Reinstall ${packageName} with optional dependencies enabled (remove --omit=optional or optional=false).`, { cause: error });
  }
}

function runBinary(binary, args, { spawnImpl = spawn, processImpl = process } = {}) {
  const child = spawnImpl(binary, args, { stdio: 'inherit' });
  const handlers = new Map();
  for (const signal of ['SIGINT', 'SIGTERM']) {
    const handler = () => {
      // Windows sends console Ctrl+C to both processes; child.kill('SIGINT')
      // would forcibly terminate the CLI before it can restore the terminal.
      if (processImpl.platform === 'win32' && signal === 'SIGINT') return;
      child.kill(signal);
    };
    handlers.set(signal, handler);
    processImpl.on(signal, handler);
  }
  function cleanup() {
    for (const [signal, handler] of handlers) processImpl.removeListener(signal, handler);
  }
  child.once('error', error => {
    cleanup();
    processImpl.stderr.write(`atc: Could not start the CLI: ${error.message}\n`);
    processImpl.exitCode = 1;
  });
  child.once('exit', (code, signal) => {
    cleanup();
    if (signal) {
      // Preserve signal termination for shells on Unix; Windows uses a numeric status.
      processImpl.exitCode = 128 + (constants.signals[signal] || 1);
      if (processImpl.platform !== 'win32') processImpl.kill(processImpl.pid, signal);
    } else {
      processImpl.exitCode = code ?? 1;
    }
  });
  return child;
}

if (require.main === module) {
  try {
    runBinary(resolveBinary(), process.argv.slice(2));
  } catch (error) {
    process.stderr.write(`atc: ${error.message}\n`);
    process.exitCode = 1;
  }
}

module.exports = { resolveBinary, runBinary };
