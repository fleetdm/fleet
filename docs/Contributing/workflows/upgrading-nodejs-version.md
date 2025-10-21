# Upgrading the Node.js version used to build Fleet

## How Node.js releases work

> Major Node.js versions enter Current release status for six months, which gives library authors time to add support for them. After six months, odd-numbered releases (9, 11, etc.) become unsupported, and even-numbered releases (10, 12, etc.) move to Active LTS status and are ready for general use. LTS release status is "long-term support", which typically guarantees that critical bugs will be fixed for a total of 30 months. Production applications should only use Active LTS or Maintenance LTS releases.

The Node.js project maintains a [release schedule](https://nodejs.org/en/about/releases/) on the Node.js site, and more detailed information can be found [on Github](https://github.com/nodejs/release?tab=readme-ov-file#release-schedule).

## Updating Node.js locally

### Using `asdf`

If you're using `asdf`, you can install the new version of Node.js using `asdf install` and set it as the default for the project using `asdf set`. For example, to upgrade to Node.js 24.10.0, you would run:

```shell
$ asdf install nodejs 24.10.0
$ asdf set nodejs 24.10.0
```

Verify that the new version is set as the default by running any of the following commands:

```shell
$ asdf current nodejs
$ node --version
$ cat .tool-versions | grep nodejs
```

### Using `nvm`

If you're using `nvm`, you can install the new version of Node.js using `nvm install` and set it as the global default using `nvm alias default`. For example, to upgrade to Node.js 24.10.0, you would run:

```shell
$ nvm install 24.10.0
$ nvm alias default 24.10.0
```

### Update `package.json`

Update the project's `package.json` file's `"engines"` key to include the new version of Node.js. Be sure to include the `^` in front of the version number to pin it to the major version. For example, to upgrade to Node.js 24.10.0, the `package.json` file would look like:

```jsonc
{
  "engines": {
    "node": "^24.10.0"
  }
}
```

#### Update `npm`, install `yarn`

Get any updates for `npm` using `npm i -g npm`. Then, install `yarn` using `npm i -g yarn`. You can check the versions with `npm --version` and `yarn --version`, respectively.

## Updating build scripts

In CI, Fleet uses `actions/setup-node` to select the Node.js version. We configure it to read from the repository root `package.json` using `node-version-file: package.json` so it respects the `engines.node` semver range.

### What `check-latest` does

When `check-latest: true` is set on `actions/setup-node`:
- The action resolves the latest available Node.js version that satisfies the provided version or range (for example, the newest `24.x` satisfying `^24.10.0`).
- If a runner already has an older patch/minor of that major cached, `check-latest` tells the action to ignore that stale cache and fetch the newer matching version instead.
- This helps ensure CI picks up new Node.js patch releases (including security fixes) automatically, without changing your `package.json`.

Important notes:
- If you specify an exact version (e.g., `24.10.0`), `check-latest` has no effect; the exact version will be used.
- With a semver range (e.g., `^24.10.0`), `check-latest` may increase setup time on the first run after a new patch is released because it downloads that newer version. Subsequent runs benefit from cache.
- This option is supported in `actions/setup-node` v3 and later.

In this repository, `check-latest: true` is set where the main Fleet app is built and published:
- `.github/workflows/build-binaries.yaml`
- `.github/workflows/goreleaser-snapshot-fleet.yaml`

For more details, see the `actions/setup-node` documentation: https://github.com/actions/setup-node



## Testing the upgrade

### Testing locally

1. Install dependencies via `yarn install`
2. Run tests and linter to verify that everything is working as expected:
   - `yarn test` and `yarn lint`, or
   - `make test-js` and `make lint-js`
3. Test `make generate-js` and `make generate-dev`
4. Test full `make-build` and `make-serve`
5. Resolve any issues that arise.

### Testing in Github Actions

1. Create a draft pull request from your branch and verify that builds and tests pass in Github Actions.
2. Resolve any issues that arise.

## Updating this guide

As the Fleet project evolves, new areas may need to be touched when upgrading Go versions. Please update this guide with any new files you find that need changing (and remove any files that are no longer relevant).