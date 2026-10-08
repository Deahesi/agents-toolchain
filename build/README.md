# Build and release

GoReleaser owns compilation, packaging and publication. Its configuration is
[goreleaser.yaml](goreleaser.yaml); CI is defined in
[release.yml](../.github/workflows/release.yml). The version comes from a Git tag
such as `v0.1.4`, not from an npm manifest.

Build locally without publishing (Go, Node.js, Python and GoReleaser required):

```sh
goreleaser check --config build/goreleaser.yaml
goreleaser release --config build/goreleaser.yaml --snapshot --clean
```

Run these commands from the repository root. Packages go to `dist/`; npm staging
files go to `.tmp/packaging/npm/`. A pushed `vX.Y.Z` tag runs the checks and then
publishes through GoReleaser. A rerun of an older tag uses that tag's configuration.

## npm registries

The `npm` and `github-packages` GoReleaser publishers use `npm.mjs` to publish the
same seven checksummed archives to npmjs.com and GitHub Packages, respectively.
Each registry is checked independently. The six platform packages must become
available before the launcher is published. Existing versions are skipped only
when the registry checksum matches; different contents stop publication. npmjs
uses SHA-512 integrity. For GitHub metadata without integrity, the adapter compares
the standard SHA-1 shasum instead. Publication is not atomic across registries.

| Registry | CI authentication | One-time setup |
| --- | --- | --- |
| npmjs.com | OIDC | Configure GitHub Actions Trusted Publisher for all seven packages: owner `Deahesi`, repository `agents-toolchain`, workflow `release.yml`, no environment, allow `npm publish`. |
| GitHub Packages | Built-in `GITHUB_TOKEN` | The release job needs `packages: write` (already configured). After the first publication, set all seven packages to Public in their Package settings. |

The GitHub publisher gets its token explicitly from GoReleaser. It creates a
temporary npm user configuration containing a `GITHUB_TOKEN` variable reference,
routes the `@deahesi` scope to GitHub, and removes the configuration on success or
failure. The token itself is not written to disk or placed in command arguments.
No additional GitHub Packages secret is required. Packages are associated with
this repository through their existing `repository` metadata. For a package
previously created elsewhere, grant this repository Actions write access in that
package's settings before using `GITHUB_TOKEN`.

To resume only GitHub Packages using the original, unchanged release files:

```sh
node build/npm.mjs publish 0.1.4 dist/deahesi-agents-toolchain-0.1.4.tar.gz github
```

Use the actual version and supply `GITHUB_TOKEN` through the environment. Local
publication requires a personal access token (classic) with package write access;
CI uses its built-in token. The existing `node build/npm.mjs publish` command
continues to target npmjs.com and reads the version from `dist/metadata.json`.

## Installing from GitHub Packages

The normal `npm install -g @deahesi/agents-toolchain` command uses npmjs.com.
For the GitHub mirror, authenticate locally and configure scope routing:

```sh
npm login --scope=@deahesi --auth-type=legacy --registry=https://npm.pkg.github.com
npm config set @deahesi:registry https://npm.pkg.github.com
npm install -g @deahesi/agents-toolchain
```

Use your GitHub username and a personal access token (classic) with `read:packages`
as the password. Scope routing also makes npm fetch the platform dependency from
GitHub Packages. This user-level setting affects all `@deahesi` packages. To return
the scope to npmjs.com:

```sh
npm config delete @deahesi:registry
```

See [GitHub's npm registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-npm-registry).
