// Local Markdown link/image checks only; this script never requests remote URLs.
import {readdir, readFile, realpath, stat} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const realRoot = await realpath(root);
const findings = [];
let documents = 0;
let links = 0;
const inside = (base, target) => {
  const relative = path.relative(base, target);
  return relative === '' || (!relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative));
};
const normalizeReference = value => value.replace(/\\([\\[\]])/g, '$1').trim().replace(/\s+/g, ' ').toLowerCase();
const mask = value => value.replace(/[^\n]/g, ' ');

// Preserve line positions while ignoring comments, fenced code and inline code.
function proseOnly(source) {
  let fence;
  return source.split('\n').map(line => {
    const marker = line.match(/^ {0,3}(`{3,}|~{3,})/);
    if (fence) {
      if (marker && marker[1][0] === fence[0] && marker[1].length >= fence.length) fence = undefined;
      return mask(line);
    }
    if (marker) { fence = marker[1]; return mask(line); }
    return line;
  }).join('\n').replace(/<!--[\s\S]*?-->/g, mask).replace(/(`+)[^\n]*?\1/g, mask);
}

function destinationAt(source, start) {
  let position = start;
  while (/\s/.test(source[position] ?? '') && position < source.length) position++;
  if (source[position] === '<') {
    const end = source.indexOf('>', position + 1);
    return end < 0 ? null : source.slice(position + 1, end);
  }
  const begin = position;
  let depth = 0;
  while (position < source.length) {
    const character = source[position];
    if (character === '\\' && position + 1 < source.length) { position += 2; continue; }
    if (character === '(') depth++;
    if (character === ')') { if (!depth) break; depth--; }
    if (/\s/.test(character) && !depth) break;
    position++;
  }
  return source.slice(begin, position);
}

async function checkDocument(full, relative) {
  documents++;
  const source = proseOnly(await readFile(full, 'utf8'));
  const references = new Map();
  const lineAt = offset => source.slice(0, offset).split('\n').length;
  const report = (offset, category) => findings.push({file: relative, line: lineAt(offset), category});
  async function checkDestination(raw, offset) {
    links++;
    const target = raw.replace(/\\([\\`*{}\[\]()#+.!_<> -])/g, '$1').replace(/&amp;/g, '&').trim();
    if (!target || target.startsWith('#') || /^https?:\/\//i.test(target)) return;
    if (/^[a-z][a-z\d+.-]*:/i.test(target) || target.startsWith('//')) { report(offset, 'unsupported_link_scheme'); return; }
    let decoded;
    try { decoded = decodeURIComponent(target.split(/[?#]/, 1)[0]); }
    catch { report(offset, 'invalid_link_encoding'); return; }
    if (decoded.includes('\0')) { report(offset, 'invalid_link_path'); return; }
    const resolved = path.resolve(path.dirname(full), decoded.replace(/\\/g, '/'));
    if (!inside(root, resolved)) { report(offset, 'link_outside_repository'); return; }
    try {
      if (!inside(realRoot, await realpath(resolved))) { report(offset, 'link_outside_repository'); return; }
      const metadata = await stat(resolved);
      if (!metadata.isFile() && !metadata.isDirectory()) report(offset, 'link_target_not_file_or_directory');
    } catch { report(offset, 'missing_local_link_target'); }
  }

  // Definitions are checked even when no link currently uses them.
  for (const match of source.matchAll(/^ {0,3}\[([^\]\n]+)\]:[^\S\n]*/gm)) {
    const destination = destinationAt(source, match.index + match[0].length);
    if (destination !== null) {
      references.set(normalizeReference(match[1]), destination);
      await checkDestination(destination, match.index);
    }
  }
  for (const match of source.matchAll(/!?\[([^\]\n]*)\](?:[^\S\n]*\[([^\]\n]*)\])?/g)) {
    const end = match.index + match[0].length;
    if (source[end] === ':') continue;
    if (source[end] === '(') {
      const destination = destinationAt(source, end + 1);
      if (destination === null) report(match.index, 'invalid_markdown_destination');
      else await checkDestination(destination, match.index);
    } else {
      const key = normalizeReference(match[2] || match[1]);
      if (references.has(key)) continue; // Its definition was already checked.
      if (match[2] !== undefined) report(match.index, 'missing_link_reference');
    }
  }
  // README layouts may also use HTML image/link tags for explicit dimensions.
  for (const match of source.matchAll(/<(?:a|img|source)\b[^>]*?\b(?:href|src)\s*=\s*(?:"([^"]*)"|'([^']*)')[^>]*>/gi)) {
    await checkDestination(match[1] ?? match[2], match.index);
  }
}

async function visit(directory, relative = '') {
  for (const entry of await readdir(directory, {withFileTypes: true})) {
    if (entry.name === '.git' || entry.name === 'node_modules' || entry.isSymbolicLink()) continue;
    const rel = [relative, entry.name].filter(Boolean).join('/');
    const full = path.join(directory, entry.name);
    if (entry.isDirectory()) await visit(full, rel);
    else if (entry.isFile() && /\.md$/i.test(entry.name)) await checkDocument(full, rel);
  }
}

await visit(root);
console.log(JSON.stringify({ok: findings.length === 0, documents, links, findings}, null, 2));
if (findings.length) process.exitCode = 1;
