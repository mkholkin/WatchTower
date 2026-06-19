import { useState } from 'react';
import { Wrench } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import * as maintenanceWindowsApi from '../api/maintenanceWindows';
import { MaintenanceWindow } from '../types';
import PageHeader from '../components/shared/PageHeader';
import MaintenanceWindowCard from '../components/maintenanceWindows/MaintenanceWindowCard';
import MaintenanceWindowForm, { createWindowWithMonitors } from '../components/maintenanceWindows/MaintenanceWindowForm';
import ConfirmDialog from '../components/shared/ConfirmDialog';
import { LoadingSpinner } from '../components/shared/LoadingSpinner';
import ErrorAlert from '../components/shared/ErrorAlert';
import EmptyState from '../components/shared/EmptyState';

export default function MaintenanceWindowsPage() {
  const { data: windows, error, loading, refetch } = useApi<MaintenanceWindow[]>(
    () => maintenanceWindowsApi.getMaintenanceWindows()
  );
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<MaintenanceWindow | null>(null);
  const [deleting, setDeleting] = useState<MaintenanceWindow | null>(null);

  const handleToggle = async (id: string, isActive: boolean) => {
    try {
      await maintenanceWindowsApi.updateMaintenanceWindow(id, { config: { type: 'manual', is_active: isActive } });
      refetch();
    } catch {}
  };

  const handleCreate = async (data: Record<string, unknown>) => {
    await createWindowWithMonitors(data);
    refetch();
  };

  const handleEdit = async (data: Record<string, unknown>) => {
    if (!editing) return;
    const { monitor_ids, ...windowData } = data;
    await maintenanceWindowsApi.updateMaintenanceWindow(editing.id, windowData);
    // Handle monitor changes if needed — for now just refetch
    setEditing(null);
    refetch();
  };

  const handleDelete = async () => {
    if (!deleting) return;
    await maintenanceWindowsApi.deleteMaintenanceWindow(deleting.id);
    setDeleting(null);
    refetch();
  };

  return (
    <div>
      <PageHeader
        title="Maintenance Windows"
        subtitle="Schedule maintenance to suppress alerts during planned downtime."
        action={{ label: '+ New Window', onClick: () => setShowForm(true) }}
      />

      {loading && <div className="py-16"><LoadingSpinner /></div>}
      {error && <ErrorAlert message={error} onRetry={refetch} />}

      {!loading && !error && windows?.length === 0 && (
        <EmptyState
          icon={<Wrench size={48} />}
          title="No maintenance windows"
          description="Create maintenance windows to suppress alerts during planned downtime."
          action={{ label: 'New Window', onClick: () => setShowForm(true) }}
        />
      )}

      {!loading && !error && windows && windows.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {windows.map((w) => (
            <MaintenanceWindowCard
              key={w.id}
              window={w}
              onToggle={handleToggle}
              onEdit={setEditing}
              onDelete={setDeleting}
            />
          ))}
        </div>
      )}

      {showForm && <MaintenanceWindowForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} />}
      {editing && <MaintenanceWindowForm initialData={editing} onSubmit={handleEdit} onCancel={() => setEditing(null)} />}

      <ConfirmDialog
        open={!!deleting}
        title="Delete Maintenance Window"
        message={`Are you sure you want to delete "${deleting?.title}"?`}
        confirmLabel="Delete"
        danger
        onConfirm={handleDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  );
}
