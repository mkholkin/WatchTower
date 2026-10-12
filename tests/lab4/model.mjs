export const PROFILES = ['ramp', 'stress', 'soak-0', 'soak-10', 'soak-20'];
export function profile(name, smoke = false) {
  if (!PROFILES.includes(name)) throw new Error(`Unknown profile: ${name}`);
  let phases = [{name: 'warmup', vus: 2, seconds: smoke ? 3 : 15}];
  if (name === 'ramp') {
    phases = phases.concat((smoke ? [1, 4, 8] : [1, 2, 4, 8, 16, 32, 64, 128]).map(vus => ({name:`vus-${vus}`, vus, seconds:smoke ? 4 : 20})));
  } else if (name === 'stress') {
    phases = phases.concat([{name:'baseline',vus:4,seconds:smoke ? 4 : 30},
      {name:'overload',vus:smoke ? 8 : 128,seconds:smoke ? 5 : 45},
      {name:'recovery',vus:4,seconds:smoke ? 6 : 90}]);
  } else phases.push({name:'steady',vus:4,seconds:smoke ? 10 : 300});
  return {name, smoke, errorPercent:name.startsWith('soak') ? Number(name.split('-')[1]) : 0,
    phases, duration:phases.reduce((n,p) => n+p.seconds,0), maxVUs:Math.max(...phases.map(p=>p.vus))};
}
export function phaseAt(p, seconds) {
  let boundary=0;
  for (const stage of p.phases) { boundary+=stage.seconds; if(seconds < boundary) return stage; }
  return {name:'cooldown',vus:0,seconds:0};
}
export function expectedError(slot, percentage) {
  return (percentage >= 10 && slot === 3) || (percentage >= 20 && slot === 6);
}
