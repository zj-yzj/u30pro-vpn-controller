# Documentation illustrations

Original, self-contained SVG illustrations for the U30 Pro VPN Controller documentation. They use only SVG geometry and text, without scripts, external images, external fonts, or `foreignObject` elements. Original illustration content is covered by the repository's MIT license.

| Asset | Size | Purpose |
| --- | --- | --- |
| `overview.svg` | 1200 × 720 | U30 Pro dedicated project overview and schematic round-screen device |
| `native-ui.svg` | 1200 × 780 | Home, left-swiped node, and manual single-node test illustrations |
| `architecture.svg` | 1200 × 790 | Client traffic and local management paths |
| `web-preview.png` | 1000 × 1050 | Browser capture of the project's offline mock web panel |

The SVGs are **technical illustrations**. Their sample node, **示例节点 A / Sample A**, is fictional. `web-preview.png` is a browser screenshot of the project's offline mock panel. All node names, locations, delay values and subscription hosts shown in these assets are synthetic.

The UI illustration follows the native application's 240 × 240 panel geometry. Each 324 px circle is a uniform 1.35× enlargement, and each centered 221.4 × 221.4 px dashed guide corresponds to its 164 × 164 px interactive safe area. The title sits above that area, with simplified labels for documentation legibility. Text is at least 16 px at the SVG's native dimensions, with local system-font fallbacks for Chinese and English.

The architecture shows the U30 Pro's rooted stock-Android integration: the Android app uses `webbridge`, the web plugin uses the existing UFI authenticated root interface, and both reach `webctl`. The Mihomo controller uses a local Unix socket.

Source references: the repository root `README.md`, `native-manager/README.md`, `native-manager/android/src/com/codex/u30vpn/MainActivity.java`, and `docs/architecture.md`.

`web-preview.png` comes from the project web panel running locally with
`web-manager/ui/preview-mock.js`. Its banner identifies the simulated data,
including five mock nodes and the `subscription.example` host. The preview
uses in-memory state and does not connect to a device. The screenshot is
covered by the repository's MIT license, with third-party components governed
by their notices in `THIRD_PARTY_NOTICES.md`.
