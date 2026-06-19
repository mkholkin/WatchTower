import { useEffect, useCallback } from 'react';
import { X } from 'lucide-react';

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'Confirm',
  danger = false,
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    },
    [onCancel]
  );

  useEffect(() => {
    if (open) document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [open, handleKeyDown]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/70" onClick={onCancel} />
      <div className="relative bg-card-bg rounded-xl border border-border p-6 w-full max-w-md mx-4 shadow-2xl shadow-black/40">
        <div className="flex items-start justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-100">{title}</h3>
          <button onClick={onCancel} className="text-slate-500 hover:text-slate-300 transition-colors">
            <X size={20} />
          </button>
        </div>
        <p className="text-sm text-slate-400 mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={loading}
            className="px-4 py-2 text-sm font-medium text-slate-300 bg-border rounded-lg hover:bg-border-hover disabled:opacity-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={`px-4 py-2 text-sm font-semibold rounded-lg disabled:opacity-50 transition-all ${
              danger
                ? 'bg-red-600 text-white hover:bg-red-500'
                : 'bg-emerald-500 text-slate-900 hover:bg-emerald-400'
            }`}
          >
            {loading ? 'Please wait...' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
