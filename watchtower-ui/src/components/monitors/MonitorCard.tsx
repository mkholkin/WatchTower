import { ArrowUpRight, Globe2, Clock3 } from 'lucide-react';
import { Monitor } from '../../types';
import StatusBadge from '../shared/StatusBadge';

interface MonitorCardProps {
  monitor: Monitor;
  onToggle: (id: string, enabled: boolean) => void;
  onClick: () => void;
  index?: number;
}

export default function MonitorCard({ monitor, onToggle, onClick, index = 0 }: MonitorCardProps) {
  return (
    <article className="group min-w-0 overflow-hidden rounded-xl border border-border bg-card-bg transition-colors hover:border-border-hover animate-fade-in" style={{ animationDelay: `${Math.min(index, 8) * 40}ms` }}>
      <button type="button" onClick={onClick} aria-label={`View ${monitor.label}`} className="block w-full rounded-t-xl p-5 text-left transition-colors hover:bg-card-hover focus-visible:outline-offset-[-4px]">
        <span className="mb-5 flex items-center justify-between gap-2">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-app-bg text-slate-400"><Globe2 size={19} /></span>
          <StatusBadge status={monitor.status} />
        </span>
        <span className="flex min-w-0 items-center justify-between gap-3">
          <span className="truncate text-base font-semibold tracking-tight text-slate-100">{monitor.label}</span>
          <ArrowUpRight size={17} className="shrink-0 text-slate-500 transition-colors group-hover:text-emerald-300" />
        </span>
        <span className="mt-2 block truncate font-mono text-xs text-slate-500" title={monitor.endpoint}>{monitor.endpoint}</span>
        <span className="mt-5 inline-flex rounded border border-border px-2 py-1 font-mono text-[10px] text-slate-400">{monitor.network_config.protocol} · {monitor.network_config.method}</span>
      </button>
      <div className="flex items-center justify-between gap-3 border-t border-border px-5 py-3.5">
        <span className="flex items-center gap-1.5 text-xs text-slate-500"><Clock3 size={13} />Every {monitor.probe_interval}s</span>
        <label className="flex cursor-pointer items-center gap-2.5 py-1 text-xs text-slate-400">
          <span>{monitor.is_enabled ? 'Enabled' : 'Disabled'}</span>
          <span className="relative inline-flex">
            <input type="checkbox" aria-label={`Enable monitoring for ${monitor.label}`} checked={monitor.is_enabled} onChange={(e) => onToggle(monitor.id, e.target.checked)} className="peer sr-only" />
            <span className="h-5 w-9 rounded-full bg-border transition-colors peer-checked:bg-emerald-400 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-4 peer-focus-visible:outline-emerald-300 after:absolute after:left-0.5 after:top-0.5 after:h-4 after:w-4 after:rounded-full after:bg-white after:transition-transform after:content-[''] peer-checked:after:translate-x-4" />
          </span>
        </label>
      </div>
    </article>
  );
}
