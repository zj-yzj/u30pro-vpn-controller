// Dependency-free deterministic ZIP writer for this small, explicitly reviewed payload.
import {readdir,readFile,lstat,writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
export const version='1.3.0-alpha.1';
export const moduleId='codex_u30pro_vpn';
const allowed=new Set(['module.prop','customize.sh','service.sh','action.sh','uninstall.sh','u30setup',
  'payload/webctl','payload/webbridge','payload/U30ProVPN.apk','payload/config.yaml',
  'payload/control.sh','payload/vpn-service.sh','payload/U30ProVPN.txt']);
const executable=new Set(['customize.sh','service.sh','action.sh','uninstall.sh','u30setup',
  'payload/webctl','payload/webbridge','payload/control.sh','payload/vpn-service.sh']);
export const sha256=data=>createHash('sha256').update(data).digest('hex');
function safe(name){
  if(!/^[A-Za-z0-9_.\-/]+$/.test(name)||name.startsWith('/')||name.split('/').some(p=>!p||p==='.'||p==='..'))throw Error('Unsafe ZIP path');
}
export function renderPinned(source,core,config){
  if(!/^[a-f0-9]{64}$/.test(core)||!/^[a-f0-9]{64}$/.test(config))throw Error('Invalid digest');
  for(const key of ['MIHOMO_SHA256','CONFIG_SHA256'])
    if(source.split('@@'+key+'@@').length!==2)throw Error('Expected one '+key+' placeholder');
  const result=source.replace('@@MIHOMO_SHA256@@',core).replace('@@CONFIG_SHA256@@',config);
  if(/@@[A-Z_]+@@/.test(result))throw Error('Unrendered placeholder');
  return result;
}
export async function createManifest(root){
  const files=[];
  async function walk(relative=''){
    for(const entry of await readdir(path.join(root,relative),{withFileTypes:true})){
      const name=(relative?relative+'/':'')+entry.name;
      safe(name);
      const item=path.join(root,name),stat=await lstat(item);
      if(stat.isSymbolicLink())throw Error('Symlink is not a payload');
      if(stat.isDirectory()){if(name!=='payload'&&name!=='licenses'&&!name.startsWith('licenses/'))throw Error('Unreviewed directory');await walk(name);continue;}
      if(!stat.isFile())throw Error('Unreviewed file type');
      if(name==='manifest.json')continue;
      if(!allowed.has(name)&&!name.startsWith('licenses/'))throw Error('Unreviewed module file: '+name);
      const data=await readFile(item);
      files.push({path:name,bytes:data.length,sha256:sha256(data),mode:executable.has(name)?0o700:0o600});
    }
  }
  await walk();
  files.sort((a,b)=>a.path.localeCompare(b.path,'en'));
  return {schema:1,version,moduleId,files};
}
function crc32(data){let c=0xffffffff;for(const byte of data){c^=byte;for(let i=0;i<8;i++)c=(c>>>1)^((c&1)?0xedb88320:0);}return(c^0xffffffff)>>>0;}
export function zipFiles(files){
  const local=[],central=[],seen=new Set();let offset=0;
  for(const file of files){
    safe(file.path);if(seen.has(file.path))throw Error('Duplicate ZIP path');seen.add(file.path);
    const name=Buffer.from(file.path),data=Buffer.from(file.data),crc=crc32(data);
    if(data.length>0xffffffff||offset>0xffffffff||name.length>65535)throw Error('ZIP64 unsupported');
    const h=Buffer.alloc(30);h.writeUInt32LE(0x04034b50);h.writeUInt16LE(20,4);h.writeUInt16LE(0x800,6);h.writeUInt16LE(33,12);
    h.writeUInt32LE(crc,14);h.writeUInt32LE(data.length,18);h.writeUInt32LE(data.length,22);h.writeUInt16LE(name.length,26);
    local.push(h,name,data);
    const c=Buffer.alloc(46);c.writeUInt32LE(0x02014b50);c.writeUInt16LE((3<<8)|20,4);c.writeUInt16LE(20,6);c.writeUInt16LE(0x800,8);c.writeUInt16LE(33,14);
    c.writeUInt32LE(crc,16);c.writeUInt32LE(data.length,20);c.writeUInt32LE(data.length,24);c.writeUInt16LE(name.length,28);
    c.writeUInt32LE(((0o100000|file.mode)<<16)>>>0,38);c.writeUInt32LE(offset,42);central.push(c,name);
    offset+=h.length+name.length+data.length;
  }
  if(files.length>65535)throw Error('ZIP64 unsupported');
  const directory=Buffer.concat(central),end=Buffer.alloc(22);end.writeUInt32LE(0x06054b50);end.writeUInt16LE(files.length,8);end.writeUInt16LE(files.length,10);
  end.writeUInt32LE(directory.length,12);end.writeUInt32LE(offset,16);
  return Buffer.concat([...local,directory,end]);
}
export async function packageModule(root,output){
  const manifest=await createManifest(root);
  for(const name of allowed)if(!manifest.files.some(f=>f.path===name))throw Error('Required payload missing: '+name);
  const manifestData=Buffer.from(JSON.stringify(manifest,null,2)+'\n');
  await writeFile(path.join(root,'manifest.json'),manifestData);
  const files=await Promise.all(manifest.files.map(async f=>({...f,data:await readFile(path.join(root,f.path))})));
  files.push({path:'manifest.json',mode:0o600,data:manifestData});
  const zip=zipFiles(files);await writeFile(output,zip);
  return {file:path.basename(output),bytes:zip.length,sha256:sha256(zip)};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
  if(process.argv.length!==4)throw Error('Usage: node release/package.mjs STAGED_MODULE OUTPUT_ZIP');
  console.log(JSON.stringify(await packageModule(process.argv[2],process.argv[3])));
}
