function createVPNLogic(forge, randomBytes) {
  'use strict';
  const LABEL = 'u30pro-vpn-web-v1';
  const STATES = Object.freeze({running:'运行中',stopped:'已关闭',starting:'正在连接',error:'需要处理',busy:'正在处理'});
  const SAFE_MESSAGES = Object.freeze({
    AUTH_REQUIRED:'请先在 UFI 页面登录，再点击刷新。',
    BRIDGE_ERROR:'暂时无法连接管理服务，请确认 UFI 已登录后重试。',
    INVALID_STATUS:'管理服务返回了无法识别的状态，请点击刷新。',
    BUSY:'设备正在处理上一项操作，请稍候。',
    ENCRYPTION_UNAVAILABLE:'当前浏览器无法安全提交操作，请换用较新的手机浏览器。',
    INVALID_INPUT:'输入格式不正确，请检查后重试。',
    root_required:'需要 UFI 的 Root 管理权限，请确认高级功能已开启。',
    not_initialized:'管理组件尚未初始化，请先完成安装。',
    not_configured:'请先添加有效订阅，再开启 VPN。',
    busy:'设备正在处理上一项操作，请稍候。',
    invalid_request:'请求无效或已过期，请刷新状态后重试。',
    replayed_request:'这项请求已经处理，请刷新状态。',
    download_failed:'订阅下载失败，已保留原配置。请检查网络或订阅链接。',
    invalid_subscription:'订阅格式或节点验证失败，已保留原配置。',
    apply_failed:'节点应用未完成，请刷新查看实际运行状态。',
    rollback_failed:'恢复尚未完成，请刷新查看运行状态后再操作。',
    invalid_node:'这个节点已经不存在，请刷新列表后重新选择。',
    control_failed:'VPN 开关操作未完成。目标状态可能已保存，请刷新查看实际状态。',
    storage_failed:'设备保存失败，请查看提示并刷新状态。',
    insecure_storage:'管理目录权限异常，操作已停止。',
    operation_failed:'操作未完成，请查看提示并刷新状态。',
    job_interrupted:'上次操作被中断，请刷新查看恢复结果。'
  });
  const SERVER_JOB_CODES=Object.freeze({
    '需要 UFI Root 权限':'root_required','管理组件尚未初始化':'not_initialized',
    '请先添加有效订阅，再开启 VPN':'not_configured',
    '正在处理上一项操作，请稍后重试':'busy','请求无效或已过期，请刷新页面后重试':'invalid_request',
    '请求已处理，请刷新页面':'replayed_request','订阅下载失败，已保留原配置':'download_failed',
    '订阅格式或节点验证失败，已保留原配置':'invalid_subscription','节点应用失败，已恢复原配置':'apply_failed',
    '恢复未完成，请查看运行状态后重试':'rollback_failed','该节点已不存在，请刷新列表':'invalid_node',
    'VPN 开关操作未完成，请刷新状态':'control_failed','设备保存失败':'storage_failed',
    '管理目录权限异常，操作已停止':'insecure_storage','操作未完成，请刷新状态':'operation_failed',
    '上次操作被中断，已尝试恢复原配置':'job_interrupted'
  });
  function binary(bytes) { return Array.from(bytes, x=>String.fromCharCode(x)).join(''); }
  function random(n) {
    const a = randomBytes(n);
    if (!(a instanceof Uint8Array) || a.length !== n) throw new Error('ENCRYPTION_UNAVAILABLE');
    return binary(a);
  }
  function b64url(s) { return forge.util.encode64(s).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,''); }
  function safeMessage(err) {
    const code = typeof err === 'string' ? err : err && err.code;
    return Object.prototype.hasOwnProperty.call(SAFE_MESSAGES,code)?SAFE_MESSAGES[code]:'操作未完成，请查看提示并刷新状态。';
  }
  function jobFailureMessage(job) {
    if(job && typeof job.code==='string' && Object.prototype.hasOwnProperty.call(SAFE_MESSAGES,job.code))return safeMessage(job.code);
    const message=job && job.message;
    const code=Object.prototype.hasOwnProperty.call(SERVER_JOB_CODES,message)?SERVER_JOB_CODES[message]:'operation_failed';
    return safeMessage(code);
  }
  function unwrap(reply) {
    if (!reply || reply.success !== true || typeof reply.content !== 'string') throw new Error('BRIDGE_ERROR');
    let parsed;
    try { parsed=JSON.parse(reply.content); } catch (_) { throw new Error('INVALID_STATUS'); }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('INVALID_STATUS');
    if (!parsed.ok) {
      const e=new Error(parsed.error && parsed.error.code || 'BRIDGE_ERROR');
      throw e;
    }
    return parsed;
  }
  function cleanStatus(value) {
    if (!value || value.ok !== true || typeof value.running !== 'boolean' || typeof value.desiredEnabled !== 'boolean' || !Array.isArray(value.nodes)) throw new Error('INVALID_STATUS');
    const s={...value};
    s.state=Object.prototype.hasOwnProperty.call(STATES,value.state) ? value.state : (value.running?'running':'stopped');
    s.nodes=value.nodes.filter(n=>n && typeof n.id==='string' && /^n-[a-f0-9]{8,128}$/.test(n.id) && typeof n.name==='string').slice(0,1024).map(n=>({id:n.id,name:n.name.slice(0,256),type:String(n.type||'').slice(0,32),delay:Number.isFinite(n.delay)&&n.delay>=0?Math.round(n.delay):null,alive:typeof n.alive==='boolean'?n.alive:null}));
    s.selectedId=value.selectedId==='Auto'||s.nodes.some(n=>n.id===value.selectedId)?value.selectedId:'Auto';
    s.activeId=typeof value.activeId==='string'?value.activeId:'';
    s.subscription={configured:value.subscription&&value.subscription.configured===true,updatedAt:typeof value.subscription?.updatedAt==='string'?value.subscription.updatedAt:'',host:typeof value.subscription?.host==='string'?value.subscription.host.slice(0,255):'',nodeCount:s.nodes.length};
    s.job=value.job&&typeof value.job==='object'?{id:String(value.job.id||'').slice(0,120),kind:String(value.job.kind||''),status:['running','succeeded','failed'].includes(value.job.status)?value.job.status:'failed',message:typeof value.job.message==='string'?value.job.message.slice(0,300):'',code:typeof value.job.code==='string'?value.job.code:''}:null;
    s.publicKeyPem=typeof value.publicKeyPem==='string'?value.publicKeyPem:'';
    s.serverTime=Number.isFinite(value.serverTime)?value.serverTime:null;
    s.message=typeof value.message==='string'?value.message.slice(0,300):'';
    return s;
  }
  function displayNode(status) { const n=status.nodes.find(n=>n.id===status.activeId)||status.nodes.find(n=>n.id===status.selectedId); return n?n.name:status.selectedId==='Auto'?'自动选择':'等待连接'; }
  function validateAction(action, data) {
    if (action==='set-enabled' && typeof data.enabled==='boolean') return {enabled:data.enabled};
    if (action==='select' && typeof data.id==='string' && (data.id==='Auto'||/^n-[a-f0-9]{8,128}$/.test(data.id))) return {id:data.id};
    if (action==='update-subscription' && typeof data.url==='string' && data.url.length<=8192) {
      if (data.url && !/^https:\/\//i.test(data.url)) throw new Error('INVALID_INPUT');
      return {url:data.url};
    }
    throw new Error('INVALID_INPUT');
  }
  function envelope(publicKeyPem, action, data, now) {
    const fields=validateAction(action,data||{});
    const key=random(32),iv=random(12),nonce=b64url(random(16));
    const plain=forge.util.encodeUtf8(JSON.stringify({v:1,issuedAt:Math.round(now),nonce,action,...fields}));
    const cipher=forge.cipher.createCipher('AES-GCM',key);
    cipher.start({iv,additionalData:LABEL,tagLength:128});
    cipher.update(forge.util.createBuffer(plain));
    if (!cipher.finish()) throw new Error('ENCRYPTION_UNAVAILABLE');
    let pub;
    try {pub=forge.pki.publicKeyFromPem(publicKeyPem);} catch (_) {throw new Error('ENCRYPTION_UNAVAILABLE');}
    const wrapped=pub.encrypt(key,'RSA-OAEP',{md:forge.md.sha256.create(),mgf1:{md:forge.md.sha256.create()},label:LABEL,seed:random(32)});
    const wrappedEnvelope={v:1,key:forge.util.encode64(wrapped),iv:forge.util.encode64(iv),data:forge.util.encode64(cipher.output.getBytes()+cipher.mode.tag.getBytes())};
    return b64url(forge.util.encodeUtf8(JSON.stringify(wrappedEnvelope)));
  }
  function createBridge(rootShell, options={}) {
    let offset=0;
    return {
      async status(){ const raw=unwrap(await rootShell('/data/adb/u30pro-vpn/webctl status',10000)); const s=cleanStatus(raw); if(s.serverTime!==null) offset=s.serverTime-Date.now(); return s; },
      async request(status,action,data){
        const payload=envelope(status.publicKeyPem,action,data,Date.now()+offset);
        if(!/^[A-Za-z0-9_-]+$/.test(payload)) throw new Error('ENCRYPTION_UNAVAILABLE');
        const raw=unwrap(await rootShell("/data/adb/u30pro-vpn/webctl request '"+payload+"'",15000));
        return raw;
      }
    };
  }
  return Object.freeze({STATES,safeMessage,jobFailureMessage,unwrap,cleanStatus,displayNode,validateAction,envelope,createBridge});
}
