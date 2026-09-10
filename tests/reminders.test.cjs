const {test} = require('node:test');
const assert = require('node:assert/strict');
const {dueEvents, powerEvents} = require('../internal/web/public/js/reminders.js');
const now = Date.parse('2026-09-10T10:00:00Z');
const at = minutes => new Date(now + minutes * 60000).toISOString();

test('lead window includes its boundary and excludes expired or invalid events', () => {
 const sample = {serverTime:now, receivedAt:1000, events:[-1,0,1,5,6].map(n=>({id:n,at:at(n)})).concat({at:'invalid'})};
 assert.deepEqual(dueEvents(sample,5,1000).map(e=>e.id),[1,5]);
 assert.deepEqual(dueEvents(sample,1,1000).map(e=>e.id),[1]);
});
test('monotonic elapsed time advances reminders and stale samples disappear', () => {
 const sample = {serverTime:now, receivedAt:1000, events:[{at:at(2)}]};
 assert.equal(dueEvents(sample,5,62000)[0].minutes,1);
 assert.deepEqual(dueEvents(sample,5,92000),[]);
 assert.deepEqual(dueEvents(null,5,1000),[]);
});
test('only active power actions are announced and waiting wake excludes shutdown', () => {
 const state = {phase:'active',spec:{shutdown_enabled:true,wake_enabled:true},shutdown_at:at(3),wake_at:at(10)};
 assert.deepEqual(powerEvents(state).map(e=>e.operation),['shutdown','wake']);
 assert.deepEqual(powerEvents({...state,phase:'waiting_wake'}).map(e=>e.operation),['wake']);
 for (const phase of ['disabled','completed','error','preparing','shutdown_sent']) assert.deepEqual(powerEvents({...state,phase}),[]);
 assert.deepEqual(powerEvents({...state,spec:{shutdown_enabled:false,wake_enabled:false}}),[]);
 assert.deepEqual(powerEvents({...state,error:'device unavailable'}),[]);
});
