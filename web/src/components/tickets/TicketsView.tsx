import React, { useState } from 'react';
import QRCode from 'qrcode';
import {
  Ticket,
  QrCode,
  Copy,
  Printer,
  Clock,
  Users,
  Sparkles,
  Check,
} from 'lucide-react';
import { IssuedTicketResponse } from '../../domain/types';
import { api } from '../../services/api';
import { toast } from '../../store/useToastStore';

interface IssuedPassItem {
  response: IssuedTicketResponse;
  qrDataUrl: string;
}

const CATEGORIES = [
  'GENERAL_ADMISSION',
  'VIP_ALL_ACCESS',
  'WATERPARK_ENTRY',
  'ZOO_FAMILY_PASS',
];

export const TicketsView: React.FC = () => {
  const [category, setCategory] = useState('GENERAL_ADMISSION');
  const [pax, setPax] = useState(1);
  const [validHours, setValidHours] = useState(24);
  const [orderId, setOrderId] = useState('');
  const [isIssuing, setIsIssuing] = useState(false);

  const [activePass, setActivePass] = useState<IssuedPassItem | null>(null);
  const [history, setHistory] = useState<IssuedPassItem[]>([]);
  const [copied, setCopied] = useState(false);

  const handleIssueTicket = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsIssuing(true);
    try {
      const res = await api.issueTicket({
        category,
        pax: Math.max(1, pax),
        valid_hours: Math.max(1, validHours),
        order_id: orderId.trim() || undefined,
      });
      const qrDataUrl = await QRCode.toDataURL(res.qr_string, {
        width: 280,
        margin: 2,
        errorCorrectionLevel: 'M',
      });
      const pass: IssuedPassItem = { response: res, qrDataUrl };
      setActivePass(pass);
      setHistory((prev) => [pass, ...prev.slice(0, 9)]);
      toast.success(`Issued ${category} pass for ${pax} pax`);
    } catch (err: any) {
      toast.error(err.message || 'Failed to issue ticket');
    } finally {
      setIsIssuing(false);
    }
  };

  const handleCopy = async () => {
    if (!activePass) return;
    await navigator.clipboard.writeText(activePass.response.qr_string);
    setCopied(true);
    toast.info('QR token copied to clipboard');
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="flex-1 flex flex-col h-full bg-slate-50 dark:bg-slate-950 p-3 sm:p-4 lg:p-6 overflow-y-auto transition-colors">
      <div className="flex items-center gap-3 mb-5">
        <div className="w-10 h-10 rounded-xl bg-violet-50 dark:bg-violet-500/10 text-violet-700 dark:text-violet-400 border border-violet-200 dark:border-violet-500/20 flex items-center justify-center shrink-0">
          <Ticket className="w-5 h-5" />
        </div>
        <div>
          <h2 className="font-bold text-slate-900 dark:text-slate-100 text-sm sm:text-base">Offline-Verifiable Ticket Issuer</h2>
          <p className="text-xs text-slate-600 dark:text-slate-400">Sign Ed25519 QR passes that gate turnstiles verify 100% offline</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-6">
        {/* Left: Issue Form */}
        <div className="bg-white dark:bg-slate-900/70 rounded-2xl border border-slate-200 dark:border-slate-800 p-4 sm:p-5 space-y-5 shadow-sm">
          <form onSubmit={handleIssueTicket} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Pass Category</label>
              <select
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-violet-500"
              >
                {CATEGORIES.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5 flex items-center gap-1.5">
                  <Users className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" />
                  Pax Count
                </label>
                <input
                  type="number"
                  min={1}
                  max={50}
                  value={pax}
                  onChange={(e) => setPax(Number(e.target.value) || 1)}
                  className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm font-mono focus:outline-none focus:border-violet-500"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5 flex items-center gap-1.5">
                  <Clock className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" />
                  Validity (Hours)
                </label>
                <input
                  type="number"
                  min={1}
                  max={720}
                  value={validHours}
                  onChange={(e) => setValidHours(Number(e.target.value) || 24)}
                  className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm font-mono focus:outline-none focus:border-violet-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Order Reference ID <span className="text-slate-500 dark:text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="text"
                value={orderId}
                onChange={(e) => setOrderId(e.target.value)}
                placeholder="e.g. ORD-001"
                className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm font-mono focus:outline-none focus:border-violet-500"
              />
            </div>

            <button
              type="submit"
              disabled={isIssuing}
              className="w-full min-h-[44px] py-3 rounded-xl bg-violet-600 hover:bg-violet-700 dark:hover:bg-violet-500 text-white font-semibold text-xs sm:text-sm flex items-center justify-center gap-2 transition-all shadow-lg shadow-violet-600/20 disabled:opacity-50"
            >
              <Sparkles className="w-4 h-4" />
              <span>{isIssuing ? 'Signing Ed25519 Pass...' : 'Issue Signed QR Ticket'}</span>
            </button>
          </form>

          {history.length > 0 && (
            <div className="pt-4 border-t border-slate-200 dark:border-slate-800 space-y-2">
              <p className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                Recent Passes ({history.length})
              </p>
              <div className="space-y-1.5 max-h-40 overflow-y-auto">
                {history.map((item) => (
                  <button
                    key={item.response.ticket.payload.tid}
                    type="button"
                    onClick={() => setActivePass(item)}
                    className={`w-full min-h-[40px] text-left px-3 py-2 rounded-xl border text-xs flex items-center justify-between transition-colors ${
                      activePass?.response.ticket.payload.tid === item.response.ticket.payload.tid
                        ? 'bg-violet-50 dark:bg-violet-500/10 border-violet-300 dark:border-violet-500/40 text-violet-900 dark:text-violet-200'
                        : 'bg-slate-50 dark:bg-slate-950 border-slate-200 dark:border-slate-800 text-slate-800 dark:text-slate-300 hover:border-slate-300 dark:hover:border-slate-700'
                    }`}
                  >
                    <span className="font-semibold truncate">{item.response.ticket.payload.cat}</span>
                    <span className="font-mono text-[11px] text-slate-600 dark:text-slate-400">
                      {item.response.ticket.payload.pax} pax
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Right: Scannable Pass Preview */}
        <div className="bg-white dark:bg-slate-900/70 rounded-2xl border border-slate-200 dark:border-slate-800 p-4 sm:p-5 flex flex-col justify-between shadow-sm">
          {!activePass ? (
            <div className="flex-1 min-h-[300px] flex flex-col items-center justify-center text-slate-600 dark:text-slate-400 text-xs sm:text-sm gap-2 text-center">
              <QrCode className="w-12 h-12 text-slate-400 dark:text-slate-600" />
              <p>Issue a ticket to generate a scannable Ed25519 QR pass</p>
            </div>
          ) : (
            <div className="space-y-5">
              <div className="bg-white border border-slate-200 text-slate-950 rounded-2xl p-6 flex flex-col items-center text-center shadow-inner print:shadow-none">
                <span className="text-[11px] font-bold tracking-widest uppercase text-violet-700">
                  {activePass.response.ticket.payload.cat}
                </span>
                <img
                  src={activePass.qrDataUrl}
                  alt="Signed Ticket QR Code"
                  className="w-48 h-48 sm:w-52 sm:h-52 my-3"
                />
                <div className="text-xs font-mono font-bold text-slate-900 break-all">
                  {activePass.response.ticket.payload.tid}
                </div>
                <div className="text-[11px] text-slate-600 mt-1">
                  Admits {activePass.response.ticket.payload.pax} Pax • Valid until{' '}
                  {new Date(activePass.response.ticket.payload.exp * 1000).toLocaleString()}
                </div>
              </div>

              <div className="flex flex-col sm:flex-row gap-2">
                <button
                  type="button"
                  onClick={handleCopy}
                  className="flex-1 min-h-[44px] px-4 py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                >
                  {copied ? <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" /> : <Copy className="w-4 h-4" />}
                  <span>{copied ? 'Copied QR String' : 'Copy QR Token'}</span>
                </button>
                <button
                  type="button"
                  onClick={() => window.print()}
                  className="flex-1 min-h-[44px] px-4 py-2.5 rounded-xl bg-violet-600 hover:bg-violet-700 dark:hover:bg-violet-500 text-white text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                >
                  <Printer className="w-4 h-4" />
                  <span>Print Pass</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
