# UFI 高级后台安装材料存档 / UFI installation archive

为 U30 Pro 安装教程提供配套下载，保留维护者提供的整套 UFI 安装材料。**已有可用 UFI 的设备可跳过安装。** VPN 控制器正式版仍是 [v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0)。

| 下载 | 用途 |
| --- | --- |
| [完整安装材料 ZIP](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/U30Pro-UFI-Tools-Archive-20261019.zip) | 约 80 MB；含 Windows 一键安装器 1.6、4.1.2 / 4.1.1 APK、原始 PDF 手册及 ADB 工具 |
| [UFI 4.1.2 APK](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS_WEB_V4.1.2_20260819_1854.apk) | 约 23 MB；本教程手动 ADB 路线使用的版本，安装到 U30 Pro |
| [4.1.2 上游源码参考](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/UFI-TOOLS-4.1.2-Source-Reference.zip) | 可选，供查阅源码和原始许可证，日常安装不需要下载 |
| [文件清单](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/ufi-file-manifest.json) / [SHA256SUMS](https://github.com/zj-yzj/u30pro-vpn-controller/releases/download/ufi-tools-archive-20261019/SHA256SUMS) | 核对文件完整性 |

按[中文完整教程](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.md)操作。Windows 一键安装器在电脑上运行，手动路线则安装单独下载的 4.1.2 APK；两者选一种即可。ZIP 中的 4.1.1 作为旧版备份保留。临时无线 ADB 工具只标注适用于部分固件，不是通用 root 方法；USB ADB 已可用时无需运行它。

UFI-TOOLS 由 [Minikano / kanoqwq](https://github.com/kanoqwq/UFI-TOOLS) 开发。存档完整保留 13 个原始文件的字节、文件名和相对路径，仅附加说明、哈希、署名和许可材料。UFI 原创源码采用 MIT，其他独立工具及组件保留各自的权利和许可状态，不能将整套材料统一视为本项目的 MIT 代码。

源码参考固定于 [51b86de](https://github.com/kanoqwq/UFI-TOOLS/tree/51b86de2036859050094b24780a55ad7ff687d78)，附有逐文件来源记录。它不代表已复现整个 APK 或补齐所有内置第三方二进制的对应构建源码。未知构建来源记录为 `NOASSERTION`；准备存档时未执行这些程序，也未进行完整二进制安全审计。

---

This archive supplies the UFI files used by the U30 Pro setup guide. **Skip UFI installation if it already works on your device.** The current VPN controller release remains [v1.3.0](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0).

- **Complete installation ZIP:** about 80 MB, containing Windows one-click installer 1.6, both original APKs, the PDF manual and ADB tools.
- **Standalone UFI 4.1.2 APK:** about 23 MB, the version used by the manual ADB route. Install this APK on the U30 Pro.
- **Source reference and receipt:** optional for installation; provided for source access and attribution.
- **Manifest and SHA256SUMS:** identify the supplied files and verify download integrity.

Follow the [English setup guide](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/setup.en.md). Choose either the Windows installer on your computer or the pinned 4.1.2 APK with manual ADB. The older 4.1.1 APK is retained as a historical backup. The temporary wireless-ADB helper lists only selected firmware; it is not a general root method and is unnecessary when USB ADB already works.

[UFI-TOOLS](https://github.com/kanoqwq/UFI-TOOLS) is developed by Minikano / kanoqwq. All 13 supplied files retain their original bytes, names, paths and embedded notices. The added material consists of instructions, hashes, attribution and license records. UFI's original source uses MIT; other independent tools and components retain their own rights and license status. The controller's MIT license does not cover the whole collection.

The source reference is pinned to commit [51b86de](https://github.com/kanoqwq/UFI-TOOLS/tree/51b86de2036859050094b24780a55ad7ff687d78), with a receipt for all 231 files. It does not establish a reproducible full APK build, an official publisher signature or complete corresponding source for every embedded native binary. Unconfirmed build provenance is recorded as `NOASSERTION`. No supplied program was executed during preparation, and this is not a complete binary security audit.
