import { useState, FormEvent } from 'react';
import { X } from 'lucide-react';
import { MaintenanceWindow, MaintenanceWindowType } from '../../types';
import MonitorPicker from '../shared/MonitorPicker';
import * as maintenanceWindowsApi from '../../api/maintenanceWindows';

interface MaintenanceWindowFormProps {
  initialData?: MaintenanceWindow;
  initialMonitorIds?: string[];
  onSubmit: (data: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
}

const inputClass = "min-w-0 w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all";
const labelClass = "block text-sm font-medium text-slate-400 mb-1.5";

export default function MaintenanceWindowForm({ initialData, initialMonitorIds, onSubmit, onCancel }: MaintenanceWindowFormProps) {
  const isEdit = !!initialData;
  const [title, setTitle] = useState(initialData?.title || '');
  const [description, setDescription] = useState(initialData?.description || '');
  const [type, setType] = useState<MaintenanceWindowType>(initialData?.config?.type || 'one_time');
  const [startTime, setStartTime] = useState(() => {
    if (initialData?.config?.type === 'one_time') return initialData.config.start_time.slice(0, 16);
    return '';
  });
  const [endTime, setEndTime] = useState(() => {
    if (initialData?.config?.type === 'one_time') return initialData.config.end_time.slice(0, 16);
    return '';
  });
  const [monitorIds, setMonitorIds] = useState<string[]>(initialMonitorIds || []);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      let config: Record<string, unknown>;
      if (type === 'one_time') {
        config = { type: 'one_time', start_time: new Date(startTime).toISOString(), end_time: new Date(endTime).toISOString() };
      } else {
        config = { type: 'manual', is_active: (initialData?.config?.type === 'manual' && initialData.config.is_active) || false };
      }
      // Build submit data — consumer handles creating window + adding monitors
      await onSubmit({ title, description: description || null, config, monitor_ids: monitorIds });
      onCancel();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to save maintenance window');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[3vh]">
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative bg-card-bg rounded-2xl border border-border shadow-2xl shadow-black/40 w-full max-w-lg mx-4 max-h-[94dvh] overflow-y-auto">
        <div className="sticky top-0 bg-card-bg border-b border-border px-6 py-4 flex items-center justify-between rounded-t-xl z-10">
          <h3 className="font-semibold text-slate-100">{isEdit ? 'Edit Maintenance Window' : 'New Maintenance Window'}</h3>
          <button aria-label="Close form" onClick={onCancel} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          <div className="space-y-4">
            <div>
              <label htmlFor="maintenanceWindows-title" className={labelClass}>Title *</label>
              <input id="maintenanceWindows-title" type="text" required value={title} onChange={(e) => setTitle(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor="maintenanceWindows-description" className={labelClass}>Description</label>
              <input id="maintenanceWindows-description" type="text" value={description} onChange={(e) => setDescription(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor="maintenanceWindows-type" className={labelClass}>Type *</label>
              <select id="maintenanceWindows-type" value={type} onChange={(e) => setType(e.target.value as MaintenanceWindowType)} className={inputClass}>
                <option value="one_time">One-time</option>
                <option value="manual">Manual</option>
              </select>
            </div>
            {type === 'one_time' && (
              <>
                <div>
                  <label htmlFor="maintenanceWindows-start-time" className={labelClass}>Start Time *</label>
                  <input id="maintenanceWindows-start-time" type="datetime-local" required value={startTime} onChange={(e) => setStartTime(e.target.value)} className={inputClass} />
                </div>
                <div>
                  <label htmlFor="maintenanceWindows-end-time" className={labelClass}>End Time *</label>
                  <input id="maintenanceWindows-end-time" type="datetime-local" required value={endTime} onChange={(e) => setEndTime(e.target.value)} className={inputClass} />
                </div>
              </>
            )}
          </div>

          {/* Monitor picker */}
          <div className="border-t border-border pt-5">
            <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Monitors</h4>
            <MonitorPicker selectedIds={monitorIds} onChange={setMonitorIds} />
          </div>

          {error && (
            <div className="bg-red-500/5 border border-red-500/20 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
          )}

          <div className="flex justify-end gap-3 pt-2 border-t border-border">
            <button type="button" onClick={onCancel} disabled={submitting} className="px-4 py-2 text-sm font-medium text-slate-300 bg-border rounded-lg hover:bg-border-hover disabled:opacity-50 transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={submitting} className="primary-button">
              {submitting ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Window'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export async function createWindowWithMonitors(data: Record<string, unknown>): Promise<string> {
  const { monitor_ids, ...windowData } = data;
  const res = await maintenanceWindowsApi.createMaintenanceWindow(windowData);
  const windowId = res.data.id;
  const ids = monitor_ids as string[];
  if (ids && ids.length > 0) {
    await Promise.all(ids.map((monitorId) => maintenanceWindowsApi.addMonitorToWindow(windowId, monitorId)));
  }
  return windowId;
}
