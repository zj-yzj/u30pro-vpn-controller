# Third-party notices

The root MIT license covers the original controller code and documentation.
It does not replace the licenses of the following dependencies or external
programs. The controller source and v1.3.0 installer packages do not bundle
firmware, upstream tool repositories, UFI authentication assets, signing
identities, or a Mihomo executable. Separate third-party B23 root and UFI
installation archives are described below; the controller's MIT license does
not cover those collections.

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
- The `v1.3.0` Magisk ZIP and Android installer APK do not embed the
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
gateway startup reference is for an already rooted device. No copied UFI HTML,
requests.js, CryptoJS bundle, Magisk package, Android SDK, or root/flash tool is
included in the controller source or v1.3.0 installer packages. Names and
interface markers identifying those integrations are retained accurately.

## Separate B23 root toolkit archive

- Archive: [b23-rootkit-archive-20261019](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/b23-rootkit-archive-20261019).
- Scope: the historical third-party toolkit supplied and used by the maintainer,
  with original operational file bytes retained. Two old communication logs,
  PDB debug symbols and one empty marker are omitted. It is not an official
  vendor release or part of the controller's two installer routes.
- The Magisk APK is byte-identical to official
  [Canary 28103](https://github.com/topjohnwu/Magisk/releases/tag/canary-28103),
  source commit `1e3edb888375c75777120a018637b5d945df6d15`. The archive contains
  the full GPL v3 text. The same Release provides a separate source companion
  with that tree, its ten pinned submodules, original licenses and a source receipt.
- Original AndroidX, UNISOC/SPRD and Microsoft component notices and file
  identities are retained in the original files. Exact provenance and applicable
  license statements for the other independent scripts, images and tools remain
  unconfirmed and are recorded as `NOASSERTION`. They are not relabeled MIT or
  covered by the Magisk source's license.
- The archive contains its own `THIRD-PARTY-NOTICES.md`, `MAGISK-SOURCE.md`,
  file manifest and checksums. Device scope, bundled wireless-ADB/update changes
  and backup requirements are documented in the [B23 root record](docs/root-b23.en.md).

## Separate UFI installation archive

- Archive: [ufi-tools-archive-20261019](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/ufi-tools-archive-20261019).
- Scope: all 13 original files supplied by the maintainer, with original bytes,
  names and relative paths retained. It includes UFI 4.1.1 and 4.1.2 APKs, the
  Windows one-click installer 1.6, the original PDF manual and ADB helpers.
  The guide recommends the 4.1.2 APK; 4.1.1 is a historical backup.
- UFI-TOOLS author: Minikano / [kanoqwq](https://github.com/kanoqwq/UFI-TOOLS).
  The pinned upstream source at
  [commit 51b86de](https://github.com/kanoqwq/UFI-TOOLS/tree/51b86de2036859050094b24780a55ad7ff687d78)
  declares version 4.1.2. Its MIT license, including
  `Copyright (c) [2025] [Minikano]`, is included unchanged as
  `licenses/UFI-TOOLS-MIT.txt` in the installation archive. The same Release
  provides the upstream source snapshot and a receipt covering its 231 files.
- The supplied APK's package is `com.minikano.f50_sms`, version 4.1.2,
  version code 20260819. Its hash identifies the supplied file; the source
  snapshot does not establish an official publisher signature, a reproducible
  whole-APK build or complete corresponding source for all native components.
- Both APKs retain their embedded notices, including
  `doc-files/speclicense.html` and `okhttp3/internal/publicsuffix/NOTICE`.
  Native tools and libraries, including ADB, curl, iperf3, jq, socat and ttyd,
  retain their own terms. UFI's MIT license does not replace them.
- Exact build provenance and applicable license statements for the independent
  Windows installers and helper tools remain unconfirmed (`NOASSERTION`).
  The supplied Windows ADB files, README and shortcut are preserved as received;
  their inclusion does not establish an official Google distribution identity.
- The archive adds bilingual instructions, a file manifest, checksums, source
  references and these attribution boundaries. No supplied executable was run
  while preparing the archive, and this is not a complete binary security or
  component-license audit. See the [setup guide](docs/setup.en.md) for usage.
