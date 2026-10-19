import http from 'node:http';
import {readFile} from 'node:fs/promises';
import path from 'node:path';
import {outputOptions} from './output-options.mjs';
const {directory:dir,port}=outputOptions(process.argv.slice(2),true);
const files=new Map([['/','preview.html'],['/preview.html','preview.html'],['/preview-bundle.js','preview-bundle.js']]);
const server=http.createServer(async(req,res)=>{const name=files.get(new URL(req.url,'http://127.0.0.1').pathname);if(!name){res.writeHead(404);res.end('Not found');return;}try{const b=await readFile(path.join(dir,name));res.writeHead(200,{'Content-Type':name.endsWith('.js')?'text/javascript; charset=utf-8':'text/html; charset=utf-8','Cache-Control':'no-store','X-Content-Type-Options':'nosniff'});res.end(b);}catch(_){res.writeHead(500);res.end('Build preview first');}});
server.listen(port,'127.0.0.1',()=>console.log('Preview: http://127.0.0.1:'+server.address().port+'/'));
