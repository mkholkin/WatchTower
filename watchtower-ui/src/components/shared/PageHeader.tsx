import { ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  action?: { label: string; onClick: () => void };
  children?: ReactNode;
}

export default function PageHeader({ title, subtitle, action, children }: PageHeaderProps) {
  return (
    <div className="flex items-center justify-between mb-8">
      <div>
        <h2 className="text-2xl font-bold text-slate-100">{title}</h2>
        {subtitle && <p className="text-sm text-slate-500 mt-1">{subtitle}</p>}
        {children}
      </div>
      {action && (
        <button
          onClick={action.onClick}
          className="px-4 py-2.5 bg-emerald-500 text-slate-900 text-sm font-semibold rounded-lg hover:bg-emerald-400 transition-all"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
