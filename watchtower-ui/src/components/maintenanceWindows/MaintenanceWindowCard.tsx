import { MaintenanceWindow, OneTimeConfig, ManualConfig } from '../../types';
import { Pencil, Trash2 } from 'lucide-react';

interface MaintenanceWindowCardProps {
  window: MaintenanceWindow;
  onToggle: (id: string, isActive: boolean) => void;
  onEdit: (window: MaintenanceWindow) => void;
  onDelete: (window: MaintenanceWindow) => void;
}

function formatDate(ts: string): string {
  return new Date(ts).toLocaleString();
}

export default function MaintenanceWindowCard({ window: w, onToggle, onEdit, onDelete }: MaintenanceWindowCardProps) {
  const isOneTime = w.config.type === 'one_time';
  const isManual = w.config.type === 'manual';

  return (
    <div className={`bg-card-bg rounded-xl border border-border p-5 shadow-sm shadow-black/10 hover:bg-card-hover transition-colors duration-200 ${isManual && !(w.config as ManualConfig).is_active ? 'opacity-50' : ''}`}>
      <div className="flex items-start justify-between mb-3">
        <div>
          <h4 className="font-semibold text-slate-100">{w.title}</h4>
          <span className={`inline-block mt-1.5 px-2 py-0.5 text-[11px] font-medium rounded-md border ${
            isOneTime
              ? 'bg-purple-500/10 text-purple-400 border-purple-500/10'
              : 'bg-orange-500/10 text-orange-400 border-orange-500/10'
          }`}>
            {isOneTime ? 'One-time' : 'Manual'}
          </span>
        </div>
        <div className="flex items-center gap-0.5">
          <button onClick={() => onEdit(w)} className="p-1.5 text-slate-600 hover:text-slate-300 rounded transition-colors">
            <Pencil size={15} />
          </button>
          <button onClick={() => onDelete(w)} className="p-1.5 text-slate-600 hover:text-red-400 rounded transition-colors">
            <Trash2 size={15} />
          </button>
        </div>
      </div>

      {w.description && (
        <p className="text-sm text-slate-400 mb-3">{w.description}</p>
      )}

      {isOneTime && (
        <div className="text-xs text-slate-500 space-y-1 font-mono bg-app-bg rounded-lg p-3">
          <div>Start: {formatDate((w.config as OneTimeConfig).start_time)}</div>
          <div>End: &nbsp;{formatDate((w.config as OneTimeConfig).end_time)}</div>
        </div>
      )}

      {isManual && (
        <div className="flex items-center justify-between mt-3 pt-3 border-t border-border">
          <span className={`text-xs font-medium ${(w.config as ManualConfig).is_active ? 'text-amber-400' : 'text-slate-600'}`}>
            {(w.config as ManualConfig).is_active ? 'Active' : 'Inactive'}
          </span>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={(w.config as ManualConfig).is_active}
              onChange={() => onToggle(w.id, !(w.config as ManualConfig).is_active)}
              className="sr-only peer"
            />
            <div className="w-9 h-5 bg-border rounded-full peer peer-checked:bg-emerald-500 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
          </label>
        </div>
      )}
    </div>
  );
}
