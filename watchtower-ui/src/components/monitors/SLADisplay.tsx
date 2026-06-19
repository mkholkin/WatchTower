import { MonitorSLA } from '../../types';
import { LoadingSpinner } from '../shared/LoadingSpinner';

interface SLADisplayProps {
  sla: MonitorSLA | null;
  loading: boolean;
}

function formatDuration(totalSec: number): string {
  if (totalSec === 0) return '0s';
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  const parts = [];
  if (h) parts.push(`${h}h`);
  if (m) parts.push(`${m}m`);
  if (s) parts.push(`${s}s`);
  return parts.join(' ');
}

function pctColor(pct: number): string {
  if (pct >= 99.9) return 'text-emerald-400';
  if (pct >= 99) return 'text-amber-400';
  return 'text-red-400';
}

export default function SLADisplay({ sla, loading }: SLADisplayProps) {
  if (loading) {
    return <div className="py-8"><LoadingSpinner /></div>;
  }

  if (!sla) {
    return <p className="text-sm text-slate-500 py-4">No SLA data available.</p>;
  }

  return (
    <div className="text-center py-4">
      <div className={`text-5xl font-bold font-mono tracking-tight ${pctColor(sla.uptime_percentage)}`}>
        {sla.uptime_percentage.toFixed(3)}%
      </div>
      <div className="flex items-center justify-center gap-4 mt-3">
        <div className="text-center">
          <p className="text-[11px] text-slate-600 uppercase tracking-wider">Downtime</p>
          <p className="text-sm font-mono text-slate-300 mt-0.5">{formatDuration(sla.downtime_duration_sec)}</p>
        </div>
      </div>
      <p className="text-[11px] text-slate-600 mt-3">
        {new Date(sla.start_time).toLocaleDateString()} — {new Date(sla.end_time).toLocaleDateString()}
      </p>
    </div>
  );
}
