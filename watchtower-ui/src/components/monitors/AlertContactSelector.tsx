import { useState, useEffect } from 'react';
import { X, Plus, Trash2 } from 'lucide-react';
import { AlertContact } from '../../types';
import * as alertContactsApi from '../../api/alertContacts';
import * as monitorsApi from '../../api/monitors';
import AlertContactForm from '../alertContacts/AlertContactForm';

interface AlertContactSelectorProps {
  monitorId: string;
  assignedContacts: AlertContact[];
  onChanged: () => void;
}

export default function AlertContactSelector({ monitorId, assignedContacts, onChanged }: AlertContactSelectorProps) {
  const [allContacts, setAllContacts] = useState<AlertContact[]>([]);
  const [open, setOpen] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  useEffect(() => {
    if (open) { alertContactsApi.getAlertContacts().then((res) => setAllContacts(res.data)); }
  }, [open]);

  const assignedIds = new Set(assignedContacts.map((c) => c.id));
  const available = allContacts.filter((c) => !assignedIds.has(c.id));

  const handleAdd = async (contactId: string) => {
    await monitorsApi.addAlertContact(monitorId, contactId);
    onChanged();
  };

  const handleRemove = async (contactId: string) => {
    await monitorsApi.removeAlertContact(monitorId, contactId);
    onChanged();
  };

  const handleCreate = async (data: Record<string, unknown>) => {
    const res = await alertContactsApi.createAlertContact(data);
    await monitorsApi.addAlertContact(monitorId, res.data.id);
    setShowCreate(false);
    onChanged();
  };

  return (
    <>
      <button type="button" onClick={() => setOpen(true)} className="text-xs font-medium text-emerald-400 hover:text-emerald-300 transition-colors">+ Add</button>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/70" onClick={() => setOpen(false)} />
          <div className="relative bg-card-bg rounded-xl border border-border shadow-2xl shadow-black/40 w-full max-w-md mx-4 p-6 max-h-[80vh] overflow-y-auto">
            {showCreate ? (
              <>
                <div className="flex items-center justify-between mb-5">
                  <h4 className="font-semibold text-slate-100">New Alert Contact</h4>
                  <button onClick={() => setShowCreate(false)} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
                </div>
                <AlertContactForm
                  onSubmit={handleCreate}
                  onCancel={() => setShowCreate(false)}
                />
              </>
            ) : (
              <>
                <div className="flex items-center justify-between mb-5">
                  <h4 className="font-semibold text-slate-100">Manage Alert Contacts</h4>
                  <button onClick={() => setOpen(false)} className="text-slate-500 hover:text-slate-300 transition-colors"><X size={20} /></button>
                </div>
                {assignedContacts.length > 0 && (
                  <div className="mb-5">
                    <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">Assigned</p>
                    <div className="space-y-1">
                      {assignedContacts.map((c) => (
                        <div key={c.id} className="flex items-center justify-between py-2 px-3 bg-app-bg rounded-lg">
                          <span className="text-sm text-slate-200">{c.name}</span>
                          <button onClick={() => handleRemove(c.id)} className="text-slate-600 hover:text-red-400 transition-colors"><Trash2 size={14} /></button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                <div>
                  <p className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider mb-2">Available</p>
                  {available.length === 0 ? (
                    <p className="text-sm text-slate-600">No more contacts available.</p>
                  ) : (
                    <div className="space-y-1 max-h-48 overflow-y-auto">
                      {available.map((c) => (
                        <div key={c.id} className="flex items-center justify-between py-2 px-3 hover:bg-app-bg rounded-lg">
                          <span className="text-sm text-slate-300">{c.name}</span>
                          <button onClick={() => handleAdd(c.id)} className="text-emerald-400 hover:text-emerald-300 transition-colors"><Plus size={14} /></button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => setShowCreate(true)}
                  className="mt-4 w-full py-2 border border-dashed border-border rounded-lg text-xs text-slate-500 hover:text-slate-300 hover:border-slate-500 transition-colors"
                >
                  + Create New Contact
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </>
  );
}
