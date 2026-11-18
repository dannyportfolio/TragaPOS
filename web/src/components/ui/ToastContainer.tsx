import React from 'react';
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react';
import { useToastStore, ToastType } from '../../store/useToastStore';

const STYLES: Record<ToastType, { border: string; icon: React.ReactNode }> = {
  success: {
    border: 'border-emerald-300 dark:border-emerald-500/40 bg-white/95 dark:bg-slate-900/95 text-slate-900 dark:text-emerald-100',
    icon: <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0" />,
  },
  error: {
    border: 'border-rose-300 dark:border-rose-500/40 bg-white/95 dark:bg-slate-900/95 text-slate-900 dark:text-rose-100',
    icon: <AlertCircle className="w-5 h-5 text-rose-600 dark:text-rose-400 shrink-0" />,
  },
  info: {
    border: 'border-indigo-300 dark:border-indigo-500/40 bg-white/95 dark:bg-slate-900/95 text-slate-900 dark:text-indigo-100',
    icon: <Info className="w-5 h-5 text-indigo-600 dark:text-indigo-400 shrink-0" />,
  },
};

export const ToastContainer: React.FC = () => {
  const { toasts, removeToast } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div
      aria-live="polite"
      className="fixed bottom-4 right-4 left-4 sm:left-auto z-[100] flex flex-col gap-2.5 sm:w-96 pointer-events-none"
    >
      {toasts.map((t) => {
        const style = STYLES[t.type];
        return (
          <div
            key={t.id}
            role="status"
            className={`pointer-events-auto flex items-start gap-3 p-3.5 rounded-xl border shadow-2xl backdrop-blur-md transition-all ${style.border}`}
          >
            {style.icon}
            <div className="flex-1 min-w-0">
              {t.title && <p className="text-xs font-semibold mb-0.5">{t.title}</p>}
              <p className="text-xs text-slate-700 dark:text-slate-200 break-words leading-relaxed">{t.message}</p>
            </div>
            <button
              type="button"
              onClick={() => removeToast(t.id)}
              aria-label="Dismiss notification"
              className="min-w-[36px] min-h-[36px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 flex items-center justify-center transition-colors shrink-0"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
};
