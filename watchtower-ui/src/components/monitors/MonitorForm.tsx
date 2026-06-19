import { useState, FormEvent } from 'react';
import { X, Plus, Trash2 } from 'lucide-react';
import { Monitor, HTTPMethod, MonitorProtocol } from '../../types';

const PROBE_INTERVALS = [
  { label: '5 seconds', value: 5 },
  { label: '1 minute', value: 60 },
  { label: '5 minutes', value: 300 },
  { label: '15 minutes', value: 900 },
  { label: '30 minutes', value: 1800 },
  { label: '1 hour', value: 3600 },
];

const HTTP_METHODS: HTTPMethod[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

type HeaderEntry = { key: string; value: string };

interface MonitorFormProps {
  initialData?: Monitor;
  onSubmit: (data: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
}

function buildHeaders(entries: HeaderEntry[]): Record<string, string> | null {
  const filled = entries.filter((h) => h.key.trim());
  return filled.length > 0
    ? Object.fromEntries(filled.map((h) => [h.key.trim(), h.value]))
    : null;
}

const inputClass = "w-full px-3 py-2 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all";
const labelClass = "block text-sm font-medium text-slate-400 mb-1.5";
const sectionClass = "bg-app-bg/50 rounded-xl border border-border p-5 space-y-4";

export default function MonitorForm({ initialData, onSubmit, onCancel }: MonitorFormProps) {
  const isEdit = !!initialData;

  const [protocol, setProtocol] = useState<MonitorProtocol>(
    initialData?.network_config?.protocol || 'HTTP'
  );
  const [label, setLabel] = useState(initialData?.label || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [endpoint, setEndpoint] = useState(initialData?.endpoint || '');
  const [probeInterval, setProbeInterval] = useState(initialData?.probe_interval || 60);

  // HTTP config
  const [method, setMethod] = useState<HTTPMethod>(initialData?.network_config?.method || 'GET');
  const [headers, setHeaders] = useState<HeaderEntry[]>(() => {
    if (initialData?.network_config?.headers) {
      return Object.entries(initialData.network_config.headers).map(([k, v]) => ({ key: k, value: v }));
    }
    return [{ key: '', value: '' }];
  });
  const [body, setBody] = useState(initialData?.network_config?.body || '');
  const [followRedirects, setFollowRedirects] = useState(
    initialData?.network_config?.follow_redirects ?? true
  );

  // HTTP expectations
  const [expectedStatusCodes, setExpectedStatusCodes] = useState(
    initialData?.expectations?.expected_status_codes?.join(', ') || '200'
  );
  const [expectedResponseTime, setExpectedResponseTime] = useState(
    initialData?.expectations?.expected_response_time_ms || 2000
  );

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const addHeader = () => setHeaders([...headers, { key: '', value: '' }]);
  const removeHeader = (idx: number) => setHeaders(headers.filter((_, i) => i !== idx));
  const updateHeader = (idx: number, field: 'key' | 'value', val: string) => {
    const next = [...headers];
    next[idx] = { ...next[idx], [field]: val };
    setHeaders(next);
  };

  const parseStatusCodes = (s: string): number[] =>
    s.split(',').map((c) => parseInt(c.trim(), 10)).filter((n) => !isNaN(n));

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      const data: Record<string, unknown> = {
        label,
        description: description || null,
        endpoint,
        probe_interval: probeInterval,
        protocol,
        network_config: {
          protocol,
          method,
          headers: buildHeaders(headers),
          body: method === 'POST' || method === 'PUT' ? body || null : null,
          follow_redirects: followRedirects,
        },
        expectations: {
          protocol,
          expected_status_codes: parseStatusCodes(expectedStatusCodes),
          expected_response_time_ms: expectedResponseTime,
        },
      };
      await onSubmit(data);
      onCancel();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to save monitor');
    } finally {
      setSubmitting(false);
    }
  };

  const showBodyField = method === 'POST' || method === 'PUT';

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[3vh]">
      <div className="absolute inset-0 bg-black/70" onClick={onCancel} />
      <div className="relative bg-card-bg rounded-xl border border-border shadow-2xl shadow-black/40 w-full max-w-2xl mx-4 max-h-[94vh] overflow-y-auto">
        <div className="sticky top-0 bg-card-bg border-b border-border px-6 py-4 flex items-center justify-between rounded-t-xl z-10">
          <h3 className="text-lg font-semibold text-slate-100">{isEdit ? 'Edit Monitor' : 'New Monitor'}</h3>
          <button onClick={onCancel} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          {/* Basic info */}
          <div className={sectionClass}>
            <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Basic Information</h4>
            <div className="grid grid-cols-2 gap-4">
              <div className="col-span-2 sm:col-span-1">
                <label className={labelClass}>Label *</label>
                <input type="text" required value={label} onChange={(e) => setLabel(e.target.value)} className={inputClass} />
              </div>
              <div className="col-span-2 sm:col-span-1">
                <label className={labelClass}>Protocol</label>
                <select value={protocol} onChange={(e) => setProtocol(e.target.value as MonitorProtocol)} className={inputClass}>
                  <option value="HTTP">HTTP</option>
                </select>
              </div>
              <div className="col-span-2">
                <label className={labelClass}>Description</label>
                <input type="text" value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />
              </div>
              <div className="col-span-2">
                <label className={labelClass}>Endpoint URL *</label>
                <input type="url" required value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder="https://example.com/health" className={inputClass} />
              </div>
              <div className="col-span-2 sm:col-span-1">
                <label className={labelClass}>Probe Interval *</label>
                <select value={probeInterval} onChange={(e) => setProbeInterval(Number(e.target.value))} className={inputClass}>
                  {PROBE_INTERVALS.map((p) => (<option key={p.value} value={p.value}>{p.label}</option>))}
                </select>
              </div>
            </div>
          </div>

          {/* Network Configuration */}
          {protocol === 'HTTP' && (
            <div className={sectionClass}>
              <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Network Configuration — HTTP</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Method</label>
                  <select value={method} onChange={(e) => setMethod(e.target.value as HTTPMethod)} className={inputClass}>
                    {HTTP_METHODS.map((m) => (<option key={m} value={m}>{m}</option>))}
                  </select>
                </div>
                <div className="flex items-end pb-1">
                  <label className="flex items-center gap-2 text-sm text-slate-400 cursor-pointer select-none">
                    <input type="checkbox" checked={followRedirects} onChange={(e) => setFollowRedirects(e.target.checked)} className="rounded border-border bg-app-bg accent-emerald-500" />
                    Follow redirects
                  </label>
                </div>
              </div>
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm font-medium text-slate-400">Headers</label>
                  <button type="button" onClick={addHeader} className="inline-flex items-center gap-1 text-xs text-emerald-400 hover:text-emerald-300 transition-colors">
                    <Plus size={14} /> Add
                  </button>
                </div>
                <div className="space-y-2">
                  {headers.map((h, i) => (
                    <div key={i} className="flex gap-2">
                      <input type="text" value={h.key} onChange={(e) => updateHeader(i, 'key', e.target.value)} placeholder="Header name" className={`flex-1 ${inputClass}`} />
                      <input type="text" value={h.value} onChange={(e) => updateHeader(i, 'value', e.target.value)} placeholder="Value" className={`flex-1 ${inputClass}`} />
                      <button type="button" onClick={() => removeHeader(i)} className="p-1.5 text-slate-600 hover:text-red-400 transition-colors"><Trash2 size={16} /></button>
                    </div>
                  ))}
                </div>
              </div>
              {showBodyField && (
                <div>
                  <label className={labelClass}>Request Body</label>
                  <textarea value={body} onChange={(e) => setBody(e.target.value)} rows={3} className={`${inputClass} font-mono resize-vertical`} />
                </div>
              )}
            </div>
          )}

          {/* Success Criteria (Expectations) */}
          {protocol === 'HTTP' && (
            <div className={sectionClass}>
              <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Success Criteria — HTTP</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Expected Status Codes</label>
                  <input type="text" value={expectedStatusCodes} onChange={(e) => setExpectedStatusCodes(e.target.value)} placeholder="200, 201" className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>Max Response Time (ms)</label>
                  <input type="number" value={expectedResponseTime} onChange={(e) => setExpectedResponseTime(Number(e.target.value))} className={inputClass} />
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="bg-red-500/5 border border-red-500/20 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
          )}

          <div className="flex justify-end gap-3 pt-2 border-t border-border">
            <button type="button" onClick={onCancel} disabled={submitting} className="px-4 py-2 text-sm font-medium text-slate-300 bg-border rounded-lg hover:bg-border-hover disabled:opacity-50 transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={submitting} className="px-4 py-2 text-sm font-semibold text-slate-900 bg-emerald-500 rounded-lg hover:bg-emerald-400 disabled:opacity-50 transition-all">
              {submitting ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Monitor'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
