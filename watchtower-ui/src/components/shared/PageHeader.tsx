import { ReactNode } from 'react';
import { Plus } from 'lucide-react';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  action?: { label: string; onClick: () => void };
  children?: ReactNode;
}

export default function PageHeader({ title, subtitle, action, children }: PageHeaderProps) {
  return (
    <div className="mb-8 flex flex-wrap items-center justify-between gap-5 border-b border-border pb-7">
      <div>
        <p className="eyebrow mb-3">Workspace / {title}</p>
        <h1 className="text-3xl font-semibold tracking-tight text-slate-100">{title}</h1>
        {subtitle && <p className="mt-2 max-w-xl text-sm leading-relaxed text-slate-400">{subtitle}</p>}
        {children}
      </div>
      {action && (
        <button
          onClick={action.onClick}
          className="primary-button shrink-0"
        >
          <Plus size={16} aria-hidden="true" />{action.label.replace(/^\+\s*/, '')}
        </button>
      )}
    </div>
  );
}
