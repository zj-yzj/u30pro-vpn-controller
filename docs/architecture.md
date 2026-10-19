# 架构

[English](architecture.en.md) · [返回首页](../README.md)

![U30 Pro 数据与管理路径示意](assets/architecture.svg)

网页面板与圆屏应用共同使用设备端 Go 管理器，Mihomo 作为独立进程负责代理转发。

```text
UFI 网页面板 ── 既有 UFI 认证 root 接口 ──┐
                                         ├── webctl ── Unix HTTP socket ── Mihomo
圆屏 Android 应用 ── 本地 socket ── webbridge ┘       └── 网关 control.sh
```

## 管理边界

`webctl` 负责目标开关状态、节点选择、订阅规范化、持久化和操作事务。订阅内容在设备端处理，不提交到公共转换服务。网页与原生请求使用 RSA-OAEP-SHA256/AES-256-GCM 封装；root 私钥在设备运行时生成，源码没有预置私钥。

`webbridge` 限制请求类型，通过内核 peer credentials、Package Manager 信息和 APK 摘要识别应用。圆屏控制应用没有 Internet 权限，也不直接取得 root。包名和 socket 名称是配套协议的一部分；修改时需要同时修改双方与安装信任记录。

## 安装路径

Magisk ZIP 与独立 `com.codex.u30vpn.installer` APK 使用相同模块字节。安装器 APK 经用户 root 授权执行 `magisk --install-module`；它也无 Internet 权限，核心的官方在线下载由 root helper 完成。`u30setup` 先验证固定设备/文件/网络对象基线，准备时仅暂存模块，重启后再协调 runtime、APK、trust 和提交。

首次状态没有订阅或节点、VPN 关闭，管理密钥在设备生成；信任记录仅接受实际 APK 与发行物固定摘要匹配的 UID/路径。未知已有运行目录或同包应用会被拒绝。完整流程、冷启动行为与卸载保留数据见[安装与恢复](install.md)，测试记录见[测试与设备验证](testing.md)。

Mihomo controller 使用设备本地 Unix socket，不为本项目新增 LAN TCP 管理端口。网页依赖 UFI 的现有 root 管理权限；该环境本身的安全性不由请求加密替代。

## 测速语义

只有用户触发列表刷新或单节点测速时，才提交相应的连通性测量任务。单节点测速保留当前手动选择。前台界面及可选悬浮入口读取本地状态和可见性；状态读取获取已有结果，不发起外部测量。

订阅更新会按状态验证配置及出口可用性；这是更新事务的一部分。参考配置不设置定时 provider health-check 或 URLTest 自动测速组。

## 网关参考

网关脚本需要匹配设备原厂网络环境，包括桥接口、策略路由、DNS 与 SFP 加速状态。参考源码包含保护和恢复逻辑，具体集成前提见[网关文档](../gateway/README.md)。
