# 测试记录

[English](testing.en.md) · [项目首页](../README.md) · [构建说明](building.md)

## 实机使用

项目已在自用 U30 Pro 上测试，日常使用正常。这是当前设备的使用结果；其他固件、全新设备安装、断电恢复和完整卸载仍需分别验证。

适配条件和安装方法见[安装说明](install.md)。安装新设备时，建议检查客户端联网、VPN 关闭后的原厂联网，以及重启后的设置恢复。

## v1.3.0 正式版检查

正式版沿用已在自用设备测试的功能，统一版本标识并递增 Android 版本码。发布前重新运行了以下检查：

| 检查 | 结果 |
| --- | --- |
| Go 普通 / diagnostic 测试 | 233 / 234 个测试事件通过 |
| Go vet | 通过 |
| 网页逻辑 | 12 项通过 |
| 模块、打包与版本一致性 | 12 项通过 |
| 安装器核心检查与子进程测试 | 37 项及输出上限、超时检查通过 |
| 圆屏几何 | 通过 |

Windows 下仍跳过同一项 POSIX 文件权限专项。正式版的构建提交、文件摘要和证书信息见 [v1.3.0 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0) 的 `release-manifest.json`、`SHA256SUMS` 和 `APK-CERTIFICATE.txt`。

## v1.3.0-alpha.1 发行检查

以下是这次安装包构建时的记录，源码提交和文件摘要保存在 Release 的 `release-manifest.json` 中。后续文档更新不改变安装包的构建来源。

| 检查 | 结果 |
| --- | --- |
| Go 普通测试 | 233 个测试事件通过 |
| Go diagnostic 测试 | 234 个测试事件通过 |
| Go vet、Linux/arm64 静态构建 | 通过 |
| 网页逻辑 | 12 项通过 |
| 模块与打包 | 10 项通过 |
| 安装器 | 37 项核心检查、子进程输出上限与超时测试通过 |
| 圆屏控制应用 | 加密互操作、手势与圆屏几何检查通过 |
| 发行资产 | APK 签名、ZIP 文件清单与权限、内嵌 ZIP 一致性通过 |

两组 Go 测试均在 Windows 上跳过了一项 POSIX 文件权限专项：`TestBootstrapRejectsUnsafeProductionFileModes`。安装引擎还覆盖了事务中断重入、未知应用拒绝、校验失败、并发请求和延期卸载等模拟场景。

上传后已从 GitHub 重新下载六份资产，核对摘要、模块中的 20 项文件、可执行架构、配置摘要和必要许可证。

测试命令见[构建说明](building.md)。校验下载文件时使用同一 Release 的 `SHA256SUMS`。
