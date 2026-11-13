const {test}=require('node:test');
const assert=require('node:assert/strict');
const {execFile}=require('node:child_process');
const {promisify}=require('node:util');
const {VIBIUM}=require('../helpers');
const {fixture}=require('../run/fixture.cjs');
const exec=promisify(execFile);
test('CLI shorthand and explicit Run use the same model loop; typos never start it',{timeout:180000},async()=>{
 const f=await fixture();
 const env={...process.env,...f.env,VIBIUM_SESSION:`naming-${process.pid}`,VIBIUM_ENGINE:'chrome',VIBIUM_ENGINE_CHANNEL:'',VIBIUM_ENGINE_PATH:'',VIBIUM_CONNECT_URL:''};
 // Browser startup can exceed 30 seconds on the macOS VM. This test checks
 // routing and state, not launch speed; allow startup without weakening assertions.
 const cli=async(...args)=>JSON.parse((await exec(VIBIUM,['--json','--headless',...args],{env,timeout:120000})).stdout).result;
 try{
  // Include the unshipped command names to guard against accidental aliases.
  for(const args of [['staart'],['perform','change name'],['verify','a claim'],['open','the','page'],['--model','model with spaces']]) await assert.rejects(cli(...args));
  assert.equal(f.requests.length,0);assert.match(await cli('stop'),/No browser session/);
  await cli('go',f.url);
  assert.equal((await cli('--model','model with spaces','change name')).status,'completed');
  assert.equal((await cli('run','change name')).status,'completed');
  assert.equal((await cli('check','the name persisted')).status,'passed');
  await cli('eval',"document.body.innerHTML='<input id=box type=checkbox><input id=radio type=radio>';'ready'");
  await cli('set','#box');assert.equal(await cli('is','set','#box'),'true');
  await cli('set','#box');assert.equal(await cli('is','set','#box'),'true');
  await cli('unset','#box');assert.equal(await cli('is','set','#box'),'false');
 }finally{await exec(VIBIUM,['daemon','stop'],{env}).catch(()=>{});await f.close();}
});
