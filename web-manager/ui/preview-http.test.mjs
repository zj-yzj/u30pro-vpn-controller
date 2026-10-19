import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import {readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {outputOptions} from './output-options.mjs';

const directory=path.dirname(fileURLToPath(import.meta.url));
const {directory:out}=outputOptions();
const child=spawn(process.execPath,[path.join(directory,'serve-preview.mjs'),'--out',out,'--port','0'],{
  stdio:['ignore','pipe','pipe'],windowsHide:true
});
let startup='';
const closed=once(child,'close');
try {
  const base=await new Promise((resolve,reject)=>{
    const timer=setTimeout(()=>reject(new Error('Preview startup timed out')),5000);
    child.stdout.on('data',chunk=>{
      startup+=chunk.toString();
      const match=startup.match(/Preview: (http:\/\/127\.0\.0\.1:[0-9]+\/)/);
      if(match){clearTimeout(timer);resolve(match[1]);}
    });
    child.once('error',error=>{clearTimeout(timer);reject(error);});
    child.once('exit',()=>{clearTimeout(timer);reject(new Error('Preview exited before startup'));});
  });
  const htmlResponse=await fetch(base);
  assert.equal(htmlResponse.status,200);
  assert.match(htmlResponse.headers.get('content-type'),/text\/html/);
  const html=await htmlResponse.text();
  assert.match(html,/使用模拟节点与模拟操作，不连接真实设备/);
  assert.match(html,/<script src="preview-bundle\.js"><\/script>/);
  assert.doesNotMatch(html,/<script[^>]+src=["']https?:/i);
  const scriptResponse=await fetch(new URL('preview-bundle.js',base));
  assert.equal(scriptResponse.status,200);
  assert.match(scriptResponse.headers.get('content-type'),/javascript/);
  const script=await scriptResponse.text();
  assert.match(script,/const mock=createVPNPreviewTransport\(forge\)/);
  const license=(await readFile(path.join(directory,'FORGE-LICENSE.txt'),'utf8')).trim();
  assert(script.includes(license),'Preview retains the complete selected Forge license');
  for(const route of ['api/root_shell','requests.js','crypto.js','U30ProVPN.txt','package.json']){
    assert.equal((await fetch(new URL(route,base))).status,404,route+' is not a preview route');
  }
  console.log(JSON.stringify({previewHTTP:'passed',externalAssetRequests:0,deviceCalls:0,forgeLicense:'retained'}));
} finally {
  child.kill();
  await closed;
}
