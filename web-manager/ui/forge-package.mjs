import {readFile} from 'node:fs/promises';
import {createRequire} from 'node:module';
import {createHash} from 'node:crypto';
import path from 'node:path';

const require = createRequire(process.env.U30PRO_NODE_RUNTIME
  ? path.join(path.resolve(process.env.U30PRO_NODE_RUNTIME), 'package.json')
  : import.meta.url);

export const forgeSHA256 = 'd9b9074e6861200d676e25dcfe97889db5e8f4150bb766d29c491ad709a51b58';

export async function loadPinnedForge() {
  const packagePath = require.resolve('node-forge/package.json');
  const metadata = JSON.parse(await readFile(packagePath, 'utf8'));
  if (metadata.name !== 'node-forge' || metadata.version !== '1.4.0') {
    throw new Error('Expected node-forge 1.4.0; run npm ci.');
  }
  const directory = path.dirname(packagePath);
  const [source, license] = await Promise.all([
    readFile(path.join(directory, 'dist/forge.min.js'), 'utf8'),
    readFile(path.join(directory, 'LICENSE'), 'utf8')
  ]);
  if (createHash('sha256').update(source).digest('hex') !== forgeSHA256) {
    throw new Error('Pinned forge bundle checksum mismatch');
  }
  const start = license.indexOf('New BSD License');
  const end = license.indexOf('        GNU GENERAL PUBLIC LICENSE');
  if (start < 0 || end <= start) throw new Error('Expected upstream Forge BSD license');
  const bsd = license.slice(start, end).replace(/-+\s*$/, '').trim();
  return {source, bsd};
}
