'use strict';

const targets = ['win32', 'linux', 'darwin'].flatMap(platform =>
  ['x64', 'arm64'].map(arch => ({
    id: `${platform}-${arch}`,
    platform,
    arch,
    goos: platform === 'win32' ? 'windows' : platform,
    goarch: arch === 'x64' ? 'amd64' : arch,
    executable: platform === 'win32' ? 'atc.exe' : 'atc',
  })),
);

function getTarget(platform, arch) {
  const target = targets.find(item => item.platform === platform && item.arch === arch);
  if (!target) {
    throw new Error(`Unsupported platform: ${platform}/${arch}. Supported: ${targets.map(item => item.id).join(', ')}.`);
  }
  return target;
}

module.exports = { targets, getTarget };
