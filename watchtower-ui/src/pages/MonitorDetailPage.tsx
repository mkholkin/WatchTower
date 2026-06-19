import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Pencil, Trash2 } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import * as monitorsApi from '../api/monitors';
import * as metricsApi from '../api/metrics';
import { Monitor, MonitorCheck, MonitorSLA, MonitorStatusEvent } from '../types';
import StatusBadge from '../components/shared/StatusBadge';
import SLADisplay from '../components/monitors/SLADisplay';
import ChecksTable from '../components/monitors/ChecksTable';
import StatusHistoryBar from '../components/monitors/StatusHistoryBar';
import ResponseTimeChart from '../components/monitors/ResponseTimeChart';
import AlertContactSelector from '../components/monitors/AlertContactSelector';
import MaintenanceWindowSelector from '../components/monitors/MaintenanceWindowSelector';
import MonitorForm from '../components/monitors/MonitorForm';
import ConfirmDialog from '../components/shared/ConfirmDialog';
import { LoadingSpinner } from '../components/shared/LoadingSpinner';
import ErrorAlert from '../components/shared/ErrorAlert';

export default function MonitorDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const monitorId = id!;

  const { data: monitor, error, loading, refetch } = useApi<Monitor>(
    () => monitorsApi.getMonitor(monitorId),
    [monitorId]
  );
  const { data: checks, loading: checksLoading } = useApi<MonitorCheck[]>(
    () => metricsApi.getChecks(monitorId, { limit: 50 }),
    [monitorId]
  );
  const { data: sla, loading: slaLoading } = useApi<MonitorSLA>(
    () => metricsApi.getSLA(monitorId),
    [monitorId]
  );
  const { data: statusHistory, loading: historyLoading } = useApi<MonitorStatusEvent[]>(
    () => metricsApi.getStatusHistory(monitorId),
    [monitorId]
  );

  const [showEdit, setShowEdit] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [toggling, setToggling] = useState(false);

  const handleToggle = async () => {
    if (!monitor) return;
    setToggling(true);
    try {
      if (monitor.is_enabled) {
        await monitorsApi.disableMonitor(monitorId);
      } else {
        await monitorsApi.enableMonitor(monitorId);
      }
      refetch();
    } finally {
      setToggling(false);
    }
  };

  const handleEdit = async (data: Record<string, unknown>) => {
    await monitorsApi.updateMonitor(monitorId, data);
    refetch();
  };

  const handleDelete = async () => {
    await monitorsApi.deleteMonitor(monitorId);
    navigate('/');
  };

  if (loading) {
    return (
      <div className="py-16">
        <LoadingSpinner />
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <Link to="/" className="inline-flex items-center gap-1.5 text-sm text-slate-400 hover:text-slate-200 transition-colors mb-4">
          <ArrowLeft size={16} /> Back to Dashboard
        </Link>
        <ErrorAlert message={error} onRetry={refetch} />
      </div>
    );
  }

  if (!monitor) return null;

  return (
    <div>
      {/* Top bar */}
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center gap-4 min-w-0">
          <Link to="/" className="text-slate-500 hover:text-slate-300 transition-colors shrink-0">
            <ArrowLeft size={20} />
          </Link>
          <div className="min-w-0">
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-bold text-slate-100 truncate">{monitor.label}</h2>
              <StatusBadge status={monitor.status} />
            </div>
            <p className="text-sm text-slate-500 mt-0.5 font-mono truncate">{monitor.endpoint}</p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={handleToggle}
            disabled={toggling}
            className={`px-4 py-2 text-sm font-medium rounded-lg transition-all ${
              monitor.is_enabled
                ? 'bg-border text-slate-300 hover:bg-border-hover'
                : 'bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20'
            }`}
          >
            {monitor.is_enabled ? 'Disable' : 'Enable'}
          </button>
          <button onClick={() => setShowEdit(true)} className="p-2 text-slate-500 hover:text-slate-300 rounded-lg hover:bg-border transition-colors">
            <Pencil size={17} />
          </button>
          <button onClick={() => setShowDelete(true)} className="p-2 text-slate-500 hover:text-red-400 rounded-lg hover:bg-border transition-colors">
            <Trash2 size={17} />
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left column */}
        <div className="lg:col-span-2 space-y-6">
          {/* Configuration card */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <h3 className="font-semibold text-slate-200 mb-4">Configuration</h3>
            <dl className="grid grid-cols-2 gap-y-4 gap-x-8 text-sm">
              <div>
                <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">Probe Interval</dt>
                <dd className="text-slate-200 mt-1 font-mono">{monitor.probe_interval}s</dd>
              </div>
              <div>
                <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">HTTP Method</dt>
                <dd className="text-slate-200 mt-1 font-mono">{monitor.network_config.method}</dd>
              </div>
              <div>
                <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">Follow Redirects</dt>
                <dd className="text-slate-200 mt-1">{monitor.network_config.follow_redirects ? 'Yes' : 'No'}</dd>
              </div>
              <div>
                <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">Expected Status</dt>
                <dd className="text-slate-200 mt-1 font-mono">{monitor.expectations.expected_status_codes.join(', ')}</dd>
              </div>
              <div>
                <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">Max Response Time</dt>
                <dd className="text-slate-200 mt-1 font-mono">{monitor.expectations.expected_response_time_ms} ms</dd>
              </div>
              {monitor.description && (
                <div className="col-span-2">
                  <dt className="text-slate-500 text-[11px] uppercase tracking-wider font-medium">Description</dt>
                  <dd className="text-slate-300 mt-1">{monitor.description}</dd>
                </div>
              )}
            </dl>
          </div>

          {/* Status History */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <h3 className="font-semibold text-slate-200 mb-4">Status History</h3>
            <StatusHistoryBar events={statusHistory} loading={historyLoading} />
          </div>

          {/* Response Time Chart */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <h3 className="font-semibold text-slate-200 mb-4">Response Time</h3>
            <ResponseTimeChart checks={checks} loading={checksLoading} />
          </div>

          {/* Checks table */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <h3 className="font-semibold text-slate-200 mb-4">Check History</h3>
            <ChecksTable checks={checks} loading={checksLoading} />
          </div>
        </div>

        {/* Right column */}
        <div className="space-y-6">
          {/* SLA card */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <h3 className="font-semibold text-slate-200 mb-2">Uptime (SLA)</h3>
            <SLADisplay sla={sla} loading={slaLoading} />
          </div>

          {/* Alert contacts */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-slate-200">Alert Contacts</h3>
              <AlertContactSelector
                monitorId={monitorId}
                assignedContacts={monitor.alert_contacts}
                onChanged={refetch}
              />
            </div>
            {monitor.alert_contacts.length === 0 ? (
              <p className="text-sm text-slate-600">No contacts assigned.</p>
            ) : (
              <div className="space-y-1">
                {monitor.alert_contacts.map((c) => (
                  <div
                    key={c.id}
                    className="group relative flex items-center gap-2 text-xs py-1.5 px-2.5 bg-app-bg rounded-lg hover:bg-card-hover transition-colors cursor-default"
                    title={`Chat ID: ${c.config.chat_id}${!c.is_enabled ? ' (disabled)' : ''}`}
                  >
                    {c.config.platform === 'telegram' ? (
                      <svg viewBox="0 0 24 24" className="w-3.5 h-3.5 text-blue-400 shrink-0" fill="currentColor">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69.01-.03.01-.14-.07-.2-.08-.06-.19-.04-.27-.02-.12.02-1.96 1.25-5.54 3.66-.52.36-1 .53-1.42.52-.47-.01-1.37-.26-2.03-.48-.82-.27-1.47-.42-1.41-.88.03-.24.37-.49 1.02-.74 3.98-1.73 6.63-2.87 7.96-3.42 3.79-1.57 4.58-1.85 5.09-1.86.11 0 .37.03.54.16.14.11.18.26.2.38.02.12.04.39.02.59z"/>
                      </svg>
                    ) : (
                      <span className="w-3.5 h-3.5 rounded bg-slate-600 shrink-0" />
                    )}
                    <span className={`truncate ${c.is_enabled ? 'text-slate-300' : 'text-slate-600'}`}>{c.name}</span>
                    <span className={`w-1.5 h-1.5 rounded-full shrink-0 ml-auto ${c.is_enabled ? 'bg-emerald-400' : 'bg-slate-600'}`} />
                    {/* Hover tooltip */}
                    <div className="absolute bottom-full left-0 mb-1.5 hidden group-hover:block z-20 pointer-events-none">
                      <div className="bg-slate-800 border border-border rounded-lg px-2.5 py-2 shadow-xl text-[11px] whitespace-nowrap">
                        <p className="text-slate-300 font-medium">{c.name}</p>
                        <p className="text-slate-500 mt-0.5">Platform: <span className="text-slate-400 capitalize">{c.config.platform}</span></p>
                        {c.config.platform === 'telegram' && (
                          <p className="text-slate-500">Chat ID: <span className="text-slate-400 font-mono">{c.config.chat_id}</span></p>
                        )}
                        <p className="text-slate-500">Status: <span className={c.is_enabled ? 'text-emerald-400' : 'text-slate-500'}>{c.is_enabled ? 'Active' : 'Disabled'}</span></p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Maintenance windows */}
          <div className="bg-card-bg rounded-xl border border-border p-6 shadow-sm shadow-black/10">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-slate-200">Maintenance Windows</h3>
              <MaintenanceWindowSelector
                monitorId={monitorId}
                assignedWindows={monitor.maintenance_windows}
                onChanged={refetch}
              />
            </div>
            {monitor.maintenance_windows.length === 0 ? (
              <p className="text-sm text-slate-600">No windows assigned.</p>
            ) : (
              <div className="space-y-2">
                {monitor.maintenance_windows.map((w) => (
                  <div key={w.id} className="flex items-center justify-between text-sm py-2 px-3 bg-app-bg rounded-lg">
                    <span className="text-slate-300">{w.title}</span>
                    <span className="text-[11px] text-slate-500 capitalize">{w.config.type.replace('_', ' ')}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {showEdit && (
        <MonitorForm initialData={monitor} onSubmit={handleEdit} onCancel={() => setShowEdit(false)} />
      )}

      <ConfirmDialog
        open={showDelete}
        title="Delete Monitor"
        message={`Are you sure you want to delete "${monitor.label}"? This action cannot be undone.`}
        confirmLabel="Delete"
        danger
        onConfirm={handleDelete}
        onCancel={() => setShowDelete(false)}
      />
    </div>
  );
}
