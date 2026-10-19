import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, mkdir, writeFile, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {createManifest, renderPinned, zipFiles} from './package.mjs';

test('Supervisor pins require exactly one placeholder and exact lowercase digests',()=>{
  const pin='a'.repeat(64), cfg='b'.repeat(64);
  assert.equal(renderPinned('@@MIHOMO_SHA256@@/@@CONFIG_SHA256@@',pin,cfg),pin+'/'+cfg);
  for(const source of ['@@MIHOMO_SHA256@@','@@MIHOMO_SHA256@@ @@MIHOMO_SHA256@@ @@CONFIG_SHA256@@'])
    assert.throws(()=>renderPinned(source,pin,cfg));
  assert.throws(()=>renderPinned('@@MIHOMO_SHA256@@ @@CONFIG_SHA256@@','A'.repeat(64),cfg));
});
test('Manifest records exact bytes, POSIX modes and sorted relative paths without itself',async t=>{
  const root=await mkdtemp(path.join(tmpdir(),'u30pro-package-test-'));
  t.after(()=>rm(root,{recursive:true,force:true}));
  await mkdir(path.join(root,'payload'));
  await writeFile(path.join(root,'payload/webctl'),'abc');
  await writeFile(path.join(root,'module.prop'),'id=codex_u30pro_vpn\n');
  const manifest=await createManifest(root);
  assert.equal(manifest.schema,1);
  assert.deepEqual(manifest.files.map(x=>x.path),['module.prop','payload/webctl']);
  assert.equal(manifest.files[1].sha256,'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad');
  assert.equal(manifest.files[1].bytes,3);
  assert.equal(manifest.files[1].mode,0o700);
  assert.equal(manifest.files[0].mode,0o600);
  await writeFile(path.join(root,'manifest.json'),'{}');
  assert.equal((await createManifest(root)).files.length,2);
});
test('Unreviewed files and third-party core bytes cannot enter module manifest',async t=>{
  const root=await mkdtemp(path.join(tmpdir(),'u30pro-package-test-'));
  t.after(()=>rm(root,{recursive:true,force:true}));
  await writeFile(path.join(root,'mihomo'),'secret');
  await assert.rejects(createManifest(root),/Unreviewed/);
});
test('ZIP output is deterministic, contains POSIX execute modes and no traversal',()=>{
  const files=[{path:'u30setup',data:Buffer.from('elf'),mode:0o700}];
  const zip=zipFiles(files);
  assert.deepEqual(zip,zipFiles(files));
  assert.equal(zip.readUInt32LE(0),0x04034b50);
  const central=zip.indexOf(Buffer.from([0x50,0x4b,0x01,0x02]));
  assert.equal(zip.readUInt16LE(central+4)>>8,3);
  assert.equal(zip.readUInt32LE(central+38)>>>16,0o100700);
  assert.throws(()=>zipFiles([{path:'../oops',data:Buffer.alloc(0),mode:0o600}]),/Unsafe/);
  assert.throws(()=>zipFiles([...files,...files]),/Duplicate/);
});
