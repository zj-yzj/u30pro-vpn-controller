// Report locations and categories only: never print candidate secret values.
import {readdir, readFile, lstat} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {inflateSync} from 'node:zlib';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const findings = [];
let files = 0;
let bytes = 0;
const blockedDirs = /^(?:private|inspection|tools|tools-cache|node_modules|build|dist|\.signing|\.cache|upstream-.+|reader-.+)$/i;
const blockedFiles = /(?:\.(?:p12|pfx|jks|keystore|key|pem|apk|aab|class|dex|exe|dll|so|zip|bin|log)$|^(?:provider\.yaml|state\.json|trust\.json|\.env(?:\..*)?)$)/i;
const generated = new Set(['U30ProVPN.txt', 'preview-bundle.js', 'build-manifest.json']);
const textExtension = /\.(?:go|java|mjs|js|css|html|md|json|sum|mod|ps1|yaml|yml|xml|txt|sh|in)$/i;
const reviewedTextFiles = new Set(['support/b23-backup/expected-sizes.example.csv']);
// Only these two reviewed image formats, directly inside this directory, are allowed.
const documentationAsset = /^docs\/assets\/[^/]+\.(svg|png)$/;
const report = (file, line, category) => findings.push({file, ...(line ? {line} : {}), category});
const exampleHost = hostname => /(?:^|\.)(?:example|test|invalid|localhost)$/.test(hostname)
  || /^(?:.+\.)?example\.(?:com|net|org)$/.test(hostname);

