import test from 'node:test';
import assert from 'node:assert/strict';
import { profile, phaseAt, expectedError, PROFILES } from './model.mjs';

test('full suite covers ramp, overload/recovery and three 300-second soaks', () => {
  assert.deepEqual(PROFILES, ['ramp', 'stress', 'soak-0', 'soak-10', 'soak-20']);
  for (const name of PROFILES) {
    const p = profile(name);
    assert.equal(p.phases[0].name, 'warmup');
    assert.equal(p.duration, p.phases.reduce((a, x) => a + x.seconds, 0));
    if (name.startsWith('soak')) assert.equal(p.phases[1].seconds, 300);
  }
  assert.ok(profile('ramp').phases.at(-1).vus > profile('ramp').phases[1].vus);
  assert.deepEqual(profile('stress').phases.map(p => p.name), ['warmup','baseline','overload','recovery']);
});
test('exact 10% and 20% erroneous operations in a complete ten-request journey', () => {
  for (const ratio of [0,10,20]) {
    assert.equal(Array.from({length:10}, (_,i) => expectedError(i,ratio)).filter(Boolean).length, ratio/10);
    for (const slot of [0,1,8,9]) assert.equal(expectedError(slot,ratio), false);
  }
});
test('phase boundaries and short smoke profile are explicit', () => {
  const p=profile('stress');
  assert.equal(phaseAt(p, 0).name,'warmup');
  assert.equal(phaseAt(p, 45).name,'overload');
  assert.equal(phaseAt(p, p.duration).name,'cooldown');
  assert.ok(profile('soak-10',true).duration < 30);
  assert.throws(() => profile('unknown'));
});
