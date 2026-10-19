# 构建与验证

[English](building.en.md) · [返回首页](../README.md)

所有命令从仓库根目录开始；需要 Node.js/npm、Go 1.27 或兼容版本、JDK 17。Android 构建使用 Platform 35 与 Build Tools 35，工具链由使用者自行安装。

只想安装的用户直接使用 [v1.3.0-alpha.1 Release](https://github.com/zj-yzj/u30pro-vpn-controller/releases/tag/v1.3.0-alpha.1) 和[安装说明](install.md)，无需电脑上的编译工具。

## 双入口发布构建

`release/build.ps1` 要求 PowerShell 7，构建 Linux/arm64 的 `u30setup`、`webctl`、`webbridge`，签署圆屏控制 APK，渲染 supervisor 固定摘要并生成 Magisk ZIP，然后把完全相同的 ZIP 内嵌进独立安装器 APK。它不下载或打包 Mihomo 核心。

先把下列变量设为自己的工具路径、源码树外**全新空输出目录**，以及专用发行 keystore、密码文件与 alias：

```powershell
./release/build.ps1 -Go $goExe -JavaHome $jdkHome -AndroidJar $androidJar `
  -BuildTools $buildTools -OutputDir $releaseOut -Keystore $releaseKeystore `
  -KeystorePasswordFile $passwordFile -KeyAlias $keyAlias
```

Node 可用 `-Node` 显式指定；已在源码树外安装固定 Forge 依赖时，可设置 `U30PRO_NODE_RUNTIME`。构建不自动下载 Android 工具链，不搜索原设备签名。发行身份应与原开发设备隔离、在后续发行中复用并妥善备份；私钥和密码不进入源码或上传文件。

上传文件仅为 `U30ProVPN-1.3.0-alpha.1-Magisk.zip`、`U30ProVPN-1.3.0-alpha.1-Installer.apk`、`U30ProVPN.txt`、`APK-CERTIFICATE.txt`、`release-manifest.json`、`SHA256SUMS`。输出中的 `work/` 是构建中间目录，不是待上传资产。`installer-app/build.ps1` 可单独使用 `-ModuleZip` 编译安装器，参数与主机测试见[安装器说明](../installer-app/README.md)。

构建产物的摘要、签名与 ZIP 一致性检查只说明文件身份；完整验证结果与跳过项应写入 Release 说明。本 alpha 尚无干净设备全流程实机验收，不能因构建成功就宣称首装、网络恢复或卸载已在实机通过。

## 网页插件与离线预览

```sh
npm ci
npm test
npm run build
npm run test:preview
```

具体输出目录与预览命令见 `web-manager/ui/README.md`。预览使用模拟数据，不读取设备订阅，也不连接真实网关。node-forge 版本由 lockfile 固定，构建同时核对其发行文件摘要并保留 BSD 许可头。

## Go 后端

确保 `javac` 和 `java` 位于 PATH，以运行 Java/Go 加密互操作测试；缺少 JDK 时该项可能跳过。

```sh
cd web-manager
go test -count=1 ./...
go vet ./...
```

设备程序使用 Linux/arm64，无需 CGO。以下是 PowerShell 示例：

```powershell
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'arm64'
go build -trimpath -ldflags='-s -w' -o build/webctl ./cmd/webctl
go build -trimpath -ldflags='-s -w' -o build/webbridge ./cmd/nativebridge
```

这些环境变量仅供交叉编译；运行主机测试前应清除 `GOOS`、`GOARCH` 的交叉编译设置。输出目录由 Git 忽略，Git 源码树不提交这些二进制；Release 模块另含已编译的项目程序。

## Android 应用

构建入口为 `native-manager/build.ps1`，支持显式指定工具链和输出目录，默认不需要私钥。参数和签名选项见 `native-manager/README.md`。构建运行主机加密与手势测试；圆屏几何还可单独验证：

```sh
node native-manager/android/tests/round-screen-geometry.mjs
```

## 发布前检查

```sh
node scripts/check-publication.mjs
node scripts/check-doc-links.mjs
```

该检查只输出相对路径、行号和问题类别，不打印匹配到的敏感内容。它检查发布目录结构、敏感文件类型、个人路径、私钥文本和非示例订阅凭据等常见问题；它不能替代人工代码审查。构建测试完成后，发布目录中不应包含生成物或依赖缓存。
