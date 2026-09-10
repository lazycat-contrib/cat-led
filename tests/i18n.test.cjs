const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const publicDir=path.join(__dirname,'../internal/web/public');

function setup(saved, browserLanguage='zh-CN', blocked=false) {
 const storage=new Map(saved ? [['cat-led-language',saved]] : []);
 const events=[];
 const document={documentElement:{},addEventListener(){},querySelectorAll(){return []},dispatchEvent(event){events.push(event)}};
 const window={addEventListener(){}};
 const context={window,document,navigator:{language:browserLanguage},localStorage:{getItem(key){if(blocked)throw Error('blocked');return storage.get(key)},setItem(key,value){if(blocked)throw Error('blocked');storage.set(key,value)}},CustomEvent:class{constructor(type,options){this.type=type;this.detail=options.detail}}};
 vm.createContext(context);
 for(const file of ['i18n-messages.js','i18n.js']) vm.runInContext(fs.readFileSync(path.join(publicDir,'js',file),'utf8'),context);
 return {i18n:window.I18n,messages:window.CAT_LED_MESSAGES,storage,document,events};
}

test('saved language wins, switching persists and notifies without navigation',()=>{
 const {i18n,storage,document,events}=setup('en','zh-CN');
 assert.equal(i18n.language,'en');assert.equal(i18n.t('定时开关机'),'Power schedule');
 i18n.setLanguage('zh');assert.equal(i18n.t('保存设置'),'保存设置');assert.equal(storage.get('cat-led-language'),'zh');assert.equal(document.documentElement.lang,'zh-CN');assert.equal(events.at(-1).type,'languagechange');
 i18n.setLanguage('fr');assert.equal(i18n.language,'zh');
});
test('language switching works without local storage',()=>{
 const {i18n}=setup(null,'en-US',true);assert.equal(i18n.language,'en');i18n.setLanguage('zh');assert.equal(i18n.language,'zh');
});
test('interpolation keeps untrusted parameters as text and preserves template tokens',()=>{
 const {i18n}=setup('en');const name='<img src=x onerror=alert(1)>';
 assert.equal(i18n.t('角色: {0}',{'0':name}),'Role: '+name);
 assert.equal(i18n.t('{{.Name}} 任务执行成功，灯已开启'),'{{.Name}} completed. The light is on.');
 assert.equal(i18n.t('__proto__'),'__proto__');
});
test('API errors and previously rendered errors follow the selected language',()=>{
 const {i18n}=setup('en');
 assert.equal(i18n.error('无法打开 RTC 设备: permission denied'),'Could not open the RTC device: permission denied');
 assert.match(i18n.error('暂时无法核实懒猫管理员身份'),/administrator/);
 assert.equal(i18n.error('测试通知已发送到1个客户端'),'Test notification sent to 1 client.');
 const text=i18n.error('开机时间至少需在两分钟之后');i18n.setLanguage('zh');assert.equal(i18n.error(text),'开机时间至少需在两分钟之后');
});
test('English messages have matching placeholders and no unintended Chinese',()=>{
 const {messages}=setup('en');
 for(const [source,english] of Object.entries(messages)){
  assert.deepEqual(source.match(/\{\d+\}/g)||[],english.match(/\{\d+\}/g)||[],source);
  if(source!=='切换为中文')assert.doesNotMatch(english,/[\u4e00-\u9fff]/,source);
 }
});
test('all literal translation keys and static markup markers have translations',()=>{
 const {messages}=setup('en');
 for(const file of ['index.html','config.html','login.html','js/app.js','js/config.js','js/power.js','js/settings.js','js/about.js']){
  const source=fs.readFileSync(path.join(publicDir,file),'utf8');
  const keys=[...source.matchAll(/I18n\.t\(("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')/g)].map(m=>vm.runInNewContext(m[1]));
  for(const key of keys)assert.ok(Object.hasOwn(messages,key),`${file}: ${key}`);
  if(file.endsWith('.html'))for(const m of source.matchAll(/data-i18n(?:-title|-aria-label|-placeholder)?="([^"]*)"/g)){
   const key=m[1].replaceAll('&quot;','"').replaceAll('&amp;','&').replaceAll('&lt;','<').replaceAll('&gt;','>');
   assert.ok(Object.hasOwn(messages,key),`${file}: ${key}`);
  }
 }
});

test('formatted failure messages retain details through language switches',()=>{
 const {i18n}=setup('en');
 const english=i18n.error('保存任务失败: permission denied');
 assert.equal(english,'Could not save the schedule: permission denied');
 i18n.setLanguage('zh');assert.equal(i18n.error(english),'保存任务失败: permission denied');
 i18n.setLanguage('en');assert.equal(i18n.error('RTC 开机暂不可用：无法打开 RTC 设备: permission denied'),'RTC wake is unavailable: Could not open the RTC device: permission denied');
});
test('dynamic message binding retains keys and parameters without changing data',()=>{
 const {i18n}=setup('en');const attrs=new Map();
 const node={nodeType:1,textContent:'',getAttribute(key){return attrs.has(key)?attrs.get(key):null},setAttribute(key,value){attrs.set(key,value)},hasAttribute(key){return attrs.has(key)},querySelectorAll(){return []}};
 i18n.text(node,'登录失败: {0}',{'0':'<bad request>'});
 assert.equal(node.textContent,'Sign-in failed: <bad request>');
 i18n.setLanguage('zh');i18n.render(node);assert.equal(node.textContent,'登录失败: <bad request>');
});

test('formatting an error twice preserves unrecognized user details',()=>{
 const {i18n}=setup('en');
 const once=i18n.error('保存任务失败: 自定义原因');
 assert.equal(once,'Could not save the schedule: 自定义原因');
 assert.equal(i18n.error(once),once);
 i18n.setLanguage('zh');assert.equal(i18n.error(once),'保存任务失败: 自定义原因');
});
