# Third-party notices

The root MIT license covers the original controller code and documentation.
It does not replace the licenses of the following dependencies or external
programs. This repository does not bundle firmware, upstream tool repositories,
UFI authentication assets, signing identities, or a Mihomo executable.

## node-forge 1.4.0

- Upstream: https://github.com/digitalbazaar/forge/tree/v1.4.0
- Used by: the browser-side cryptographic envelope implementation.
- Selected license: BSD-3-Clause, under the upstream dual-license offer.
- Copyright: Digital Bazaar, Inc.
- License copy: [web-manager/ui/FORGE-LICENSE.txt](web-manager/ui/FORGE-LICENSE.txt).
- The build verifies the pinned upstream `dist/forge.min.js` SHA-256 and embeds
  the complete selected license in the generated plugin and preview bundle.

## gopkg.in/yaml.v3 v3.0.1

- Upstream: https://github.com/go-yaml/yaml/tree/v3.0.1
- Used by: Go subscription parsing and normalization.
- Licensing is file-specific: the libyaml-derived files listed in the upstream
  LICENSE use MIT; the remaining files use Apache-2.0. These are not an
  either/or license choice for the whole dependency.
- Copyright holders include Kirill Simonov and Canonical Ltd.
- Preserved texts: [LICENSE](third_party/licenses/yaml.v3/LICENSE),
  [NOTICE](third_party/licenses/yaml.v3/NOTICE), and the complete
  [Apache-2.0 text](third_party/licenses/APACHE-2.0.txt).

## Go runtime and standard library

- Upstream: https://go.dev/
- The Go SDK itself is not included. Compiled Go helpers contain runtime and
  standard-library code. Include the [Go BSD license](third_party/licenses/go/LICENSE)
  with any such binary distribution.

## Mihomo: external runtime dependency

- Upstream: https://github.com/MetaCubeX/mihomo
- Reference release: v1.19.32.
- Source commit used for the referenced runtime:
  `88dcbf7f1614a67c3b36b848ee3592dfa92ada36`.
- Upstream license:
  https://github.com/MetaCubeX/mihomo/blob/88dcbf7f1614a67c3b36b848ee3592dfa92ada36/LICENSE
- The controller communicates with a separately installed Mihomo process over
  its Unix HTTP controller. This repository does not distribute that executable
  or its source. A release that adds the executable must separately handle its
  license, notices, and corresponding-source requirements.
- The `v1.3.0-alpha.1` Magisk ZIP and Android installer APK do not embed the
  executable. Their device-side preparation downloads the pinned official
  upstream asset directly with TLS, size and digest verification. This project
  makes no complete corresponding-source distribution claim for that separately
  acquired upstream binary. See [installation details](docs/install.en.md#the-core-is-downloaded-online).

## UFI-TOOLS, Magisk, and Android: integration environment

- UFI-TOOLS: https://github.com/kanoqwq/UFI-TOOLS
- Magisk: https://github.com/topjohnwu/Magisk
- Android: https://source.android.com/

The web plugin uses UFI's existing authenticated `runShellWithRoot` interface.
The control and installer apps use Android platform APIs. The installer calls
the existing Magisk CLI after user root consent; it does not ship Magisk. The
gateway startup reference is for an already rooted device. No copied UFI HTML, requests.js, CryptoJS bundle,
Magisk package, Android SDK, or root/flash tool is distributed here. Names and
interface markers identifying those integrations are retained accurately.
