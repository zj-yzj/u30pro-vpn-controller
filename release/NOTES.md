# v1.3.0

## 中文

给已 root 的 U30 Pro 加上圆屏 VPN 控制、节点切换、手动测速和订阅管理。我在自用设备上测试使用正常。

这是 v1.3.0 正式版，与此前自测的 v1.3.0-alpha.1 功能相同，继续使用同一公开签名证书。本版仍只用于全新安装，不提供覆盖已有 alpha 或其他部署的升级。

这次提供两种安装入口，选一个就行：

| 文件 | 安装方式 |
| --- | --- |
| `U30ProVPN-1.3.0-Magisk.zip` | 在 U30 Pro 的 Magisk 中从本地安装 |
| `U30ProVPN-1.3.0-Installer.apk` | 安装到 U30 Pro 本机，打开后授权 root 并安装模块 |
| `U30ProVPN.txt` | 导入已有 UFI-TOOLS，用来录入订阅 |

安装完成后重启，在 UFI 中导入插件并保存自己的订阅，再开启 VPN。首次安装默认关闭，节点列表为空；节点页“刷新”可以整批测速，左滑普通节点可以单独测速，没有定时全节点测速。

适配 **MU5358 / U30 Pro B23、Android API 35、arm64-v8a、Magisk 28.1+**，SELinux 原本需为 Permissive，设备已有可用的 UFI-TOOLS。本版用于全新安装，暂不支持覆盖已有部署。准备阶段会联网下载约 **21.9 MB** 的官方 Mihomo 核心。

[中文安装与恢复](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/install.md) · [使用说明](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/usage.md) · [测试记录](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/testing.md)

下载文件可用 `SHA256SUMS` 校验；`APK-CERTIFICATE.txt` 是签名指纹，`release-manifest.json` 记录安装包的构建来源。原创代码使用 MIT，第三方许可证随模块保留。

## English

On-device VPN controls, node switching, manual latency tests and subscription management for a rooted U30 Pro. I've tested it on my own device and it works normally in daily use.

This is the stable v1.3.0 release, with the same functionality as the previously self-tested v1.3.0-alpha.1 and the same public signing certificate. It still supports fresh installations only; it does not add an upgrade path over an existing alpha or other deployment.

Choose either installation route:

| File | How to install |
| --- | --- |
| `U30ProVPN-1.3.0-Magisk.zip` | Install from local storage in Magisk on the U30 Pro |
| `U30ProVPN-1.3.0-Installer.apk` | Install on the U30 Pro, grant root and install the embedded module |
| `U30ProVPN.txt` | Import into an existing UFI-TOOLS installation to add your subscription |

Reboot after installation, import the plugin, save your own subscription and enable VPN. A fresh install starts with VPN off and an empty node list. Refresh on the node page tests the list; swipe left on a regular node to test it individually. There are no scheduled all-node tests.

Supported setup: **MU5358 / U30 Pro B23, Android API 35, arm64-v8a and Magisk 28.1+**, with SELinux already Permissive and working UFI-TOOLS. This version is for fresh installations. Preparation downloads the official Mihomo core, about **21.9 MB**, over the device's Internet connection.

[Installation and recovery](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/install.en.md) · [Usage](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/usage.en.md) · [Testing](https://github.com/zj-yzj/u30pro-vpn-controller/blob/main/docs/testing.en.md)

Use `SHA256SUMS` to check downloads. `APK-CERTIFICATE.txt` records the signing fingerprint and `release-manifest.json` records the package build provenance. Original code is MIT licensed; third-party licenses are included in the module.
