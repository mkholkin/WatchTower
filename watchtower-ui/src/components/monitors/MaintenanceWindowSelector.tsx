import { useState, useEffect } from 'react';
import { X, Plus, Trash2 } from 'lucide-react';
import { MaintenanceWindow } from '../../types';
import * as maintenanceWindowsApi from '../../api/maintenanceWindows';
import MaintenanceWindowForm from '../maintenanceWindows/MaintenanceWindowForm';

interface MaintenanceWindowSelectorProps {
  monitorId: string;
  assignedWindows: MaintenanceWindow[];
  onChanged: () => void;
}

export default function MaintenanceWindowSelector({ monitorId, assignedWindows, onChanged }: MaintenanceWindowSelectorProps) {
  const [allWindows, setAllWindows] = useState<MaintenanceWindow[]>([]);
  const [open, setOpen] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    if (open) { maintenanceWindowsApi.getMaintenanceWindows().then((res) => setAllWindows(res.data)); }
  }, [open]);

  const assignedIds = new Set(assignedWindows.map((w) => w.id));
  const available = allWindows.filter((w) => !assignedIds.has(w.id));

  const handleAdd = async (windowId: string) => {
    await maintenanceWindowsApi.addMonitorToWindow(windowId, monitorId);
    onChanged();
  };

  const handleRemove = async (windowId: string) => {
    await maintenanceWindowsApi.removeMonitorFromWindow(windowId, monitorId);
    onChanged();
  };

  const handleCreate = async (data: Record<string, unknown>) => {
    const { monitor_ids, ...windowData } = data;
    // Create window first
    const res = await maintenanceWindowsApi.createMaintenanceWindow(windowData);
    const windowId = res.data.id;
    // Add monitors — ensure current monitor is included
    const ids: string[] = (monitor_ids as string[]) || [];
    if (!ids.includes(monitorId)) ids.push(monitorId);
    await Promise.all(ids.map((id: string) => maintenanceWindowsApi.addMonitorToWindow(windowId, id)));
    setShowCreate(false);
    onChanged();
  };

  return (
    <>
      <button type="button" onClick={() => setOpen(true)} className="text-xs font-medium text-emerald-400 hover:text-emerald-300 transition-colors">+ Add</button>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/70" onClick={() => setOpen(false)} />
          <div className="relative bg-card-bg rounded-xl border border-border shadow-2xl shadow-black/40 w-full max-w-md mx-4 p-6 max-h-[80vh] overflow-y-auto">
            {showCreate ? (
              <>
                <div className="flex items-center justify-between mb-5">
                  <h4 className="font-semibold text-slate-100">New Maintenance Window</h4>
                  <button onClick={() => setShowCreate(false)} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
                </div>
                <MaintenanceWindowForm
                  initialMonitorIds={[monitorId]}
                  onSubmit={handleCreate}
                  onCancel={() => setShowCreate(false)}
                />
              </>
            ) : (
              <>
                <div className="flex items-center justify-between mb-5">
                  <h4 className="font-semibold text-slate-100">Manage Maintenance Windows</h4>
                  <button onClick={() => setOpen(false)} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
                </div>
                {assignedWindows.length > 0 && (
                  <div className="mb-5">
                    <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">Assigned</p>
                    <div className="space-y-1">
                      {assignedWindows.map((w) => (
                        <div key={w.id} className="flex items-center justify-between py-2 px-3 bg-app-bg rounded-lg">
                          <span className="text-sm text-slate-200">{w.title}</span>
                          <button onClick={() => handleRemove(w.id)} className="text-slate-600 hover:text-red-400 transition-colors"><Trash2 size={14} /></button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                <div>
                  <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">Available</p>
                  {available.length === 0 ? (
                    <p className="text-sm text-slate-600">No more windows available.</p>
                  ) : (
                    <div className="space-y-1 max-h-48 overflow-y-auto">
                      {available.map((w) => (
                        <div key={w.id} className="flex items-center justify-between py-2 px-3 hover:bg-app-bg rounded-lg">
                          <span className="text-sm text-slate-300">{w.title}</span>
                          <button onClick={() => handleAdd(w.id)} className="text-emerald-400 hover:text-emerald-300 transition-colors"><Plus size={14} /></button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => setShowCreate(true)}
                  className="mt-4 w-full py-2 border border-dashed border-border rounded-lg text-xs text-slate-500 hover:text-slate-300 hover:border-slate-500 transition-colors"
                >
                  + Create New Window
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </>
  );
}
