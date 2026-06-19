import { ReactNode } from 'react';
import { Plus } from 'lucide-react';

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description: string;
  action?: { label: string; onClick: () => void };
}

export default function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      {icon && (
        <div className="w-16 h-16 rounded-2xl bg-slate-500/5 flex items-center justify-center mb-6 text-slate-600">
          {icon}
        </div>
      )}
      <h3 className="text-lg font-semibold text-slate-200 mb-2">{title}</h3>
      <p className="text-sm text-slate-500 mb-8 max-w-sm">{description}</p>
      {action && (
        <button
          onClick={action.onClick}
          className="inline-flex items-center gap-2 px-4 py-2.5 bg-emerald-500 text-slate-900 rounded-lg text-sm font-semibold hover:bg-emerald-400 transition-all"
        >
          <Plus size={16} />
          {action.label}
        </button>
      )}
    </div>
  );
}
