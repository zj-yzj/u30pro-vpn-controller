import path from 'node:path';
import {fileURLToPath} from 'node:url';

const defaultDirectory = fileURLToPath(new URL('../../build/web-ui/', import.meta.url));

export function outputOptions(args = process.argv.slice(2), allowPort = false) {
  let directory = process.env.U30PRO_UI_OUT || defaultDirectory;
  let port = 18771;
  for (let index = 0; index < args.length; index++) {
    const argument = args[index];
    if (argument === '--out' && args[index + 1]) directory = args[++index];
    else if (argument === '--port' && allowPort && /^(0|[1-9][0-9]{0,4})$/.test(args[index + 1] || '')) {
      port = Number(args[++index]);
      if (port > 65535) throw new Error('Invalid preview port');
    } else throw new Error('Usage: --out <directory>' + (allowPort ? ' [--port <0..65535>]' : ''));
  }
  return {directory: path.resolve(directory), port};
}
