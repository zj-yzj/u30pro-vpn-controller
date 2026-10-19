# Building and verification

[简体中文](building.md) · [Project home](../README.en.md)

Run commands from the repository root unless another directory is stated.
Install Node.js 18 or newer, npm, Go 1.27 or a compatible toolchain, and JDK 17.
Android builds also require Android Platform 35 and Build Tools 35.
SDKs and signing identities are supplied by the person building the project.

Downloaders can use the [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1)
and [installation guide](install.en.md) without installing host build tools.

## Build both release entries

`release/build.ps1` requires PowerShell 7 and builds Linux/arm64 `u30setup`, `webctl` and `webbridge`, signs
the circular control APK, renders fixed supervisor digests and packages a Magisk
ZIP. It then embeds those exact ZIP bytes in the separate installer APK. It
neither downloads nor packages a Mihomo executable.

Set the following variables to your toolchain paths, a **fresh empty directory
outside the checkout**, and a dedicated release keystore/password file/alias:

```powershell
./release/build.ps1 -Go $goExe -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $releaseOut -Keystore $releaseKeystore `
  -KeystorePasswordFile $passwordFile -KeyAlias $keyAlias
```

`-Node` can select Node explicitly. Use `U30PRO_NODE_RUNTIME` for pinned Forge
dependencies installed outside the checkout. The build downloads no Android
toolchain and does not discover the original device signer. Keep a dedicated
release identity separate from the development device, reuse it for subsequent
releases and back it up privately. Keys/passwords are never upload artifacts.

Upload only `U30ProVPN-1.3.0-alpha.1-Magisk.zip`,
`U30ProVPN-1.3.0-alpha.1-Installer.apk`, `U30ProVPN.txt`, `APK-CERTIFICATE.txt`,
`release-manifest.json` and `SHA256SUMS`. The output `work/` directory contains
intermediate files and is not an asset. For the standalone installer build with
`-ModuleZip`, parameters and host tests, see [installer documentation](../installer-app/README.md).

Checksums, signatures and embedded-ZIP identity checks establish artifact identity.
Record actual verification results and skips in Release notes. This alpha has
no complete fresh-device acceptance; a successful build does not establish
physical installation, network restoration or teardown success.

## Web plugin and offline preview

```sh
npm ci
npm test
npm run build
npm run test:preview
npm run preview
```

Open the loopback URL printed by the preview server. The preview runs with
synthetic nodes and in-memory state; it does not connect to the hotspot or
read its subscription. The generated plugin, HTML, JavaScript and manifest
are placed in `build/web-ui/` by default, outside the source directories.
This build directory is ignored by Git.

The lockfile pins node-forge. The builder additionally verifies the upstream
distribution file's SHA-256 and embeds the selected complete BSD notice.
The generated plugin loads no CDN scripts.

See [web UI documentation](../web-manager/ui/README.md) for custom output
directories, the `U30PRO_UI_OUT` setting, and externally installed test dependencies.

## Go backend

Put `java` and `javac` on PATH so the Java/Go envelope interoperability test
runs. That test is skipped when the JDK is unavailable; a successful command
with skipped tests is not equivalent to the complete verification.

```sh
cd web-manager
go test -count=1 ./...
go test -count=1 -tags diagnostic ./...
go vet ./...
go vet -tags diagnostic ./...
```

For Android arm64 targets, build Linux/arm64 executables with CGO disabled.
Example in PowerShell, from `web-manager/`:

```powershell
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'arm64'
go build -trimpath -ldflags='-s -w' -o build/webctl ./cmd/webctl
go build -trimpath -ldflags='-s -w' -o build/webbridge ./cmd/nativebridge
```

Clear cross-compilation environment variables before returning to host tests.
The normal build does not enable the optional `diagnostic` build tag.
Git source excludes these executables; Release modules separately contain the
compiled project programs.

## Android application

Use `native-manager/build.ps1` with `JAVA_HOME` and `ANDROID_SDK_ROOT`, or
explicit toolchain paths. The build does not need Gradle. From the repository root:

```powershell
./native-manager/build.ps1 -JavaHome $jdkHome -AndroidJar $androidJar -BuildTools $buildTools
node ./native-manager/android/tests/round-screen-geometry.mjs
```

The default output is an **unsigned APK** under `native-manager/build/`.
The script compiles Android sources and executes host crypto and swipe tests.
For `TestOnly`, custom output directories, explicit signing options and the
smoke test, see [native application documentation](../native-manager/README.md).

Keep signing keys and password files outside the repository. Installing an
application requires its own signing identity and root-side trust provisioning;
a successful build does not perform those operations.

## Publication checks

```sh
node scripts/check-publication.mjs
node scripts/check-doc-links.mjs
```

Run publication checks on a clean source tree without build outputs or dependency
caches. The publication scanner reports locations and categories rather than
matching secret values. It checks common sensitive file types, personal paths,
private-key material, non-example credential URLs and supported image assets.
The link checker validates local Markdown file and image references.
These checks support human review; they do not prove that arbitrary future
changes contain no sensitive material.
