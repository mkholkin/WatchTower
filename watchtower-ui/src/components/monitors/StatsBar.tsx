import { Monitor } from '../../types';
import { Activity, CheckCircle2, XCircle, PauseCircle, HelpCircle } from 'lucide-react';

interface StatsBarProps {
  monitors: Monitor[];
}

export default function StatsBar({ monitors }: StatsBarProps) {
  const total = monitors.length;
  const up = monitors.filter((m) => m.status === 'up').length;
  const down = monitors.filter((m) => m.status === 'down').length;
  const maintenance = monitors.filter((m) => m.status === 'maintenance').length;
  const unknown = monitors.filter((m) => m.status === 'unknown').length;

  const items = [
    { label: 'Total', value: total, icon: Activity, color: 'text-slate-400', bg: 'bg-slate-500/10' },
    { label: 'Up', value: up, icon: CheckCircle2, color: 'text-emerald-400', bg: 'bg-emerald-500/10' },
    { label: 'Down', value: down, icon: XCircle, color: 'text-red-400', bg: 'bg-red-500/10' },
    { label: 'Maint.', value: maintenance, icon: PauseCircle, color: 'text-amber-400', bg: 'bg-amber-500/10' },
    { label: 'Unknown', value: unknown, icon: HelpCircle, color: 'text-slate-400', bg: 'bg-slate-500/10' },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-6">
      {items.map(({ label, value, icon: Icon, color, bg }) => (
        <div key={label} className="bg-card-bg rounded-xl border border-border p-4 shadow-sm shadow-black/10">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[10px] font-medium text-slate-500 uppercase tracking-wider">{label}</span>
            <div className={`w-7 h-7 rounded-lg ${bg} flex items-center justify-center`}>
              <Icon size={14} className={color} />
            </div>
          </div>
          <p className={`text-2xl font-bold font-mono ${color}`}>{value}</p>
        </div>
      ))}
    </div>
  );
}
