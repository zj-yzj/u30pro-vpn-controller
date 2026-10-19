import {readFile,writeFile,mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {createHash} from 'node:crypto';
import vm from 'node:vm';
import {loadPinnedForge,forgeSHA256} from './forge-package.mjs';
import {outputOptions} from './output-options.mjs';
const dir=path.dirname(fileURLToPath(import.meta.url));
const {directory:out}=outputOptions();
const [{source:forge,bsd},logic,panel,css,mock]=await Promise.all([
  loadPinnedForge(),
  readFile(path.join(dir,'logic.js'),'utf8'),readFile(path.join(dir,'panel.js'),'utf8'),readFile(path.join(dir,'styles.css'),'utf8'),readFile(path.join(dir,'preview-mock.js'),'utf8')]);
await mkdir(out,{recursive:true});
const prelude=`/* node-forge 1.4.0. Selected license: BSD-3-Clause.\n${bsd}\n*/\n(function(){'use strict';const module={exports:{}};const exports=module.exports;\n${forge}\nconst forge=module.exports;\n${logic}\n${panel}\nconst css=${JSON.stringify(css)};\nfunction secureRandom(n){if(!globalThis.crypto||typeof globalThis.crypto.getRandomValues!=='function')throw Error('ENCRYPTION_UNAVAILABLE');return globalThis.crypto.getRandomValues(new Uint8Array(n));}\nconst logic=createVPNLogic(forge,secureRandom);\n`;
const real=`${prelude}function start(){let attempts=0;function wait(){if(typeof runShellWithRoot==='function'){mountVPNPanel({logic,bridge:logic.createBridge((command,timeout)=>runShellWithRoot(command,timeout)),css});return;}if(++attempts<60)setTimeout(wait,500);}wait();}if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start,{once:true});else start();\n})();`;
new vm.Script(real,{filename:'U30ProVPN.generated.js'});
const escaped=real.replace(/<\/script/gi,'<\\/script');
const txt=`<!-- [KANO_PLUGIN_START] U30ProVPN -->\n<script>\n${escaped}\n</script>\n<!-- [KANO_PLUGIN_END] U30ProVPN -->\n`;
if(Buffer.byteLength(txt,'utf8')>1145*1024)throw Error('Plugin exceeds UFI file import limit');
await writeFile(path.join(out,'U30ProVPN.txt'),txt,'utf8');
const preview=`${prelude}${mock}\nconst mock=createVPNPreviewTransport(forge);globalThis.u30Preview=mock;globalThis.u30Panel=mountVPNPanel({logic,bridge:logic.createBridge((command,timeout)=>mock.rootShell(command,timeout)),css});\n})();`;
new vm.Script(preview,{filename:'preview-bundle.generated.js'});
await writeFile(path.join(out,'preview-bundle.js'),preview,'utf8');
await writeFile(path.join(out,'preview.html'),`<!doctype html>\n<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>U30 Pro VPN · 设备管理预览</title><style>body{margin:0;padding:28px 18px 40px;background:#E8EEF5;font-family:"Microsoft YaHei UI","PingFang SC",sans-serif}.preview-note{max-width:720px;margin:0 auto 18px;color:#61778E;font-size:12px;line-height:1.6}.preview-note strong{color:#17304B}.container{max-width:760px;margin:auto}@media(max-width:420px){body{padding:12px 10px 25px}}</style></head><body><p class="preview-note"><strong>U30 Pro 设备界面预览</strong>　使用模拟节点与模拟操作，不连接真实设备。订阅测试可输入 https://fail.example/test 验证失败状态。</p><main class="container"></main><script src="preview-bundle.js"></script></body></html>\n`,'utf8');
const manifest={plugin:'U30ProVPN.txt',bytes:Buffer.byteLength(txt,'utf8'),sha256:createHash('sha256').update(txt).digest('hex'),forgeVersion:'1.4.0',forgeSha256:forgeSHA256,externalAssets:false};
await writeFile(path.join(out,'build-manifest.json'),JSON.stringify(manifest,null,2)+'\n');
await writeFile(path.join(out,'FORGE-LICENSE.txt'),bsd+'\n');
console.log(JSON.stringify(manifest));
