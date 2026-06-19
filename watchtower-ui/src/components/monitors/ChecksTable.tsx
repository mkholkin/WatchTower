import { MonitorCheck } from '../../types';
import StatusBadge from '../shared/StatusBadge';
import { LoadingSpinner } from '../shared/LoadingSpinner';

interface ChecksTableProps {
  checks: MonitorCheck[] | null;
  loading: boolean;
}

function formatTime(ts: string): string {
  const d = new Date(ts);
  return d.toLocaleString();
}

export default function ChecksTable({ checks, loading }: ChecksTableProps) {
  if (loading) {
    return <div className="py-8"><LoadingSpinner /></div>;
  }

  if (!checks || checks.length === 0) {
    return <p className="text-sm text-slate-600 py-6 text-center">No check data available.</p>;
  }

  return (
    <div className="overflow-x-auto -mx-6">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border">
            <th className="text-left py-3 px-6 font-medium text-slate-500 text-[11px] uppercase tracking-wider">Time</th>
            <th className="text-left py-3 px-3 font-medium text-slate-500 text-[11px] uppercase tracking-wider">Status</th>
            <th className="text-right py-3 px-3 font-medium text-slate-500 text-[11px] uppercase tracking-wider">Latency</th>
            <th className="text-right py-3 px-3 font-medium text-slate-500 text-[11px] uppercase tracking-wider">Code</th>
            <th className="text-left py-3 px-6 font-medium text-slate-500 text-[11px] uppercase tracking-wider">Error</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#1e2a3d]/50">
          {checks.map((c, i) => (
            <tr key={i} className="hover:bg-card-hover/50 transition-colors">
              <td className="py-3 px-6 text-slate-400 whitespace-nowrap text-xs font-mono">{formatTime(c.check_time)}</td>
              <td className="py-3 px-3"><StatusBadge status={c.status} /></td>
              <td className="py-3 px-3 text-right font-mono text-xs text-slate-300">{c.latency_ms} ms</td>
              <td className="py-3 px-3 text-right font-mono text-xs">
                <span className={c.status_code && c.status_code >= 400 ? 'text-red-400' : 'text-slate-400'}>
                  {c.status_code ?? '-'}
                </span>
              </td>
              <td className="py-3 px-6 text-xs text-slate-500 max-w-[200px] truncate">
                {c.network_failure
                  ? <span className="text-red-400">Network failure</span>
                  : (c.failure_reason || <span className="text-slate-700">-</span>)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
