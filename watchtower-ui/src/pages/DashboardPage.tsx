import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Radio } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import * as monitorsApi from '../api/monitors';
import PageHeader from '../components/shared/PageHeader';
import StatsBar from '../components/monitors/StatsBar';
import MonitorCard from '../components/monitors/MonitorCard';
import MonitorForm from '../components/monitors/MonitorForm';
import SkeletonCard from '../components/shared/SkeletonCard';
import ErrorAlert from '../components/shared/ErrorAlert';
import EmptyState from '../components/shared/EmptyState';
import { Monitor } from '../types';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { data: monitors, error, loading, refetch } = useApi<Monitor[]>(() => monitorsApi.getMonitors());
  const [showCreate, setShowCreate] = useState(false);

  const handleToggle = useCallback(
    async (id: string, enabled: boolean) => {
      try {
        if (enabled) {
          await monitorsApi.enableMonitor(id);
        } else {
          await monitorsApi.disableMonitor(id);
        }
        refetch();
      } catch {
        // ignore
      }
    },
    [refetch]
  );

  const handleCreate = async (data: Record<string, unknown>) => {
    await monitorsApi.createMonitor(data);
    refetch();
  };

  return (
    <div>
      <PageHeader
        title="Overview"
        subtitle="A clear view of your endpoints. A closer look when it matters."
        action={{ label: '+ New Monitor', onClick: () => setShowCreate(true) }}
      />

      {loading && (
        <div>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-6">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="bg-card-bg rounded-xl border border-border p-4 shadow-sm shadow-black/10">
                <div className="h-3 w-16 rounded skeleton mb-2" />
                <div className="h-7 w-10 rounded skeleton" />
              </div>
            ))}
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 2xl:grid-cols-3 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <SkeletonCard key={i} />
            ))}
          </div>
        </div>
      )}

      {error && <ErrorAlert message={error} onRetry={refetch} />}

      {!loading && !error && monitors?.length === 0 && (
        <EmptyState
          icon={<Radio size={48} />}
          title="No monitors yet"
          description="Create your first monitor to start tracking endpoint availability and performance."
          action={{ label: 'New Monitor', onClick: () => setShowCreate(true) }}
        />
      )}

      {!loading && !error && monitors && monitors.length > 0 && (
        <>
          <StatsBar monitors={monitors} />
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold text-slate-200">Your monitors <span className="ml-2 rounded-md border border-border px-2 py-0.5 text-xs font-normal text-slate-400">{monitors.length}</span></h2>
            <span className="text-xs text-slate-500">Endpoint availability</span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 2xl:grid-cols-3 gap-4">
            {monitors.map((m, i) => (
              <MonitorCard
                key={m.id}
                monitor={m}
                onToggle={handleToggle}
                onClick={() => navigate(`/monitors/${m.id}`)}
                index={i}
              />
            ))}
          </div>
        </>
      )}

      {showCreate && (
        <MonitorForm onSubmit={handleCreate} onCancel={() => setShowCreate(false)} />
      )}
    </div>
  );
}
