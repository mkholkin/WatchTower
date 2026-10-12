import http from 'k6/http';
import { check, sleep } from 'k6';
import exec from 'k6/execution';
import { Trend, Gauge } from 'k6/metrics';
import { profile, phaseAt, expectedError } from './model.mjs';

const cfg=profile(__ENV.LAB4_PROFILE || 'ramp', __ENV.LAB4_SMOKE === '1');
const base=__ENV.LAB4_API_URL || 'http://api:8080';
const latency=new Trend('user_request_ms',true);
const journey=new Trend('user_journey_ms',true);
const epoch=new Gauge('scenario_epoch_ms');
const password='Lab4-Disposable-123!';
export const options={
  setupTimeout:'180s',
  scenarios:{users:{executor:'ramping-vus',startVUs:0,gracefulRampDown:'5s',gracefulStop:'10s',
    stages:cfg.phases.flatMap(p=>[{duration:'1s',target:p.vus},{duration:`${p.seconds-1}s`,target:p.vus}])}},
  summaryTrendStats:['min','avg','med','max','p(50)','p(75)','p(90)','p(95)','p(99)'],
  systemTags:['method','status','name','scenario','expected_response'],
};
function payload(label) {
  return {label,endpoint:'http://probe:8081/health',probe_interval:30,alert_contact_ids:[],maintenance_window_ids:[],
    network_config:{protocol:'HTTP',method:'GET',headers:{},body:'',follow_redirects:false},
    expectations:{protocol:'HTTP',expected_status_codes:[200],expected_response_time_ms:5000}};
}
function setupRequest(method,path,body,want,token='') {
  const r=http.request(method,base+path,body===null ? null : JSON.stringify(body),
    {headers:{'Content-Type':'application/json',Authorization:`Bearer ${token}`},timeout:'10s',tags:{name:`setup ${path}`}});
  if(r.status!==want) throw new Error(`setup ${path}: status ${r.status}, want ${want}`);
  return r;
}
export function setup() {
  for(let i=0;i<=cfg.maxVUs;i++) setupRequest('POST','/auth/register',{login:`lab4_${i}`,password},201);
  // Keep one monitor alive so target creation races do not dominate the workload.
  const token=setupRequest('POST','/auth/login',{login:'lab4_0',password},200).json('access_token');
  setupRequest('POST','/monitors',payload('shared target seed'),201,token);
  return {};
}
export default function () {
  epoch.add(exec.scenario.startTime);
  const start=Date.now();let success=true;
  function request(slot,operation,method,path,body,want,token='') {
    const invalid=expectedError(slot,cfg.errorPercent);
    const params={headers:{'Content-Type':'application/json'},timeout:'2s',
      tags:{name:operation},responseCallback:http.expectedStatuses(want)};
    if(token) params.headers.Authorization=`Bearer ${token}`;
    const before=Date.now();
    const r=http.request(method,base+path,body===null ? null : JSON.stringify(body),params);
    let valid=r.status===want;
    if(valid && (operation==='login' || operation==='create')) {
      try { valid=!!r.json(operation==='login' ? 'access_token' : 'id'); } catch (_) {valid=false;}
    }
    latency.add(Date.now()-before,{operation,expected_error:String(invalid),outcome:valid?'ok':'unexpected',
      status:String(r.status),phase:phaseAt(cfg,(Date.now()-exec.scenario.startTime)/1000).name});
    check(r,{'expected status and payload':()=>valid});success=success&&valid;
    return {response:r,valid};
  }
  const login=request(0,'login','POST','/auth/login',{login:`lab4_${__VU}`,password},200);
  if(!login.valid) {sleep(1);journey.add(Date.now()-start,{outcome:'failed'});return;}
  const token=login.response.json('access_token');
  const created=request(1,'create','POST','/monitors',payload(`VU ${__VU} journey ${__ITER}`),201,token);
  if(!created.valid) {sleep(1);journey.add(Date.now()-start,{outcome:'failed'});return;}
  const path='/monitors/'+created.response.json('id');
  request(2,'list','GET','/monitors',null,200,token);
  request(3,'detail','GET',path,null,cfg.errorPercent>=10?401:200,cfg.errorPercent>=10?'':token);
  request(4,'checks','GET',path+'/checks?limit=10',null,200,token);
  request(5,'sla','GET',path+'/sla',null,200,token);
  request(6,'edit','PATCH',path,{label:cfg.errorPercent>=20?42:'Renamed monitor'},cfg.errorPercent>=20?400:200,token);
  request(7,'detail-after-edit','GET',path,null,200,token);
  request(8,'delete','DELETE',path,null,204,token);
  request(9,'list-after-delete','GET','/monitors',null,200,token);
  sleep(0.8+(__VU%5)*0.1);
  journey.add(Date.now()-start,{outcome:success?'ok':'failed'});
}
export function handleSummary(data) {
  return {'/artifacts/k6-summary.json':JSON.stringify({profile:cfg,data},null,2),
    stdout:`\n${cfg.name}: ${data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0} HTTP requests; raw metrics saved\n`};
}
