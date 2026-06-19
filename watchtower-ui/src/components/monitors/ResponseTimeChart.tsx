import { useState } from 'react';
import { MonitorCheck } from '../../types';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  ReferenceDot,
} from 'recharts';

interface ResponseTimeChartProps {
  checks: MonitorCheck[] | null;
  loading: boolean;
}

function formatTime(ts: string): string {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatDate(ts: string): string {
  const d = new Date(ts);
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' +
    d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

const CustomTooltip = ({ active, payload }: any) => {
  if (!active || !payload?.length) return null;
  const d = payload[0].payload;
  return (
    <div className="bg-slate-800 border border-border rounded-lg px-3 py-2 shadow-xl">
      <p className="text-xs font-mono text-emerald-400 font-semibold">{d.latency_ms} ms</p>
      <p className="text-[10px] text-slate-500 mt-0.5">{formatDate(d.check_time)}</p>
    </div>
  );
};

export default function ResponseTimeChart({ checks, loading }: ResponseTimeChartProps) {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);

  if (loading) {
    return <div className="h-[160px] skeleton rounded-lg" />;
  }

  if (!checks || checks.length < 2) {
    return <p className="text-sm text-slate-500 py-4">Not enough data for chart.</p>;
  }

  const filtered = checks.filter((c) => c.latency_ms != null);
  if (filtered.length < 2) {
    return <p className="text-sm text-slate-500 py-4">Not enough data for chart.</p>;
  }

  const data = filtered.map((c) => ({
    ...c,
    timeLabel: formatTime(c.check_time),
  }));

  const maxLatency = Math.max(...filtered.map((c) => c.latency_ms), 1);

  return (
    <div className="w-full" onMouseLeave={() => setHoverIndex(null)}>
      <ResponsiveContainer width="100%" height={160}>
        <LineChart
          data={data}
          margin={{ top: 8, right: 8, left: -16, bottom: 0 }}
          onMouseMove={(e) => {
            if (e?.activeTooltipIndex != null) setHoverIndex(Number(e.activeTooltipIndex));
          }}
          onMouseLeave={() => setHoverIndex(null)}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#1e2a3d" vertical={false} />
          <XAxis
            dataKey="timeLabel"
            tick={{ fontSize: 10, fill: '#5e6f8d', fontFamily: 'JetBrains Mono, monospace' }}
            axisLine={{ stroke: '#1e2a3d' }}
            tickLine={false}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={[0, Math.ceil(maxLatency * 1.15)]}
            tick={{ fontSize: 10, fill: '#5e6f8d', fontFamily: 'JetBrains Mono, monospace' }}
            axisLine={false}
            tickLine={false}
            tickFormatter={(v: number) => `${v}ms`}
            width={44}
          />
          <Tooltip content={<CustomTooltip />} cursor={{ stroke: '#334155', strokeWidth: 1, strokeDasharray: '4 4' }} />
          <Line
            type="monotone"
            dataKey="latency_ms"
            stroke="#10b981"
            strokeWidth={1}
            dot={false}
            activeDot={false}
            isAnimationActive={false}
          />
          {hoverIndex != null && data[hoverIndex] && (
            <ReferenceDot
              x={data[hoverIndex].timeLabel}
              y={data[hoverIndex].latency_ms}
              r={4}
              fill="#10b981"
              stroke="#0b1120"
              strokeWidth={2}
              style={{ filter: 'drop-shadow(0 0 6px rgba(16, 185, 129, 0.6))' }}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
