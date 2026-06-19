import { useState } from 'react';
import { Bell } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import * as alertContactsApi from '../api/alertContacts';
import { AlertContact } from '../types';
import PageHeader from '../components/shared/PageHeader';
import AlertContactCard from '../components/alertContacts/AlertContactCard';
import AlertContactForm from '../components/alertContacts/AlertContactForm';
import ConfirmDialog from '../components/shared/ConfirmDialog';
import { LoadingSpinner } from '../components/shared/LoadingSpinner';
import ErrorAlert from '../components/shared/ErrorAlert';
import EmptyState from '../components/shared/EmptyState';

export default function AlertContactsPage() {
  const { data: contacts, error, loading, refetch } = useApi<AlertContact[]>(
    () => alertContactsApi.getAlertContacts()
  );
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<AlertContact | null>(null);
  const [deleting, setDeleting] = useState<AlertContact | null>(null);

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      if (enabled) await alertContactsApi.enableAlertContact(id);
      else await alertContactsApi.disableAlertContact(id);
      refetch();
    } catch {}
  };

  const handleCreate = async (data: Record<string, unknown>) => {
    await alertContactsApi.createAlertContact(data);
    refetch();
  };

  const handleEdit = async (data: Record<string, unknown>) => {
    if (!editing) return;
    await alertContactsApi.updateAlertContact(editing.id, data);
    setEditing(null);
    refetch();
  };

  const handleDelete = async () => {
    if (!deleting) return;
    await alertContactsApi.deleteAlertContact(deleting.id);
    setDeleting(null);
    refetch();
  };

  return (
    <div>
      <PageHeader
        title="Alert Contacts"
        subtitle="Manage who gets notified when monitors go down."
        action={{ label: '+ New Contact', onClick: () => setShowForm(true) }}
      />

      {loading && (
        <div className="py-16"><LoadingSpinner /></div>
      )}

      {error && <ErrorAlert message={error} onRetry={refetch} />}

      {!loading && !error && contacts?.length === 0 && (
        <EmptyState
          icon={<Bell size={48} />}
          title="No alert contacts"
          description="Create alert contacts to get notified when your monitors go down."
          action={{ label: 'New Contact', onClick: () => setShowForm(true) }}
        />
      )}

      {!loading && !error && contacts && contacts.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {contacts.map((c) => (
            <AlertContactCard
              key={c.id}
              contact={c}
              onToggle={handleToggle}
              onEdit={setEditing}
              onDelete={setDeleting}
            />
          ))}
        </div>
      )}

      {showForm && <AlertContactForm onSubmit={handleCreate} onCancel={() => setShowForm(false)} />}
      {editing && <AlertContactForm initialData={editing} onSubmit={handleEdit} onCancel={() => setEditing(null)} />}

      <ConfirmDialog
        open={!!deleting}
        title="Delete Alert Contact"
        message={`Are you sure you want to delete "${deleting?.name}"?`}
        confirmLabel="Delete"
        danger
        onConfirm={handleDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  );
}
