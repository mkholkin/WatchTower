import { Monitor } from '../../types';
import StatusBadge from '../shared/StatusBadge';

const borderColor: Record<string, string> = {
  up: 'border-l-emerald-500',
  down: 'border-l-red-500',
  maintenance: 'border-l-amber-500',
  unknown: 'border-l-slate-500',
};

interface MonitorCardProps {
  monitor: Monitor;
  onToggle: (id: string, enabled: boolean) => void;
  onClick: () => void;
  index?: number;
}

export default function MonitorCard({ monitor, onToggle, onClick, index = 0 }: MonitorCardProps) {
  const disabled = !monitor.is_enabled;
  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    onToggle(monitor.id, !monitor.is_enabled);
  };

  return (
    <div
      onClick={onClick}
      className={`bg-card-bg rounded-xl border border-border border-l-[3px] ${borderColor[monitor.status]} p-5 cursor-pointer transition-all duration-200 animate-fade-in shadow-sm shadow-black/10 ${
        disabled
          ? 'opacity-40 hover:opacity-50'
          : 'hover:bg-card-hover hover:shadow-md hover:shadow-black/20'
      }`}
      style={{ animationDelay: `${index * 50}ms` }}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="min-w-0 flex-1">
          <h4 className="font-semibold text-slate-100 truncate">{monitor.label}</h4>
          <p className="text-xs text-slate-500 truncate mt-1 font-mono">{monitor.endpoint}</p>
        </div>
        <StatusBadge status={monitor.status} />
      </div>
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-border">
        <span className="text-[11px] text-slate-600 font-mono">
          Every {monitor.probe_interval}s
        </span>
        <label className="relative inline-flex items-center cursor-pointer" onClick={handleToggle}>
          <input
            type="checkbox"
            checked={monitor.is_enabled}
            onChange={() => {}}
            className="sr-only peer"
          />
          <div className="w-9 h-5 bg-border rounded-full peer peer-checked:bg-emerald-500 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
        </label>
      </div>
    </div>
  );
}
