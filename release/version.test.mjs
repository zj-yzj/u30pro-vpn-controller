import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {version} from './package.mjs';

const root=new URL('../',import.meta.url);
const source=relative=>readFile(new URL(relative,root),'utf8');
function attribute(xml,key){
  const value=new RegExp('android:'+key+'="([^"]+)"').exec(xml);
  assert.ok(value,'Missing Android '+key);return value[1];
}
test('Stable release versions agree across package, backend and Android entry points',async()=>{
  assert.equal(version,'1.3.0');
  for(const file of ['package.json','package-lock.json']){
    const data=JSON.parse(await source(file));assert.equal(data.version,version,file);
    if(data.packages)assert.equal(data.packages[''].version,version,file+' root package');
  }
  for(const file of ['web-manager/internal/manager/types.go','web-manager/internal/installer/manifest.go'])
    assert.equal(/const Version = "([^"]+)"/.exec(await source(file))?.[1],version,file);
  for(const file of ['native-manager/android/AndroidManifest.xml','installer-app/android/AndroidManifest.xml'])
    assert.equal(attribute(await source(file),'versionName'),version,file);
  for(const file of ['release/build.ps1','installer-app/build.ps1','release/verify.ps1'])
    assert.ok((await source(file)).includes("'"+version+"'"),file+' release gate');
});
test('Stable Android and module version codes advance from the prerelease',async()=>{
  const native=Number(attribute(await source('native-manager/android/AndroidManifest.xml'),'versionCode'));
  const installer=Number(attribute(await source('installer-app/android/AndroidManifest.xml'),'versionCode'));
  const props=await source('release/module/module.prop.in');
  assert.ok(native>7,'Native APK must advance beyond alpha code 7');
  assert.ok(installer>130001,'Installer must advance beyond alpha code 130001');
  assert.equal(Number(/^versionCode=(\d+)$/m.exec(props)?.[1]),installer,'Magisk and installer release codes');
  assert.doesNotMatch(props,/\balpha\b|预发布/i);
  assert.doesNotMatch(await source('installer-app/android/src/com/codex/u30vpn/installer/MainActivity.java'),/预发布/);
});
