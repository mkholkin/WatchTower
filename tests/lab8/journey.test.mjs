import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { profile, phaseAt, expectedError } from '../lab4/model.mjs';

function journey(environment, registration = {domain:'example.com', expires_at:'2027-08-13T04:00:00Z'}) {
  const requests = [], measurements = [];
  class Metric {
    constructor(name) { this.name = name; }
    add(value, tags) { measurements.push({name:this.name, value, tags}); }
  }
  const context = {
    __ENV:{LAB4_PROFILE:'soak-0', ...environment}, __VU:1, __ITER:0,
    profile, phaseAt, expectedError, Trend:Metric, Gauge:Metric,
    exec:{scenario:{startTime:Date.now()}}, sleep() {}, check() {},
    http:{
      expectedStatuses:status => status,
      request(method, url, body, options) {
        requests.push({method, url, body:body ? JSON.parse(body) : null, options});
        let payload = {};
        if(url.endsWith('/auth/login')) payload = {access_token:'test-token'};
        if(method === 'POST' && url.endsWith('/monitors')) payload = {id:'test-monitor'};
        if(url.endsWith('/registration')) payload = registration;
        return {status:method === 'DELETE' ? 204 : (method === 'POST' && url.endsWith('/monitors') ? 201 : 200),
          json:key => payload[key]};
      },
    },
  };
  let source = readFileSync(new URL('../lab4/journey.js', import.meta.url), 'utf8');
  source = source.replace(/^import .*;\n/gm, '').replace('export default function ()', 'function userJourney()')
    .replace(/^export /gm, '');
  vm.runInNewContext(source+'\nuserJourney();', context);
  return {requests, measurements};
}

test('default Lab4 journey retains ten requests and local probe endpoint', () => {
  const {requests} = journey({});
  assert.equal(requests.length, 10);
  assert.equal(requests.find(r => r.method === 'POST' && r.url.endsWith('/monitors')).body.endpoint,
    'http://probe:8081/health');
  assert.equal(requests.some(r => r.url.endsWith('/registration')), false);
});

test('Lab8 opt-in adds one real integration operation before monitor deletion', () => {
  const {requests, measurements} = journey({LAB8_RDAP:'1', LAB4_TARGET_URL:'http://example.com:8081/health'});
  assert.equal(requests.length, 11);
  const registration = requests.findIndex(r => r.url.endsWith('/registration'));
  assert.ok(registration < requests.findIndex(r => r.method === 'DELETE'));
  assert.equal(requests[registration].options.headers.Authorization, 'Bearer test-token');
  assert.equal(requests.find(r => r.method === 'POST' && r.url.endsWith('/monitors')).body.endpoint,
    'http://example.com:8081/health');
  assert.equal(measurements.find(r => r.tags?.operation === 'domain-registration').tags.outcome, 'ok');
});

test('HTTP200 with invalid registration counts as an unexpected load result', () => {
  const {measurements} = journey({LAB8_RDAP:'1'}, {domain:'wrong.com', expires_at:null});
  assert.equal(measurements.find(r => r.tags?.operation === 'domain-registration').tags.outcome, 'unexpected');
  assert.equal(measurements.find(r => r.name === 'user_journey_ms').tags.outcome, 'failed');
});
