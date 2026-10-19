# 设备集成边界

[English](device-integration.en.md) · [返回首页](../README.md)

`v1.3.0-alpha.1` 提供面向固定设备条件的首次安装器：Magisk ZIP，以及内嵌同一 ZIP 的 Android 安装器 APK。下载者先读[安装与恢复](install.md)。设备需要事先完成固件、root 与 Magisk 准备；已有私有部署的升级需要另行处理。

## 已有运行环境

- device/model `MU5358`，`arm64-v8a`，API 35，display 精确为 `U30ProV1.0.0B23`。
- 已正常工作的 Magisk，版本码至少 `28100`；SELinux 已为 Permissive，不由安装器降低保护级别。
- `br0` 为 `192.168.0.1/24`、IPv4 forwarding 已开启、SFP 基线 `1`，且本项目保留对象没有冲突。
- 准备阶段从官方上游在线下载并核验的 Android arm64 Mihomo；不在 ZIP/APK 内嵌，离线首次安装不可用。
- 固定运行根目录 `/data/adb/u30pro-vpn`，由 root 持有并限制其他用户访问。
- 运行目录由安装器建立；首次 provider 与节点为空、VPN 关闭，管理密钥在设备生成。
- 网页面板需要已安装 UFI 及其认证 root 接口。源码不包含 UFI 的认证实现。

`web-manager/device/config.yaml` 是无订阅凭据的参考配置。它指向运行目录里的 provider，没有可直接使用的节点。`Auto` 和 `VPN` 均使用 select 组，没有定时测速配置。`webctl bootstrap` 初始化空状态；无节点时开启返回 `not_configured`，不会先保存开启意图或调用网关。原 `webctl init` 用于已有节点输入的旧集成流程，不能用作空订阅首装。

## 安装阶段与恢复

`u30setup` 的准备阶段核验固定 profile、文件清单、空间和核心，只暂存模块。启动阶段等待 Android/PackageManager/网络管理可用，记录安装阶段，在 root 私有 staging 中初始化运行目录，再安装发行控制 APK、生成本机 trust 并提交。首次过程不启动核心、不改路由、不自动开启 VPN。

已存在的私有运行目录、未知同包 APK、未知模块与冲突对象会阻止首次安装。仅已识别的同版本中断事务可恢复，保留密钥和已提交配置。后续正常启动按保存的目标状态处理 VPN；启用状态在冷启动时有网络保护尚未就绪的窗口。

卸载先受控停止，再清理可证明归属的对象，并保留 root 私有备份。PM 或网络管理未就绪时会延后清理，详见[恢复说明](install.md#卸载与保留的数据)。

## 原生应用信任

应用默认包名为 `com.codex.u30vpn`。root helper 的信任记录必须以安装后的实际身份生成，包括 schema、包名、UID、安装 APK 路径和 SHA-256；不能复用其他设备的 UID 或旧 APK 摘要。信任记录本身不随源码分发。

Android 独立构建默认输出未签名 APK；发布构建使用保存在源码树外的专用发行身份。安装器只在实际安装 APK 摘要与发行物固定摘要一致时，用实际 UID/路径生成 trust。未知 APK 和同包换签名冲突会被拒绝，安装器不会卸载已有应用来绕过签名冲突。

`gateway/reference/` 仍是固定网关源码与模板；真正的发行模块启动入口在 `release/module/`，安装阶段由 `web-manager/internal/installer/` 处理。单独复制参考模板不等同于运行安装器。移植仍需重新审查网络基线、摘要参数与恢复机制。

## 验证范围

自动测试覆盖加密协议、输入校验、事务恢复、节点操作及界面几何，使用模拟控制器和合成节点。日常使用已在作者自用的 U30 Pro 上测试正常；完整的测试范围和设备验证记录见[测试与设备验证](testing.md)。
