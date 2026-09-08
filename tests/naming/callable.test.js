const {test}=require('node:test');
const assert=require('node:assert/strict');
const {Browser,Page}=require('../../clients/javascript/dist');
const {BrowserSync,PageSync}=require('../../clients/javascript/dist/sync');
for(const [label,B,P,sync] of [['async',Browser,Page,false],['sync',BrowserSync,PageSync,true]]) {
 test(`${label} callable session objects preserve identity, bindings and options`,async()=>{
  const calls=[];
  const transport={onEvent(){},send(method,params){calls.push({method,params});return Promise.resolve({status:'completed',goal:params.goal});},call(method,args){calls.push({method,args});return {status:'completed',goal:args.at(-2)};}};
  const browser=new B(transport,null), page=new P(transport,sync?7:'page-7');
  assert.equal(typeof page.waitUntil, 'function');
  assert.equal(typeof page.waitUntil.url, 'function');
  assert.equal(typeof page.waitUntil.loaded, 'function');
  const options={provider:'local',model:'fixture',reasoningEffort:''};
  for(const object of [browser,page]){
   assert.equal(typeof object,'function');assert.ok(object instanceof (object===browser?B:P));
   assert.strictEqual(await Promise.resolve(object),object,'not accidentally thenable');
   await object('a goal',options);const shorthand=calls.at(-1);
   assert.equal(shorthand.method, sync ? (object===browser?'browser.run':'page.run') : 'vibium:run.run');
   if (!sync) { assert.equal(shorthand.params.goal, 'a goal'); assert.equal(shorthand.params.claim, undefined); }
   const run=object.run;await run('a goal',options);assert.deepEqual(calls.at(-1),shorthand,'detached method has same binding');
   await object.check('a claim',options);const check=calls.at(-1);
   assert.equal(check.method, sync ? (object===browser?'browser.check':'page.check') : 'vibium:check.run');
   if (!sync) { assert.equal(check.params.claim, 'a claim'); assert.equal(check.params.goal, undefined); }
   assert.equal(object.set, undefined, 'session objects assess claims; checkbox setters belong to elements');
   // Unshipped names must not remain as public aliases.
   assert.equal(object.perform, undefined); assert.equal(object.verify, undefined);
  }
  assert.strictEqual(await page.mainFrame(),page);
  if (!sync) {
    const element = page.find('#fixture');
    assert.equal(typeof element.click, 'function');
    assert.equal(typeof element.set, 'function');
    await element;
  }
  if(!sync){assert.equal(page.id,'page-7');assert.equal(calls.at(-1).params.context,'page-7');}
 });
}
