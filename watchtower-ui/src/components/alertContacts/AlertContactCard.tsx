import { AlertContact } from '../../types';
import { Pencil, Trash2 } from 'lucide-react';

interface AlertContactCardProps {
  contact: AlertContact;
  onToggle: (id: string, enabled: boolean) => void;
  onEdit: (contact: AlertContact) => void;
  onDelete: (contact: AlertContact) => void;
}

export default function AlertContactCard({ contact, onToggle, onEdit, onDelete }: AlertContactCardProps) {
  return (
    <div className={`bg-card-bg rounded-xl border border-border p-5 shadow-sm shadow-black/10 hover:bg-card-hover transition-colors duration-200 ${!contact.is_enabled ? 'opacity-50' : ''}`}>
      <div className="flex items-start justify-between">
        <div>
          <h4 className="font-semibold text-slate-100">{contact.name}</h4>
          <span className="inline-block mt-1.5 px-2 py-0.5 bg-blue-500/10 text-blue-400 text-[11px] font-medium rounded-md border border-blue-500/10">
            Telegram
          </span>
        </div>
        <div className="flex items-center gap-0.5">
          <button onClick={() => onEdit(contact)} className="p-1.5 text-slate-600 hover:text-slate-300 rounded transition-colors">
            <Pencil size={15} />
          </button>
          <button onClick={() => onDelete(contact)} className="p-1.5 text-slate-600 hover:text-red-400 rounded transition-colors">
            <Trash2 size={15} />
          </button>
        </div>
      </div>
      <div className="flex items-center justify-between mt-4 pt-4 border-t border-border">
        <span className={`text-xs font-medium ${contact.is_enabled ? 'text-emerald-400' : 'text-slate-600'}`}>
          {contact.is_enabled ? 'Active' : 'Disabled'}
        </span>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={contact.is_enabled}
            onChange={() => onToggle(contact.id, !contact.is_enabled)}
            className="sr-only peer"
          />
          <div className="w-9 h-5 bg-border rounded-full peer peer-checked:bg-emerald-500 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all" />
        </label>
      </div>
    </div>
  );
}
