import { useState, FormEvent } from 'react';
import { X } from 'lucide-react';
import { AlertContact, AlertContactPlatform } from '../../types';

interface AlertContactFormProps {
  initialData?: AlertContact;
  onSubmit: (data: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
}

const inputClass = "w-full px-3 py-2.5 bg-app-bg border border-border rounded-lg text-sm text-slate-100 placeholder-slate-600 focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50 outline-none transition-all";
const labelClass = "block text-sm font-medium text-slate-400 mb-1.5";

export default function AlertContactForm({ initialData, onSubmit, onCancel }: AlertContactFormProps) {
  const isEdit = !!initialData;

  const [name, setName] = useState(initialData?.name || '');
  const [platform, setPlatform] = useState<AlertContactPlatform>(
    initialData?.config?.platform || 'telegram'
  );
  const [chatId, setChatId] = useState(
    initialData?.config?.platform === 'telegram' ? initialData.config.chat_id?.toString() || '' : ''
  );
  const [token, setToken] = useState(
    initialData?.config?.platform === 'telegram' ? initialData.config.token || '' : ''
  );

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      let config: Record<string, unknown>;
      if (platform === 'telegram') {
        config = { platform: 'telegram', chat_id: Number(chatId), token };
      } else {
        config = { platform };
      }
      await onSubmit({ name, platform, config });
      onCancel();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to save alert contact');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/70" onClick={onCancel} />
      <div className="relative bg-card-bg rounded-xl border border-border shadow-2xl shadow-black/40 w-full max-w-md mx-4 p-6">
        <div className="flex items-center justify-between mb-5">
          <h3 className="font-semibold text-slate-100">{isEdit ? 'Edit Alert Contact' : 'New Alert Contact'}</h3>
          <button onClick={onCancel} className="text-slate-500 hover:text-slate-300 transition-colors">
            <X size={20} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className={labelClass}>Name *</label>
            <input type="text" required value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
          </div>
          <div>
            <label className={labelClass}>Platform</label>
            <select value={platform} onChange={(e) => setPlatform(e.target.value as AlertContactPlatform)} className={inputClass}>
              <option value="telegram">Telegram</option>
            </select>
          </div>

          {platform === 'telegram' && (
            <>
              <div>
                <label className={labelClass}>Chat ID *</label>
                <input type="number" required value={chatId} onChange={(e) => setChatId(e.target.value)} className={inputClass} />
              </div>
              <div>
                <label className={labelClass}>Bot Token *</label>
                <input type="text" required value={token} onChange={(e) => setToken(e.target.value)} className={inputClass} />
              </div>
            </>
          )}

          {error && (
            <div className="bg-red-500/5 border border-red-500/20 text-red-400 px-4 py-2.5 rounded-lg text-sm">{error}</div>
          )}
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={onCancel} disabled={submitting} className="px-4 py-2 text-sm font-medium text-slate-300 bg-border rounded-lg hover:bg-border-hover disabled:opacity-50 transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={submitting} className="px-4 py-2 text-sm font-semibold text-slate-900 bg-emerald-500 rounded-lg hover:bg-emerald-400 disabled:opacity-50 transition-all">
              {submitting ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Contact'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
