# U30 Pro VPN 管理后端

此目录提供 Go 管理程序、原生屏幕控制桥及测试源码。它面向已具备 root、Mihomo 网关和受保护运行目录的 U30 Pro Android 系统，不是刷机包，也不包含从原厂系统开始的安装器。

## 程序与接口

- `cmd/webctl`：状态读取、加密操作请求、任务执行、订阅规范化和持久化恢复。生产运行目录固定为 `/data/adb/u30pro-vpn`，仅允许 root 执行。
- `cmd/nativebridge`：Android 本机应用与管理后端之间的受限控制桥。它检查连接进程身份和本机应用的信任记录，再调用固定命令；不开放新的局域网 TCP 控制端口。
- `cmd/normalize`：将私有的 Clash/Mihomo inline `proxies` 转为初始 provider 和节点元数据。输入、输出都可能含节点凭据，必须保存在公开仓库之外。
- `internal/manager`、`internal/subscription`、`internal/nativebridge`：业务逻辑、输入边界、事务恢复及测试。
- `device/config.yaml`：不含订阅或节点的配置模板。provider 从 root 私有目录加载，controller 使用 Unix socket。
- `ui/`：网页客户端源码；其依赖与构建说明见该目录中的文件。

网页客户端通过设备已有的 UFI 认证和 root helper 调用后端。UFI-TOOLS、Magisk 和 Mihomo 由使用者另外取得、审查并集成，源码包不包含这些项目的程序、固件或安装材料。Android 本机界面位于相邻的 `../native-manager/`。

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

`../gateway/reference/` 提供原有固定布局网关的控制、supervisor 和启动参考源码，其中摘要参数仍需部署方填入。该目录不是通用路由器安装流程，也没有设备历史迁移或刷写脚本。

部署方必须提供受保护目录、初始 provider 和状态、生成的密钥、Android 应用签名与本机信任记录、必要权限、可恢复的安装事务以及适配实际系统的网络前置检查。程序会检查目录所有权、权限、稳定节点 ID、请求时效和重放、操作锁及事务摘要；不能为方便安装而删除这些检查。

订阅地址、节点凭据、管理私钥、签名私钥、运行状态快照、日志、备份、provider 和实际设备信任记录都不属于公开源码。不要将它们提交到仓库。这里没有原安装设备的可直接重跑安装包；构建和单元测试通过也不代表某台未审查设备已具备正确的路由、断网保护或开机恢复能力。
