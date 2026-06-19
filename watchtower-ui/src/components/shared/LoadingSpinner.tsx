import { Loader2 } from 'lucide-react';

export function LoadingSpinner({ className = '' }: { className?: string }) {
  return (
    <div className={`flex items-center justify-center ${className}`}>
      <Loader2 className="animate-spin text-emerald-400" size={32} />
    </div>
  );
}
