# Documentation illustrations

Original, self-contained SVG illustrations for the U30 Pro VPN Controller documentation. They use only SVG geometry and text, without scripts, external images, external fonts, or `foreignObject` elements. Original illustration content is covered by the repository's MIT license.

| Asset | Size | Purpose |
| --- | --- | --- |
| `overview.svg` | 1200 × 720 | U30 Pro dedicated project overview and schematic round-screen device |
| `native-ui.svg` | 1200 × 780 | Home, left-swiped node, and manual single-node test illustrations |
| `architecture.svg` | 1200 × 790 | Client traffic and local management paths |
| `web-preview.png` | 1000 × 1050 | Browser capture of the project's offline mock web panel |

These are **technical illustrations, not device photographs or screenshots**. The only sample node, **示例节点 A / Sample A**, is fictional. No subscription, endpoint, credential, device capture, latency measurement, or other live data is represented.

The UI illustration follows the native application's 240 × 240 panel geometry. Each 324 px circle is a uniform 1.35× enlargement, and each centered 221.4 × 221.4 px dashed guide corresponds to its 164 × 164 px interactive safe area. The separate title follows the application's placement above that area. Labels are simplified for documentation legibility; this is not a pixel-exact rendering of Android widgets. Text is at least 16 px at the SVG's native dimensions, with local system-font fallbacks for Chinese and English.

The architecture reflects the existing U30 Pro rooted stock-Android integration: the Android app uses `webbridge`, the web plugin uses the existing UFI authenticated root interface, and both reach `webctl`. The Mihomo controller uses a local Unix socket. The diagram does not introduce a LAN management listener, firmware replacement, or new kernel.

Source references: the repository root `README.md`, `native-manager/README.md`, `native-manager/android/src/com/codex/u30vpn/MainActivity.java`, and `docs/architecture.md`.

`web-preview.png` is a screenshot of the actual project web panel running with
`preview-mock.js`, captured in a local browser. The preview banner explicitly
identifies simulated data. The five visible mock nodes, geographical labels,
delay values and `subscription.example` host are synthetic; no device was
contacted. The separate HTML-injection test fixture was omitted from the
in-memory demonstration list for this documentation capture. No product
source was changed for the screenshot. The PNG contains no text or EXIF
metadata chunks. It is not evidence of device connectivity or performance.