function scanText(rel, source) {
  if (source.includes('\0')) report(rel, 0, 'binary_content');
  const lines = source.split(/\r?\n/);
  lines.forEach((line, index) => {
    if (/[A-Z]:[\\/]Users[\\/][^\\/\s]+/i.test(line)) report(rel, index + 1, 'personal_windows_path');
    if (/BEGIN (?:RSA |EC |OPENSSH |ENCRYPTED )?PRIVATE KEY/.test(line)
        && /^[A-Za-z0-9+/=]{32,}$/.test((lines[index + 1] || '').trim())) {
      report(rel, index + 1, 'private_key_material');
    }
    for (const match of line.matchAll(/https?:\/\/[^\s"'<>`\\]+/g)) {
      let url;
      try { url = new URL(match[0]); } catch { continue; }
      if ((url.username || url.password) && !exampleHost(url.hostname)) {
        report(rel, index + 1, 'url_embedded_credentials');
      }
      if (!exampleHost(url.hostname)
          && [...url.searchParams.keys()].some(key => /^(?:token|access_token|auth|password|passwd|secret|api_key|apikey)$/i.test(key))) {
        report(rel, index + 1, 'non_example_subscription_credential');
      }
    }
  });
}

function checkSvg(rel, source) {
  // Entity declarations are disallowed; decode only XML's built-in/numeric entities.
  const decoded = source.replace(/&(#x[0-9a-f]+|#\d+|amp|lt|gt|quot|apos);/gi, (entity, value) => {
    if (value[0] !== '#') return {amp: '&', lt: '<', gt: '>', quot: '"', apos: "'"}[value.toLowerCase()];
    const code = value[1].toLowerCase() === 'x' ? parseInt(value.slice(2), 16) : parseInt(value.slice(1), 10);
    return code > 0 && code <= 0x10ffff ? String.fromCodePoint(code) : entity;
  });
  scanText(rel, decoded);
  const content = decoded.replace(/^\uFEFF/, '').replace(/<\?xml\s[^?]*\?>/i, '').replace(/<!--[\s\S]*?-->/g, '').trim();
  if (!/^<svg(?:\s|>)/i.test(content) || !/<\/svg\s*>$/i.test(content)) report(rel, 0, 'invalid_svg_document');
  if (/<!\s*(?:DOCTYPE|ENTITY)\b|<\?(?!xml\s)/i.test(decoded)) report(rel, 0, 'svg_external_entity_or_processing_instruction');
  if (/<\s*(?:[\w.-]+:)?(?:script|foreignObject|animate|animateMotion|animateTransform|set)\b/i.test(decoded)) report(rel, 0, 'svg_active_element');
  if (/\s(?:[\w.-]+:)?on[\w.-]+\s*=/i.test(decoded)) report(rel, 0, 'svg_event_handler');
  if (/data\s*:/i.test(decoded)) report(rel, 0, 'svg_embedded_data_url');
  for (const match of decoded.matchAll(/\s((?:[\w.-]+:)?href|xml:base)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/gi)) {
    const value = (match[2] ?? match[3] ?? match[4]).trim();
    if (match[4] !== undefined || (value && (!value.startsWith('#') || match[1].toLowerCase() === 'xml:base'))) {
      report(rel, 0, 'svg_external_reference');
    }
  }
  // CSS may reference local paint servers (#gradient), but cannot load resources.
  if (/@import\b/i.test(decoded)) report(rel, 0, 'svg_external_stylesheet');
  for (const match of decoded.matchAll(/url\s*\(([^)]*)\)/gi)) {
    if (!/^\s*["']?#[\w:.-]+["']?\s*$/.test(match[1])) report(rel, 0, 'svg_external_reference');
  }
  for (const match of decoded.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style\s*>|\sstyle\s*=\s*(?:"([^"]*)"|'([^']*)')/gi)) {
    if (/[\\@]/.test(match[1] ?? match[2] ?? match[3])) report(rel, 0, 'svg_escaped_or_dynamic_stylesheet');
  }
  for (const match of decoded.matchAll(/\s(?:fill|stroke|filter|clip-path|mask|marker(?:-start|-mid|-end)?|cursor)\s*=\s*(?:"([^"]*)"|'([^']*)')/gi)) {
    if (/[\\@]/.test(match[1] ?? match[2])) report(rel, 0, 'svg_escaped_or_dynamic_stylesheet');
  }
}

const pngSignature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
const pngMetadata = new Set(['tEXt', 'zTXt', 'iTXt', 'eXIf']);
const pngAncillaryLengths = new Map([['cHRM', 32], ['gAMA', 4], ['sRGB', 1], ['pHYs', 9]]);
function crc32(buffer) {
  let crc = 0xffffffff;
  for (const byte of buffer) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function checkPng(rel, buffer) {
  const invalid = category => report(rel, 0, category);
  if (buffer.length < 8 || !buffer.subarray(0, 8).equals(pngSignature)) { invalid('invalid_png_signature'); return; }
  let offset = 8;
  let header;
  let ended = false;
  let afterImageData = false;
  let paletteEntries = 0;
  const seen = new Set();
  const imageData = [];
  while (offset < buffer.length) {
    if (buffer.length - offset < 12) { invalid('invalid_png_structure'); return; }
    const length = buffer.readUInt32BE(offset);
    const end = offset + 12 + length;
    if (length > 0x7fffffff || end > buffer.length) { invalid('invalid_png_structure'); return; }
    const type = buffer.toString('latin1', offset + 4, offset + 8);
    const data = buffer.subarray(offset + 8, offset + 8 + length);
    if (!/^[A-Za-z]{2}[A-Z][A-Za-z]$/.test(type)
        || crc32(buffer.subarray(offset + 4, offset + 8 + length)) !== buffer.readUInt32BE(offset + 8 + length)) {
      invalid('invalid_png_chunk_or_checksum'); return;
    }
    if (pngMetadata.has(type)) { invalid('png_text_or_exif_metadata'); return; }
    if (!['IHDR', 'PLTE', 'IDAT', 'IEND', 'tRNS', 'sBIT', 'bKGD'].includes(type) && !pngAncillaryLengths.has(type)) {
      invalid('unreviewed_png_chunk'); return;
    }
    if ((!header && type !== 'IHDR') || (type !== 'IDAT' && seen.has(type))) { invalid('invalid_png_structure'); return; }
    if (type === 'IHDR') {
      if (offset !== 8 || length !== 13) { invalid('invalid_png_structure'); return; }
      const width = data.readUInt32BE(0), height = data.readUInt32BE(4), depth = data[8], color = data[9];
      const validDepths = {0: [1, 2, 4, 8, 16], 2: [8, 16], 3: [1, 2, 4, 8], 4: [8, 16], 6: [8, 16]};
      if (!width || !height || width > 0x7fffffff || height > 0x7fffffff
          || !validDepths[color]?.includes(depth) || data[10] !== 0 || data[11] !== 0 || data[12] > 1) {
        invalid('invalid_png_header'); return;
      }
      header = {width, height, depth, color, interlace: data[12]};
    } else if (type === 'PLTE') {
      paletteEntries = length / 3;
      if (seen.has('IDAT') || !length || length % 3 || paletteEntries > 256 || [0, 4].includes(header.color)
          || (header.color === 3 && paletteEntries > 2 ** header.depth)) { invalid('invalid_png_palette'); return; }
    } else if (type === 'IDAT') {
      if (afterImageData || (header.color === 3 && !paletteEntries)) { invalid('invalid_png_structure'); return; }
      imageData.push(data);
    } else if (type === 'IEND') {
      if (length || !imageData.length || end !== buffer.length) { invalid('invalid_png_structure'); return; }
      ended = true;
    } else {
      if (seen.has('IDAT') || (['cHRM', 'gAMA', 'sRGB', 'sBIT'].includes(type) && seen.has('PLTE'))) {
        invalid('invalid_png_chunk_order'); return;
      }
      if (pngAncillaryLengths.has(type) && length !== pngAncillaryLengths.get(type)) { invalid('invalid_png_structure'); return; }
      if ((type === 'sRGB' && data[0] > 3) || (type === 'pHYs' && data[8] > 1)
          || (type === 'gAMA' && data.readUInt32BE(0) === 0)) { invalid('invalid_png_structure'); return; }
      if (type === 'tRNS' && !((header.color === 0 && length === 2) || (header.color === 2 && length === 6)
          || (header.color === 3 && length > 0 && length <= paletteEntries))) { invalid('invalid_png_structure'); return; }
      const channels = {0: 1, 2: 3, 3: 3, 4: 2, 6: 4}[header.color];
      if (type === 'sBIT' && (length !== channels || [...data].some(n => !n || n > (header.color === 3 ? 8 : header.depth)))) {
        invalid('invalid_png_structure'); return;
      }
      if (type === 'bKGD' && (length !== ({0: 2, 2: 6, 3: 1, 4: 2, 6: 6}[header.color])
          || (header.color === 3 && (!paletteEntries || data[0] >= paletteEntries)))) { invalid('invalid_png_structure'); return; }
    }
    if (seen.has('IDAT') && type !== 'IDAT') afterImageData = true;
    seen.add(type);
    offset = end;
  }
  if (!ended) { invalid('invalid_png_structure'); return; }
  // Validate the compressed scanline stream too; pixel values never become text.
  const channels = {0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[header.color];
  const passes = header.interlace ? [[0, 0, 8, 8], [4, 0, 8, 8], [0, 4, 4, 8], [2, 0, 4, 4], [0, 2, 2, 4], [1, 0, 2, 2], [0, 1, 1, 2]] : [[0, 0, 1, 1]];
  const rows = passes.map(([x, y, dx, dy]) => {
    const width = Math.max(0, Math.ceil((header.width - x) / dx));
    return {count: width ? Math.max(0, Math.ceil((header.height - y) / dy)) : 0, bytes: 1 + Math.ceil(width * channels * header.depth / 8)};
  });
  const expected = rows.reduce((sum, row) => sum + row.count * row.bytes, 0);
  if (expected > 128 * 1024 * 1024) { invalid('png_decoded_size_limit'); return; }
  try {
    const compressed = Buffer.concat(imageData);
    const inflated = inflateSync(compressed, {maxOutputLength: expected + 1, info: true});
    const pixels = inflated.buffer;
    if (pixels.length !== expected || inflated.engine.bytesWritten !== compressed.length) { invalid('invalid_png_image_data'); return; }
    let position = 0;
    for (const row of rows) for (let index = 0; index < row.count; index++, position += row.bytes) {
      if (pixels[position] > 4) { invalid('invalid_png_image_data'); return; }
    }
  } catch { invalid('invalid_png_image_data'); }
}

async function visit(directory, relative = '') {
  for (const entry of await readdir(directory, {withFileTypes: true})) {
    if (entry.name === '.git') continue; // Git metadata is not publication content.
    const rel = [relative, entry.name].filter(Boolean).join('/');
    const full = path.join(directory, entry.name);
    if (entry.isSymbolicLink()) { report(rel, 0, 'symbolic_link'); continue; }
    if (entry.isDirectory()) {
      if (blockedDirs.test(entry.name)) report(rel, 0, 'excluded_directory');
      else await visit(full, rel);
      continue;
    }
    const stat = await lstat(full);
    files++;
    bytes += stat.size;
    if (!stat.isFile()) { report(rel, 0, 'non_regular_file'); continue; }
    if (blockedFiles.test(entry.name) || generated.has(entry.name)) report(rel, 0, 'sensitive_or_generated_file');
    const asset = rel.match(documentationAsset);
    if (asset) {
      const buffer = await readFile(full);
      if (asset[1] === 'png') checkPng(rel, buffer);
      else {
        try { checkSvg(rel, new TextDecoder('utf-8', {fatal: true}).decode(buffer)); }
        catch { report(rel, 0, 'invalid_svg_encoding'); }
      }
      continue;
    }
    if (!textExtension.test(entry.name) && !reviewedTextFiles.has(rel) && !/^(?:LICENSE|NOTICE|\.gitignore|\.gitattributes)$/.test(entry.name)) {
      report(rel, 0, 'unreviewed_file_type'); continue;
    }
    scanText(rel, await readFile(full, 'utf8'));
  }
}

await visit(root);
console.log(JSON.stringify({ok: findings.length === 0, files, bytes, findings}, null, 2));
if (findings.length) process.exitCode = 1;
