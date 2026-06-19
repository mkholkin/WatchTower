import { MonitorStatus } from '../../types';

const config: Record<MonitorStatus, { label: string; dotClass: string; textClass: string; bgClass: string; pulse: boolean }> = {
  up: {
    label: 'Up',
    dotClass: 'bg-emerald-400',
    textClass: 'text-emerald-400',
    bgClass: 'bg-emerald-500/10',
    pulse: true,
  },
  down: {
    label: 'Down',
    dotClass: 'bg-red-400',
    textClass: 'text-red-400',
    bgClass: 'bg-red-500/10',
    pulse: true,
  },
  maintenance: {
    label: 'Maint.',
    dotClass: 'bg-amber-400',
    textClass: 'text-amber-400',
    bgClass: 'bg-amber-500/10',
    pulse: false,
  },
  unknown: {
    label: 'Unknown',
    dotClass: 'bg-slate-500',
    textClass: 'text-slate-400',
    bgClass: 'bg-slate-500/10',
    pulse: false,
  },
};

export default function StatusBadge({ status, className = '' }: { status: MonitorStatus; className?: string }) {
  const { label, dotClass, textClass, bgClass, pulse } = config[status];
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-semibold ${bgClass} ${textClass} ${className}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${dotClass} ${pulse && status === 'up' ? 'pulse-dot' : ''} ${pulse && status === 'down' ? 'pulse-dot-down' : ''}`} />
      {label}
    </span>
  );
}
