import React, { useEffect, useState } from 'react';
import { X, Printer, CheckCircle2, Copy } from 'lucide-react';
import { Order } from '../../domain/types';
import { api } from '../../services/api';

interface ReceiptModalProps {
  order: Order | null;
  isOpen: boolean;
  onClose: () => void;
}

export const ReceiptModal: React.FC<ReceiptModalProps> = ({
  order,
  isOpen,
  onClose,
}) => {
  const [receiptText, setReceiptText] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (isOpen && order) {
      setIsLoading(true);
      api
        .getReceipt(order.id)
        .then((text) => setReceiptText(text))
        .catch((err) => {
          setReceiptText(`Order: ${order.order_number}\nTotal: ${order.total_amount}\n${err.message}`);
        })
        .finally(() => setIsLoading(false));
    }
  }, [isOpen, order]);

  if (!isOpen || !order) return null;

  const handlePrint = () => {
    window.print();
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(receiptText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl flex flex-col max-h-[90dvh]">
        {/* Header */}
        <div className="px-5 sm:px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
            <div>
              <h3 className="font-semibold text-slate-900 dark:text-slate-100 text-sm">Payment Successful</h3>
              <p className="text-xs text-slate-600 dark:text-slate-400 font-mono">{order.order_number}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close modal"
            className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Thermal Print Preview Container */}
        <div className="p-5 sm:p-6 flex-1 overflow-y-auto flex justify-center bg-slate-100 dark:bg-slate-950">
          {isLoading ? (
            <div className="h-64 flex items-center justify-center text-xs text-slate-600 dark:text-slate-400">
              Generating formatted thermal receipt...
            </div>
          ) : (
            <div className="w-full max-w-[18rem] bg-white text-black p-4 rounded-lg border border-slate-200 shadow-inner font-mono text-[11px] leading-relaxed whitespace-pre-wrap select-text">
              {receiptText}
            </div>
          )}
        </div>

        {/* Footer actions */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 flex gap-2 print:hidden">
          <button
            type="button"
            onClick={handleCopy}
            className="min-h-[44px] px-3.5 py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 border border-slate-300 dark:border-slate-700 text-xs font-semibold flex items-center gap-1.5 transition-colors"
          >
            <Copy className="w-4 h-4" />
            <span>{copied ? 'Copied!' : 'Copy'}</span>
          </button>
          <button
            type="button"
            onClick={handlePrint}
            className="flex-1 min-h-[44px] py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 dark:hover:bg-indigo-500 text-white text-xs font-semibold flex items-center justify-center gap-2 transition-all shadow-lg shadow-indigo-600/20"
          >
            <Printer className="w-4 h-4" />
            <span>Print Receipt</span>
          </button>
        </div>
      </div>
    </div>
  );
};
