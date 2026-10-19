function mountVPNPanel({logic,bridge,css,hostDocument=document}) {
  if(hostDocument.getElementById('u30pro-vpn-panel')) return;
  const host=hostDocument.createElement('div');
  host.id='u30pro-vpn-panel';
  const root=host.attachShadow({mode:'open'});
  root.innerHTML=`<style>${css}</style><article class="panel" aria-label="U30 Pro VPN 管理">
    <header class="header"><div class="brand"><span class="device-icon" aria-hidden="true">${icon('router',25)}</span><div><h2>U30 Pro VPN</h2><p class="subtitle">随身 Wi-Fi 的网络出口</p></div></div><button class="refresh" type="button" aria-label="刷新 VPN 状态">${icon('refresh',16)}<span>刷新</span></button></header>
    <div class="connection"><div class="power-wrap"><button class="power off" type="button" aria-label="开启 VPN" disabled aria-pressed="false">${icon('power',37)}</button><span class="power-label">点击开启 VPN</span></div><div><p class="state-line" aria-live="polite"><span class="state-dot"></span><span class="state-text">正在读取</span></p><p class="state-help">正在连接设备管理服务…</p><div class="route"><span>当前出口</span><strong class="route-name">—</strong></div></div></div>
    <div class="error" role="alert" hidden><span aria-hidden="true">!</span><span class="error-text"></span></div><p class="notice" role="status" hidden></p>
    <div class="tabs" role="tablist" aria-label="管理选项"><button class="tab" type="button" role="tab" id="nodes-tab" aria-controls="nodes-section" aria-selected="true">${icon('nodes',16)}节点<span class="count">0</span></button><button class="tab" type="button" role="tab" id="subscription-tab" aria-controls="subscription-section" aria-selected="false">${icon('subscription',16)}订阅</button></div>
    <section class="section" id="nodes-section" role="tabpanel" aria-labelledby="nodes-tab"><div class="node-top"><h3 class="section-title">选择网络出口</h3><span class="hint">选择后立即生效</span></div><label class="search">${icon('search',17)}<input type="search" placeholder="搜索节点名称" aria-label="搜索节点名称" autocomplete="off"></label><div class="node-list" role="radiogroup" aria-label="VPN 节点"><div class="empty">读取状态后显示节点</div></div></section>
    <section class="section" id="subscription-section" role="tabpanel" aria-labelledby="subscription-tab" hidden><div class="subscription-meta"><div class="meta-row"><span>订阅来源</span><strong class="subscription-host">尚未保存</strong></div><div class="meta-row"><span>上次更新</span><strong class="subscription-time">—</strong></div><div class="meta-row"><span>可用节点</span><strong class="subscription-count">0 个</strong></div></div><label class="field-label" for="subscription-url">订阅链接</label><input class="url-input" id="subscription-url" type="password" autocomplete="new-password" autocapitalize="off" spellcheck="false" placeholder="粘贴 HTTPS 订阅链接" aria-describedby="subscription-hint"><p class="field-hint" id="subscription-hint">链接仅保存在设备中。更新失败时保留原来的节点配置。</p><div class="actions"><button type="button" class="update" disabled>保存并更新</button><button type="button" class="reuse" disabled>更新已保存的订阅</button></div><p class="job-line" aria-live="polite"></p></section>
    <footer class="footer">${icon('check',14)}<span>开关状态和节点选择将在设备重启后保留。</span></footer>
  </article>`;
  const parent=hostDocument.querySelector('.container')||hostDocument.body;
  parent.prepend(host);
  const q=s=>root.querySelector(s);
  let status=null,refreshing=false,pending=false,pollTimer=null,lastJob='',pendingSince=0,destroyed=false;
  const power=q('.power'),refresh=q('.refresh'),search=q('input[type=search]'),url=q('.url-input');
  function text(el,value){el.textContent=value;}
  function showError(message){text(q('.error-text'),message);q('.error').hidden=!message;}
  function showNotice(message){text(q('.notice'),message);q('.notice').hidden=!message;}
  function isBusy(){return pending||!!(status&&status.job&&status.job.status==='running')||!!(status&&['busy','starting'].includes(status.state));}
  function renderControls(){
    const busy=isBusy();
    power.disabled=!status||busy;
    power.classList.toggle('off',!status?.running);
    power.setAttribute('aria-pressed',String(!!status?.desiredEnabled));
    const action=status?.desiredEnabled?'关闭':'开启';
    power.setAttribute('aria-label',action+' VPN');text(q('.power-label'),'点击'+action+' VPN');
    q('.update').disabled=!status||busy||!url.value.trim();
    q('.reuse').disabled=!status||busy||!status.subscription.configured;
    root.querySelectorAll('.node').forEach(b=>b.disabled=!status||busy);
    refresh.disabled=refreshing;
  }
  function renderNodes(){
    const list=q('.node-list');list.replaceChildren();
    if(!status){const empty=hostDocument.createElement('p');empty.className='empty';text(empty,'请登录 UFI 后刷新状态');list.append(empty);return;}
    const needle=search.value.trim().toLocaleLowerCase();
    const items=[{id:'Auto',name:'自动选择',type:'',delay:null},...status.nodes].filter(n=>n.id==='Auto'||!needle||n.name.toLocaleLowerCase().includes(needle));
    for(const n of items){
      const b=hostDocument.createElement('button');b.type='button';b.className='node';b.setAttribute('role','radio');b.setAttribute('aria-checked',String(n.id===status.selectedId));b.disabled=isBusy();
      const radio=hostDocument.createElement('span');radio.className='radio';radio.setAttribute('aria-hidden','true');
      const detail=hostDocument.createElement('span'),name=hostDocument.createElement('span');name.className='node-name';text(name,n.name);detail.append(name);
      const sub=hostDocument.createElement('span');sub.className='node-sub';sub.style.display='block';
      const active=n.id!=='Auto'&&n.id===status.activeId;
      text(sub,n.id==='Auto'?'根据连通性与延迟自动选择节点':(active?'正在使用':String(n.type).toUpperCase()));detail.append(sub);
      const delay=hostDocument.createElement('span');delay.className='node-delay';
      if(n.id==='Auto'){text(delay,status.selectedId==='Auto'?'已选择':'推荐');}
      else if(n.delay!==null&&n.alive!==false){text(delay,n.delay+' ms');}
      else {text(delay,n.alive===false?'暂不可用':'未测速');delay.classList.add('unavailable');}
      b.append(radio,detail,delay);b.addEventListener('click',()=>{if(n.id!==status.selectedId)submit('select',{id:n.id});});
      b.addEventListener('keydown',e=>{if(['ArrowDown','ArrowUp'].includes(e.key)){e.preventDefault();const buttons=Array.from(list.querySelectorAll('button:not(:disabled)'));const i=buttons.indexOf(b);buttons[(i+(e.key==='ArrowDown'?1:buttons.length-1))%buttons.length]?.focus();}});
      list.append(b);
    }
    if(items.length===1&&needle){const empty=hostDocument.createElement('p');empty.className='empty';text(empty,'没有找到匹配的节点');list.append(empty);}
  }
  function formatDate(input){if(!input)return'尚未更新';const date=new Date(input);return Number.isNaN(date.getTime())?'已更新':date.toLocaleString('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit',hour12:false});}
  function render(){
    if(!status)return;
    q('.connection').classList.toggle('running',status.running);
    text(q('.state-text'),pending?'正在处理':logic.STATES[status.state]);
    text(q('.state-help'),status.running?'已连接设备的客户端通过 VPN 联网':status.state==='error'?'VPN 未运行，请查看提示后重试。':status.desiredEnabled?'设备正在恢复 VPN 连接':'设备使用原来的网络转发方式');
    text(q('.route-name'),status.running?logic.displayNode(status):'—');
    text(q('.count'),String(status.nodes.length));
    text(q('.subscription-host'),status.subscription.configured?(status.subscription.host||'已保存订阅'):'尚未保存');
    text(q('.subscription-time'),formatDate(status.subscription.updatedAt));
    text(q('.subscription-count'),status.nodes.length+' 个');
    const j=status.job;
    if(j&&j.status==='running'){q('.job-line').replaceChildren();const spin=hostDocument.createElement('span');spin.className='spinner';q('.job-line').append(spin,hostDocument.createTextNode(j.message||'设备正在处理，请稍候…'));}
    else {text(q('.job-line'),j&&j.status==='succeeded'?'最近一次操作已完成':j&&j.status==='failed'?logic.jobFailureMessage(j):'');}
    renderNodes();renderControls();
  }
  function schedule(ms){clearTimeout(pollTimer);if(!destroyed)pollTimer=setTimeout(()=>read(false),ms);}
  async function read(manual=false){
    if(refreshing)return;
    refreshing=true;renderControls();
    try{
      status=logic.cleanStatus(await bridge.status());
      showError('');
      if(status.job&&status.job.status!=='running'&&status.job.id!==lastJob){
        lastJob=status.job.id;
        if(pending){pending=false;showNotice(status.job.status==='succeeded'?'设置已应用。':'');}
      }
      if(status.job&&status.job.status==='failed')showError(logic.jobFailureMessage(status.job));
      if(pending&&Date.now()-pendingSince>120000){pending=false;showError('设备仍在处理，请稍后点击刷新查看结果。');}
      if(status.message)showNotice(status.message);
      render();schedule(isBusy()?1800:15000);
    }catch(e){
      pending=false;clearTimeout(pollTimer);showError(logic.safeMessage(e.message));
      if(!status){text(q('.state-text'),'等待登录');text(q('.state-help'),'登录 UFI 后，点击右上角刷新。');renderNodes();}
    }finally{refreshing=false;renderControls();}
  }
  async function submit(action,data){
    if(!status||isBusy())return;
    pending=true;pendingSince=Date.now();showError('');showNotice('');renderControls();
    try{
      await bridge.request(status,action,data);
      if(action==='update-subscription')url.value='';
      schedule(250);render();
    }catch(e){pending=false;showError(logic.safeMessage(e.message));renderControls();}
  }
  refresh.addEventListener('click',()=>{showNotice('');read(true);});
  power.addEventListener('click',()=>submit('set-enabled',{enabled:!status.desiredEnabled}));
  search.addEventListener('input',renderNodes);url.addEventListener('input',renderControls);
  q('.update').addEventListener('click',()=>submit('update-subscription',{url:url.value.trim()}));
  q('.reuse').addEventListener('click',()=>submit('update-subscription',{url:''}));
  function selectTab(which){for(const name of['nodes','subscription']){q('#'+name+'-tab').setAttribute('aria-selected',String(name===which));q('#'+name+'-section').hidden=name!==which;}}
  q('#nodes-tab').addEventListener('click',()=>selectTab('nodes'));q('#subscription-tab').addEventListener('click',()=>selectTab('subscription'));
  hostDocument.addEventListener('visibilitychange',()=>{if(hostDocument.hidden)clearTimeout(pollTimer);else if(status)read(false);});
  read(false);
  return {host,root,refresh:()=>read(true),destroy(){destroyed=true;clearTimeout(pollTimer);host.remove();}};
  function icon(name,size){
    const shapes={router:'<rect x="3" y="10" width="18" height="10" rx="3"/><path d="M7 10V5m10 5V5M7 15h.01M11 15h.01M16 15h2M6 4h2m8 0h2"/>',refresh:'<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6.1 7a7 7 0 0 1 11.55-2L20 8M4 16l2.35 3A7 7 0 0 0 17.9 17"/>',power:'<path d="M12 3v8"/><path d="M7 5.7a8 8 0 1 0 10 0"/>',nodes:'<circle cx="5" cy="5" r="2"/><circle cx="19" cy="7" r="2"/><circle cx="10" cy="19" r="2"/><path d="m7 5 10 2M6 7l3 10m8-8-6 8"/>',subscription:'<path d="M9 8H7a4 4 0 0 0 0 8h3m5-8h2a4 4 0 0 1 0 8h-3M8 12h8"/>',search:'<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 5 5"/>',check:'<path d="m5 12 4 4L19 6"/>'};
    return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${shapes[name]||''}</svg>`;
  }
}
