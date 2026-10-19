function createVPNPreviewTransport(forge) {
  'use strict';
  const keys=forge.pki.rsa.generateKeyPair({bits:2048,e:0x10001});
  const state={ok:true,installed:true,desiredEnabled:true,running:true,state:'running',selectedId:'Auto',activeId:'n-2222222222222222',nodes:[
    {id:'n-1111111111111111',name:'香港 · 轻量 01',type:'hysteria2',delay:39,alive:true},
    {id:'n-2222222222222222',name:'日本 · 东京 02',type:'vless',delay:58,alive:true},
    {id:'n-3333333333333333',name:'新加坡 · 标准 01',type:'hysteria2',delay:82,alive:true},
    {id:'n-4444444444444444',name:'美国 · 西海岸 03',type:'vless',delay:143,alive:true},
    {id:'n-5555555555555555',name:'英国 · 伦敦 01',type:'vless',delay:null,alive:false},
    {id:'n-6666666666666666',name:'<img src=x onerror=alert(1)>',type:'vless',delay:null,alive:null}
  ],subscription:{configured:true,updatedAt:new Date().toISOString(),host:'subscription.example',nodeCount:6},job:null,publicKeyPem:forge.pki.publicKeyToPem(keys.publicKey),message:''};
  let count=0;const nonces=new Set();
  function decrypt(arg){
    const padded=arg.replace(/-/g,'+').replace(/_/g,'/')+'='.repeat((4-arg.length%4)%4);
    const env=JSON.parse(forge.util.decodeUtf8(forge.util.decode64(padded)));
    const key=keys.privateKey.decrypt(forge.util.decode64(env.key),'RSA-OAEP',{md:forge.md.sha256.create(),mgf1:{md:forge.md.sha256.create()},label:'u30pro-vpn-web-v1'});
    const joined=forge.util.decode64(env.data),cipher=forge.cipher.createDecipher('AES-GCM',key);
    cipher.start({iv:forge.util.decode64(env.iv),additionalData:'u30pro-vpn-web-v1',tagLength:128,tag:forge.util.createBuffer(joined.slice(-16))});cipher.update(forge.util.createBuffer(joined.slice(0,-16)));
    if(!cipher.finish())throw Error('Authentication failed');
    const request=JSON.parse(forge.util.decodeUtf8(cipher.output.getBytes()));
    if(nonces.has(request.nonce))throw Error('Replay rejected');nonces.add(request.nonce);return request;
  }
  const copy=()=>JSON.parse(JSON.stringify({...state,serverTime:Date.now()}));
  return {state,decrypt,async rootShell(command){
    await new Promise(resolve=>setTimeout(resolve,110));
    if(command==='/data/adb/u30pro-vpn/webctl status')return {success:true,content:JSON.stringify(copy())};
    const m=command.match(/^\/data\/adb\/u30pro-vpn\/webctl request '([A-Za-z0-9_-]+)'$/);
    if(!m)return {success:false,content:'Unsupported command'};
    let r;try{r=decrypt(m[1]);}catch(_){return {success:true,content:JSON.stringify({ok:false,error:{code:'INVALID_INPUT',message:'Invalid encrypted request'}})};}
    const id='preview-job-'+(++count);state.job={id,kind:r.action,status:'running',message:r.action==='update-subscription'?'正在下载、校验并应用订阅…':'正在应用设置…'};
    setTimeout(()=>{
      if(r.action==='set-enabled'){state.desiredEnabled=r.enabled;state.running=r.enabled;state.state=r.enabled?'running':'stopped';}
      else if(r.action==='select'){state.selectedId=r.id;state.activeId=r.id==='Auto'?'n-2222222222222222':r.id;}
      else if(r.action==='update-subscription'){
        if(r.url.includes('fail.example')){state.job={id,kind:r.action,status:'failed',message:'下载未完成，原有配置已保留。'};return;}
        state.subscription.configured=true;state.subscription.updatedAt=new Date().toISOString();if(r.url)state.subscription.host=new URL(r.url).hostname;
      }
      state.job={id,kind:r.action,status:'succeeded',message:'设置已应用。'};
    },1200);
    return {success:true,content:JSON.stringify({ok:true,job:{id,kind:r.action,status:'running'}})};
  }};
}
