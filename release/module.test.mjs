import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, mkdir, readFile, writeFile, cp, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';

const here=path.dirname(fileURLToPath(import.meta.url));
const bash=process.env.U30PRO_TEST_BASH || (process.platform==='win32'?'C:/Program Files/Git/bin/bash.exe':'bash');

async function customize(t, overrides={}) {
  const directory=await mkdtemp(path.join(tmpdir(),'u30pro-module-test-'));
  t.after(()=>rm(directory,{recursive:true,force:true}));
  const module=path.join(directory,'module');
  await mkdir(module);
  await cp(path.join(here,'module/customize.sh'),path.join(module,'customize.sh'));
  const trace=path.join(directory,'trace.txt');
  await writeFile(path.join(module,'u30setup'),'#!/bin/sh\nprintf "helper:%s\\n" "$*" >> "$TRACE_FILE"\n[ "${MOCK_FAIL:-0}" != 1 ] || exit 23\nprintf "{\\"ok\\":true}\\n"\n',{mode:0o700});
  const script='ui_print(){ printf "%s\\n" "$*"; }; abort(){ printf "ABORT:%s\\n" "$*"; exit 41; }; set_perm_recursive(){ printf "permissions\\n" >> "$TRACE_FILE"; }; set_perm(){ :; }; . "$MODPATH/customize.sh"';
  const result=spawnSync(bash,['-c',script],{encoding:'utf8',env:{...process.env,MODPATH:module,TRACE_FILE:trace,BOOTMODE:'true',ARCH:'arm64',API:'35',...overrides}});
  if(result.error) throw result.error;
  const entries=await readFile(trace,'utf8').catch(()=> '');
  return {...result,entries};
}

test('Magisk customize delegates prepare only after permissions and reports staging',async t=>{
  const r=await customize(t);
  assert.equal(r.status,0,r.stderr+r.stdout);
  assert.match(r.entries,/^permissions\nhelper:prepare --module /);
  assert.match(r.stdout,/U30PRO_MODULE_STAGED/);
});
test('Recovery-mode ZIP flashing is refused before helper execution',async t=>{
  const r=await customize(t,{BOOTMODE:'false'});
  assert.equal(r.status,41);
  assert.doesNotMatch(r.entries,/helper:/);
  assert.doesNotMatch(r.stdout,/U30PRO_MODULE_STAGED/);
});
test('Wrong ABI is refused before any helper call',async t=>{
  const r=await customize(t,{ARCH:'x86'});
  assert.equal(r.status,41);
  assert.doesNotMatch(r.entries,/helper:/);
});
test('Failed prepare never produces a success marker',async t=>{
  const r=await customize(t,{MOCK_FAIL:'1'});
  assert.equal(r.status,41);
  assert.match(r.entries,/helper:prepare/);
  assert.doesNotMatch(r.stdout,/U30PRO_MODULE_STAGED/);
});
test('Module hooks delegate boot/uninstall without executing networking commands themselves',async()=>{
  const service=await readFile(path.join(here,'module/service.sh'),'utf8');
  const uninstall=await readFile(path.join(here,'module/uninstall.sh'),'utf8');
  assert.match(service,/u30setup["']?\s+boot\s+--module/);
  assert.match(uninstall,/u30setup["']?\s+uninstall\s+--module/);
  for(const source of[service,uninstall]) assert.doesNotMatch(source,/(?:^|\n)\s*(?:rm\s+-rf|iptables|ip6tables|setenforce|appops)\b/);
  assert.match(uninstall,/deferred/i);
});
test('Customize restores every executable manifest mode before strict preparation',async()=>{
  const source=await readFile(path.join(here,'module/customize.sh'),'utf8');
  for(const name of ['customize.sh','service.sh','action.sh','uninstall.sh','u30setup','payload/webctl','payload/webbridge','payload/control.sh','payload/vpn-service.sh']){
    const directive='set_perm "$MODPATH/'+name+'" 0 0 0700';
    assert.ok(source.includes(directive),name+' must be executable at prepare');
    assert.ok(source.indexOf(directive)<source.indexOf('"$MODPATH/u30setup" prepare'));
  }
});
