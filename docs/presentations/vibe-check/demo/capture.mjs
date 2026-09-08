// Real provider calls through the CLI. Load credentials privately before running.
import fs from 'node:fs';
import path from 'node:path';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {fileURLToPath} from 'node:url';
import {startDemo} from './server.mjs';
const exec=promisify(execFile);
const directory=path.dirname(fileURLToPath(import.meta.url));
const root=path.resolve(directory,'../../../..');
const assets=path.resolve(directory,'../assets');
const output=process.env.MEETUP_CAPTURE_DIR||assets;
fs.mkdirSync(output,{recursive:true});
const model=process.env.VIBIUM_AI_MODEL;
if(!model)throw new Error('Configure VIBIUM_AI_MODEL and its provider credential privately.');
const provider=process.env.VIBIUM_AI_PROVIDER||'openai';
const settings=['--provider',provider,'--model',model,'--base-url',process.env.VIBIUM_AI_BASE_URL||'','--reasoning-effort',process.env.VIBIUM_AI_REASONING_EFFORT||''];
const bin=process.env.VIBIUM_BIN_PATH||path.join(root,'clicker/bin/vibium');
const env={...process.env,VIBIUM_SESSION:`meetup-capture-${process.pid}`,VIBIUM_ENGINE:'firefox',VIBIUM_ENGINE_PATH:'',VIBIUM_ENGINE_CHANNEL:process.env.VIBIUM_FIREFOX_CHANNEL||'beta',VIBIUM_CONNECT_URL:''};
const cli=async(...args)=>{
  let stdout;
  try { ({stdout}=await exec(bin,['--headless','--json',...args],{env,timeout:240000,maxBuffer:16*1024*1024})); }
  catch(error) {
    let message=`Vibium ${args[0]} failed during capture`;
    try { message=JSON.parse(error.stdout).error||message; } catch {}
    for(const [key,value] of Object.entries(env)) {
      if(/KEY|TOKEN|SECRET|PASSWORD/.test(key)&&value)message=String(message).split(value).join('[redacted]');
    }
    throw new Error(String(message));
  }
  const response=JSON.parse(stdout);
  if(!response.ok)throw new Error(`Vibium ${args[0]} returned an error`);
  return response.result;
};
const {server,url}=await startDemo(0);
const manifest={capturedAt:new Date().toISOString(),provider,model,engine:'firefox',description:'Real model calls on a controlled local shop fixture. Broken mode shows Added to cart but drops the cart write.',runs:[]};
try{
  for(const mode of ['broken','fixed']){
    const zip=path.join(output,`${mode}.zip`);
    if(fs.existsSync(zip))throw new Error('Capture output exists; use a new MEETUP_CAPTURE_DIR.');
    await cli('go',url+(mode==='fixed'?'/?fixed=1':''));
    await cli('viewport','1280','800');
    await cli('record','start','--video','--video-size','1280x800','--video-fps','24','--snapshots','--title',`Run + Check: ${mode} cart`,'-o',zip);
    await cli('screenshot','-o',path.join(output,`${mode}-before.png`));
    const goal='Add one battery pack to the cart. Confirm the Added to cart message, then stay on the product page.';
    console.log(`Running ${mode} Run...`);
    const run=await cli('run',goal,...settings);
    await cli('screenshot','-o',path.join(output,`${mode}-added.png`));
    const claim='Does the cart contain exactly one battery pack? Open the cart and inspect its contents without adding or removing anything.';
    console.log(`Running ${mode} Check...`);
    const check=await cli('check',claim,...settings);
    await cli('screenshot','-o',path.join(output,`${mode}-verified.png`));
    await cli('record','stop');
    const names=(await exec('unzip',['-Z1',zip])).stdout.trim().split('\n');
    const video=names.find(n=>n.startsWith('video/')&&n.endsWith('.webm'));
    if(!video)throw new Error('Native WebM missing from the capture.');
    const {stdout:bytes}=await exec('unzip',['-p',zip,video],{encoding:'buffer',maxBuffer:128*1024*1024});
    fs.writeFileSync(path.join(output,`${mode}.webm`),bytes);
    const trace=(await exec('unzip',['-p',zip,'trace.trace'],{maxBuffer:32*1024*1024})).stdout;
    const events=trace.trim().split('\n').map(JSON.parse);
    const parents=events.filter(e=>e.type==='before'&&['vibium:run.run','vibium:check.run'].includes(e.params?.method));
    manifest.runs.push({mode,run,check,parents:parents.map(p=>({method:p.params.method,startTime:p.startTime,endTime:events.find(e=>e.type==='after'&&e.callId===p.callId)?.endTime,modelConfig:p.params.modelConfig,children:events.filter(e=>e.type==='before'&&e.parentId===p.callId).map(e=>e.method)}))});
    console.log(`${mode}: Run ${run.status}; Check ${check.status}`);
    await cli('stop');
  }
  fs.writeFileSync(path.join(output,'demo-results.json'),JSON.stringify(manifest,null,2));
}catch(error){
  console.error(error.message);process.exitCode=1;
}finally{
  await exec(bin,['daemon','stop'],{env,timeout:15000}).catch(()=>{});
  server.closeAllConnections();await new Promise(resolve=>server.close(resolve));
}
