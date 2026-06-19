import { AlertTriangle } from 'lucide-react';

interface ErrorAlertProps {
  message: string;
  onRetry?: () => void;
}

export default function ErrorAlert({ message, onRetry }: ErrorAlertProps) {
  return (
    <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-4 flex items-start gap-3">
      <div className="w-8 h-8 rounded-lg bg-red-500/10 flex items-center justify-center shrink-0">
        <AlertTriangle className="text-red-400" size={16} />
      </div>
      <div className="flex-1">
        <p className="text-red-300 text-sm">{message}</p>
        {onRetry && (
          <button
            onClick={onRetry}
            className="mt-2 text-sm font-medium text-red-400 hover:text-red-300 transition-colors"
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
