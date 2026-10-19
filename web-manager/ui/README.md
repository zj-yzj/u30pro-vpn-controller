# U30 Pro VPN 网页插件

这是项目自己的 VPN 面板，使用 Shadow DOM 隔离样式。它兼容设备上**已经安装并启用 root 管理功能的 UFI-TOOLS 环境**，依赖该环境提供的已认证 `runShellWithRoot` 方法，以及另行安装的 `/data/adb/u30pro-vpn/webctl`。本目录不包含 UFI-TOOLS 本体、其登录页面或 requests.js / crypto.js，也不会安装或取得设备 root。

## 构建与离线预览

在仓库根目录运行，需要 Node.js 18 或以上：

```powershell
npm ci
npm run test:ui
npm run build:ui
npm run test:preview
npm run preview
```

默认生成文件写入仓库 `build/web-ui/`，不写回源码目录。浏览器打开终端显示的本机地址，默认 `http://127.0.0.1:18771/`。预览使用模拟节点、临时生成的模拟密钥与内存状态，不连接设备；输入 `https://fail.example/test` 可以查看模拟订阅更新失败时保留原节点的状态。

可以为构建、预览及 HTTP 测试指定相同的输出目录：

```powershell
npm run build:ui -- --out C:/Temp/u30pro-ui-preview
npm run test:preview -- --out C:/Temp/u30pro-ui-preview
npm run preview -- --out C:/Temp/u30pro-ui-preview --port 18771
```

也可以设置 `U30PRO_UI_OUT`。测试环境若要把 npm 依赖放在仓库外，可将 `U30PRO_NODE_RUNTIME` 设为含 `node_modules` 的 npm 项目目录；未设置时按标准 npm 模块解析查找仓库依赖。此设置不改变依赖版本或完整性校验。

## 在已有 UFI 环境导入

生成的 `build/web-ui/U30ProVPN.txt` 是自包含插件，可在已有 UFI 的“插件功能 → 添加插件(.txt文件) → 提交”中导入。插件格式标记为 `U30ProVPN`。只有在已有后端正确安装并完成认证后，真实开关、节点选择与订阅操作才可用。

`logic.js` 负责固定的后端命令和加密请求，`panel.js` 负责界面。节点名称通过文本 DOM 方法显示。写操作使用 RSA-OAEP-SHA256 与 AES-256-GCM，并使用浏览器安全随机数；订阅链接不会以明文拼入 root 命令。网页上的“刷新”读取后端现有状态，不发起节点测速。设备原厂界面或 UFI 外层页面的内容不由本面板控制。

## 第三方加密库

唯一 npm 依赖固定为 `node-forge@1.4.0`。`package-lock.json` 锁定 npm 包，构建与逻辑测试另行核对 `dist/forge.min.js` 的固定 SHA-256：

```text
d9b9074e6861200d676e25dcfe97889db5e8f4150bb766d29c491ad709a51b58
```

本项目按 Forge 提供的 BSD-3-Clause 选项使用它。完整版权、条件与免责保存在 `FORGE-LICENSE.txt`，也会保留在生成插件和预览的头部。生成插件、预览不加载 CDN 或其他外部资源。`build-manifest.json` 记录构建结果大小和校验值。
