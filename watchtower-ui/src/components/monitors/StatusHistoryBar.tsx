import { useState } from 'react';
import { MonitorStatusEvent } from '../../types';

interface StatusHistoryBarProps {
    events: MonitorStatusEvent[] | null;
    loading: boolean;
}

const colorMap: Record<string, string> = {
    up: 'bg-emerald-500',
    down: 'bg-red-500',
    maintenance: 'bg-amber-500',
    unknown: 'bg-slate-500',
};

function formatTooltipDate(ts: string): string {
    return new Date(ts).toLocaleString();
}

export default function StatusHistoryBar({ events, loading }: StatusHistoryBarProps) {
    const [tooltip, setTooltip] = useState<{ x: number; event: MonitorStatusEvent } | null>(null);

    if (loading) {
        return (
            <div className="flex gap-0.5 h-6">
                {Array.from({ length: 40 }).map((_, i) => (
                    <div key={i} className="flex-1 h-full rounded-sm skeleton" />
                ))}
            </div>
        );
    }

    if (!events || events.length === 0) {
        return <p className="text-sm text-slate-500 py-2">No status history available.</p>;
    }

    // Get current time once so all calculations use the same reference point
    const now = Date.now();

    return (
        <div className="relative">
            <div
                className="flex gap-0.5 h-6 uptime-bar-group"
                onMouseLeave={() => setTooltip(null)}
            >
                {events.map((e, i) => {
                    const start = new Date(e.start_time).getTime();

                    // 1. Figure out the theoretical end time
                    let rawEnd = now;
                    if (e.end_time) {
                        rawEnd = new Date(e.end_time).getTime();
                    } else if (events[i + 1]) {
                        rawEnd = new Date(events[i + 1].start_time).getTime();
                    }

                    // 2. Clamp the end time so it cannot exceed the current time
                    const end = Math.min(rawEnd, now);

                    // Calculate duration (fallback to 1 to avoid negative/zero flexGrow values)
                    const duration = Math.max(end - start, 1);

                    return (
                        <div
                            key={i}
                            className={`h-full rounded-sm uptime-bar ${colorMap[e.status] || 'bg-slate-600'}`}
                            style={{
                                flexGrow: duration,
                                flexBasis: 0,
                                minWidth: '4px'
                            }}
                            onMouseEnter={(ev) => {
                                const rect = ev.currentTarget.parentElement!.getBoundingClientRect();
                                setTooltip({ x: ev.clientX - rect.left, event: e });
                            }}
                        />
                    );
                })}
            </div>

            {tooltip && (
                <div
                    className="absolute z-50 bottom-full mb-2 bg-slate-800 border border-border-hover rounded-lg px-3 py-2 shadow-xl pointer-events-none whitespace-nowrap"
                    style={{
                        left: `${tooltip.x}px`,
                        transform: 'translateX(-50%)'
                    }}
                >
                    <p className="text-xs text-slate-300 capitalize font-medium">{tooltip.event.status}</p>
                    <p className="text-[11px] text-slate-500 mt-0.5">{formatTooltipDate(tooltip.event.start_time)}</p>
                    {tooltip.event.reason && (
                        <p className="text-[11px] text-slate-500 mt-0.5 truncate max-w-[220px]">{tooltip.event.reason}</p>
                    )}
                </div>
            )}
        </div>
    );
}