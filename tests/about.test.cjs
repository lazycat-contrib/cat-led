const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

function setup(fetchResult) {
 const timers = new Map();
 const nodes = new Map();
 const document = {getElementById(id) {
  if (!nodes.has(id)) nodes.set(id, {dataset:{}, attributes:{'aria-pressed':'false'}, listeners:{},
   addEventListener(type, callback) {this.listeners[type] = callback;},
   setAttribute(key, value) {this.attributes[key] = value;}, getAttribute(key) {return this.attributes[key];},
   showModal() {this.open = true;}, focus() {},
  });
  return nodes.get(id);
 }};
 const requests = [];
 const context = {document, AbortController, Number,
  setTimeout(callback, delay) {const id = {}; timers.set(id, {callback, delay}); return id;},
  clearTimeout(id) {timers.delete(id);},
  fetch: async (url, options) => {requests.push({url, options}); return typeof fetchResult === 'function' ? fetchResult() : fetchResult;},
  I18n: {text(node, key, params = {}) {node.textContent = key.replace(/\{(\d+)\}/g, (_, k) => params[k]);}},
 };
 vm.runInNewContext(fs.readFileSync('internal/web/public/js/about.js', 'utf8'), context);
 const open = async () => {nodes.get('about-button').listeners.click({detail:1}); await new Promise(setImmediate);};
 const close = () => {const dialog = nodes.get('about-dialog'); dialog.open = false; dialog.listeners.close();};
 return {nodes, timers, requests, open, close};
}

test('uptime is human readable at minute, hour and day boundaries', async () => {
 for (const [seconds, expected] of [[0,'不足1分钟'],[59,'不足1分钟'],[60,'1分'],[3599,'59分'],[3600,'1小时0分'],[3720,'1小时2分'],[86400,'1天0小时0分'],[183480,'2天2小时58分']]) {
  const app = setup({ok:true, json:async () => ({uptime_seconds:seconds})});
  await app.open();
  assert.equal(app.nodes.get('about-uptime').textContent, expected);
  assert.equal(app.requests[0].url, '/api/system/uptime');
  assert.equal(app.requests[0].options.cache, 'no-store');
  app.close(); assert.equal(app.timers.size, 0);
 }
});
test('failed or malformed uptime does not display made-up duration', async () => {
 for (const response of [{ok:false}, ...[-1,NaN,Infinity,'3720',null].map(value => ({ok:true,json:async()=>({uptime_seconds:value})}))]) {
  const app = setup(response); await app.open();
  assert.equal(app.nodes.get('about-uptime').textContent, '暂时无法读取');
  app.close();
 }
});
test('closing aborts outstanding requests and prevents stale rendering or polling', async () => {
 let resolve;
 const app = setup(() => new Promise(done => {resolve = done;}));
 await app.open(); app.close();
 assert.equal(app.requests[0].options.signal.aborted, true);
 resolve({ok:true,json:async()=>({uptime_seconds:3720})});
 await new Promise(setImmediate);
 assert.equal(app.nodes.get('about-uptime').textContent, '加载中…');
 assert.equal(app.timers.size, 0);
});
test('the dialog defaults to the black cat, switches on tap and resets on reopening', async () => {
 const app = setup({ok:true,json:async()=>({uptime_seconds:60})});
 await app.open(); const cat = app.nodes.get('about-cat');
 assert.equal(cat.getAttribute('aria-pressed'), 'false');
 cat.listeners.click(); assert.equal(cat.getAttribute('aria-pressed'), 'true');
 app.close(); await app.open(); assert.equal(cat.getAttribute('aria-pressed'), 'false'); app.close();
});
