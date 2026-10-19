# U30 Pro VPN 管理后端

此目录提供 Go 管理程序、原生屏幕控制桥、固定设备首次安装引擎及测试。`v1.3.0` 提供 Magisk ZIP 和内嵌同一 ZIP 的 Android 安装器，要求已有兼容 Magisk、固定 MU5358/B23/API35 profile 与现有 UFI。设备需事先完成固件和 root 准备；下载者从[安装与恢复](../docs/install.md)开始。已有私有部署的升级需要另行处理。

## 程序与接口

- `cmd/webctl`：状态读取、加密操作请求、任务执行、订阅规范化和持久化恢复。生产运行目录固定为 `/data/adb/u30pro-vpn`，仅允许 root 执行。
- `cmd/u30setup`、`internal/installer`：固定 profile/发行清单核验、设备在线获取核心、首装 journal、APK/trust 配置与归属卸载。
- `cmd/nativebridge`：Android 本机应用与管理后端之间的受限控制桥。它检查连接进程身份和本机应用的信任记录，再调用固定命令；不开放新的局域网 TCP 控制端口。
- `cmd/normalize`：将私有的 Clash/Mihomo inline `proxies` 转为初始 provider 和节点元数据。输入、输出都可能含节点凭据，必须保存在公开仓库之外。
- `internal/manager`、`internal/subscription`、`internal/nativebridge`：业务逻辑、输入边界、事务恢复及测试。
- `device/config.yaml`：不含订阅或节点的配置模板。provider 从 root 私有目录加载，controller 使用 Unix socket。
- `ui/`：网页客户端源码；其依赖与构建说明见该目录中的文件。

网页客户端通过设备已有的 UFI 认证和 root helper 调用后端。UFI-TOOLS 与 Magisk 需事先具备，Mihomo 在准备阶段从固定官方资源在线获取；项目 ZIP/APK 不内嵌其核心。Android 圆屏界面位于 `../native-manager/`，独立安装入口位于 `../installer-app/`。

`webctl bootstrap` 在预先建立的 root 私有 Base/web/cache 目录内生成设备身份、合法空 provider 与空节点状态，`desiredEnabled=false`；无外部网络或核心调用。key→provider→state 的断点重入保留已有内容；已有 state 缺少有效 key/provider 时拒绝。`set-enabled=true` 在节点为空时先返回 `not_configured`，不写状态、不调用网关。首次订阅更新成功也保持关闭。旧 `init` 仍要求已有节点数据并保留旧集成语义。

## 手动测速语义

`device/config.yaml` 的 `Subscription` 为 file provider，没有 `health-check`；`VPN` 和 `Auto` 都是 `select` 组，没有周期 `interval` 或 `url-test`。程序不会因状态轮询、打开页面或切换页面而自动测试所有节点。

手动批量测速只测试当前订阅节点，并用本轮成功结果更新内层 `Auto` 的选择；外层 `VPN` 的用户选择保留。单节点测速只检查指定的已保存稳定节点 ID，不切换节点或选择组。没有测速记录时，延迟和连通状态可以为空；历史失败记录也不等于当前所有业务访问失败。

VPN 开关和故障恢复保留各自的运行状态检查；VPN 开启时应用订阅还会验证当前出口的 HTTPS 联网。这些检查不等同于周期节点测速；网关 supervisor 的 `healthy()` 检查的是进程、TUN、路由和保护状态，不是向全部节点发送测速请求。

## 构建与验证

本包以 Go 1.27.2 验证，模块声明的最低 Go 版本为 1.27.0。完整测试还需要 JDK 17 的 `java` 和 `javac` 在 `PATH` 中，并保留本目录与 `../native-manager/android` 的相对位置。Java 互操作测试仅在内存和临时目录生成测试密钥，不使用设备密钥。

在本目录运行：

```sh
go test -json -count=1 ./...
go test -json -count=1 -tags diagnostic ./...
go vet ./...
go vet -tags diagnostic ./...
```

`diagnostic` 构建标签包含额外的固定目标诊断源码及测试；普通发布构建不启用该标签。测试使用本地模拟服务和合成数据，不需要连接设备或访问真实订阅。

将输出位置设为仓库之外，例如：

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /path/outside/source/webctl ./cmd/webctl
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /path/outside/source/webbridge ./cmd/nativebridge
```

PowerShell 中应先分别设置 `$env:CGO_ENABLED`、`$env:GOOS`、`$env:GOARCH`，再执行相应的 `go build`。所需 Go 模块由 `go.mod`、`go.sum` 记录；无需复制 SDK、模块缓存或编译缓存。

## 集成边界与私有数据

[`../gateway/reference/`](../gateway/reference/README.md) 提供固定布局网关的控制、supervisor 和启动模板，摘要参数需由部署方填入。移植前先核对其中的网络与运行目录前提；首次安装使用发行 ZIP/APK。

发行安装器在其固定范围内准备目录、空状态、设备密钥、发行 APK 信任与首装恢复；其他移植需要审查对应基线。程序检查所有权、权限、稳定节点 ID、请求时效和重放、操作锁及摘要，这些校验应一并保留。卸载仅作用于已证明归属的内容，并保留私有数据备份；PM 或网络管理不可用时会延后清理。冷启动行为与迁移边界见安装页。

订阅地址、节点凭据、管理私钥、签名私钥、运行状态快照、日志、备份、provider 和实际设备信任记录保存在公开仓库之外。主机测试与设备验证记录见[测试与设备验证](../docs/testing.md)。
