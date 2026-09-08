import http from 'node:http';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

export function startDemo(port=4177) {
  const html=fs.readFileSync(new URL('./account.html',import.meta.url));
  const server=http.createServer((req,res)=>{res.setHeader('Content-Type','text/html; charset=utf-8');res.end(html);});
  return new Promise((resolve,reject)=>{
    server.on('error',reject);
    server.listen(port,'127.0.0.1',()=>resolve({server,url:`http://127.0.0.1:${server.address().port}`}));
  });
}
if(process.argv[1]===fileURLToPath(import.meta.url)) {
  const {url}=await startDemo(Number(process.env.PORT||4177));
  console.log(`Account demo: ${url} (intentional persistence bug)`);
  console.log(`Fixed version: ${url}/?persist=1`);
}
