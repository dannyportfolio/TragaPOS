import React, { useEffect, useState } from 'react';
import {
  BedDouble,
  Plus,
  Search,
  RefreshCw,
  CheckCircle2,
  X,
  CreditCard,
  UserCheck,
} from 'lucide-react';
import { GuestFolio } from '../../domain/types';
import { api } from '../../services/api';
import { toast } from '../../store/useToastStore';

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(amount);
}

export const FoliosView: React.FC = () => {
  const [folios, setFolios] = useState<GuestFolio[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [openOnly, setOpenOnly] = useState(true);
  const [search, setSearch] = useState('');

  // Create Folio Modal
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [roomNumber, setRoomNumber] = useState('');
  const [guestName, setGuestName] = useState('');
  const [creditLimit, setCreditLimit] = useState('5000000');
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Settle Confirmation
  const [settlingFolio, setSettlingFolio] = useState<GuestFolio | null>(null);

  const loadFolios = async () => {
    setIsLoading(true);
    try {
      const data = await api.getFolios(openOnly, 100, 0);
      setFolios(data);
    } catch (err: any) {
      toast.error(err.message || 'Failed to load guest folios');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadFolios();
  }, [openOnly]);

  const handleCreateFolio = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!roomNumber.trim() || !guestName.trim()) return;
    setIsSubmitting(true);
    try {
      await api.createFolio({
        room_number: roomNumber.trim(),
        guest_name: guestName.trim(),
        credit_limit: Number(creditLimit) || 0,
      });
      toast.success(`Folio opened for Room ${roomNumber.trim()} (${guestName.trim()})`);
      setIsCreateOpen(false);
      setRoomNumber('');
      setGuestName('');
      setCreditLimit('5000000');
      await loadFolios();
    } catch (err: any) {
      toast.error(err.message || 'Failed to create folio');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleConfirmSettle = async () => {
    if (!settlingFolio) return;
    setIsSubmitting(true);
    try {
      await api.settleFolio(settlingFolio.id);
      toast.success(`Room ${settlingFolio.room_number} folio settled`);
      setSettlingFolio(null);
      await loadFolios();
    } catch (err: any) {
      toast.error(err.message || 'Failed to settle folio');
    } finally {
      setIsSubmitting(false);
    }
  };

  const filtered = folios.filter((f) => {
    const q = search.trim().toLowerCase();
    if (!q) return true;
    return (
      f.room_number.toLowerCase().includes(q) ||
      f.guest_name.toLowerCase().includes(q)
    );
  });

  return (
    <div className="flex-1 flex flex-col h-full bg-slate-50 dark:bg-slate-950 p-3 sm:p-4 lg:p-6 overflow-hidden transition-colors">
      {/* Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4 sm:mb-5">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-teal-50 dark:bg-teal-500/10 text-teal-700 dark:text-teal-400 border border-teal-200 dark:border-teal-500/20 flex items-center justify-center shrink-0">
            <BedDouble className="w-5 h-5" />
          </div>
          <div>
            <h2 className="font-bold text-slate-900 dark:text-slate-100 text-sm sm:text-base">Hospitality Guest Folios</h2>
            <p className="text-xs text-slate-600 dark:text-slate-400">Room charge tabs, credit limits &amp; front-desk settlement</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={loadFolios}
            disabled={isLoading}
            className="min-h-[44px] px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-800 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-semibold flex items-center gap-1.5 transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
          <button
            type="button"
            onClick={() => setIsCreateOpen(true)}
            className="min-h-[44px] px-4 py-2 rounded-xl bg-teal-600 hover:bg-teal-700 dark:hover:bg-teal-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-all shadow-lg shadow-teal-600/20"
          >
            <Plus className="w-4 h-4" />
            <span>Check-In / Open Folio</span>
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3 mb-4">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500 dark:text-slate-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by room number or guest name..."
            className="w-full min-h-[44px] pl-10 pr-4 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-teal-500"
          />
        </div>

        <div className="flex bg-white dark:bg-slate-900 p-1 rounded-xl border border-slate-300 dark:border-slate-800 self-start">
          <button
            type="button"
            onClick={() => setOpenOnly(true)}
            className={`min-h-[36px] px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
              openOnly ? 'bg-teal-600 text-white' : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
            }`}
          >
            Open Folios
          </button>
          <button
            type="button"
            onClick={() => setOpenOnly(false)}
            className={`min-h-[36px] px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-colors ${
              !openOnly ? 'bg-teal-600 text-white' : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
            }`}
          >
            All Folios
          </button>
        </div>
      </div>

      {/* Folio Cards Grid */}
      <div className="flex-1 overflow-y-auto pr-1">
        {filtered.length === 0 ? (
          <div className="h-60 flex flex-col items-center justify-center text-slate-600 dark:text-slate-400 text-xs sm:text-sm gap-2 border-2 border-dashed border-slate-300 dark:border-slate-800 rounded-2xl bg-white/60 dark:bg-slate-900/40">
            <BedDouble className="w-8 h-8 text-slate-400 dark:text-slate-600" />
            <span>{isLoading ? 'Loading guest folios...' : 'No matching guest folios found'}</span>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 pb-6">
            {filtered.map((folio) => {
              const pct =
                folio.credit_limit > 0
                  ? Math.min(100, Math.round((folio.current_total / folio.credit_limit) * 100))
                  : 0;
              return (
                <div
                  key={folio.id}
                  className="p-4 rounded-2xl bg-white dark:bg-slate-900/70 border border-slate-200 dark:border-slate-800 flex flex-col justify-between gap-4 hover:border-slate-300 dark:hover:border-slate-700 transition-colors shadow-sm"
                >
                  <div className="space-y-3">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-teal-50 dark:bg-slate-950 border border-teal-200 dark:border-slate-800 text-xs font-mono font-bold text-teal-800 dark:text-teal-400">
                          Room {folio.room_number}
                        </span>
                        <h3 className="mt-2 font-semibold text-slate-900 dark:text-slate-100 text-sm flex items-center gap-1.5">
                          <UserCheck className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                          {folio.guest_name}
                        </h3>
                      </div>
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border ${
                          folio.is_open
                            ? 'bg-emerald-50 text-emerald-800 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/30'
                            : 'bg-slate-100 text-slate-700 border-slate-300 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700'
                        }`}
                      >
                        {folio.is_open ? 'OPEN' : 'SETTLED'}
                      </span>
                    </div>

                    <div className="space-y-1.5 pt-1">
                      <div className="flex justify-between text-xs">
                        <span className="text-slate-600 dark:text-slate-400">Current Tab</span>
                        <span className="font-mono font-bold text-slate-900 dark:text-slate-100">
                          {formatCurrency(folio.current_total)}
                        </span>
                      </div>
                      <div className="flex justify-between text-[11px]">
                        <span className="text-slate-600 dark:text-slate-400">Credit Limit</span>
                        <span className="font-mono text-slate-700 dark:text-slate-300">
                          {formatCurrency(folio.credit_limit)}
                        </span>
                      </div>
                      <div className="w-full h-2 rounded-full bg-slate-100 dark:bg-slate-950 overflow-hidden border border-slate-200 dark:border-slate-800">
                        <div
                          className={`h-full transition-all ${
                            pct >= 90 ? 'bg-rose-500' : pct >= 70 ? 'bg-amber-500' : 'bg-teal-500'
                          }`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  </div>

                  {folio.is_open && (
                    <button
                      type="button"
                      onClick={() => setSettlingFolio(folio)}
                      className="w-full min-h-[44px] py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-teal-600 text-slate-800 dark:text-slate-200 hover:text-white border border-slate-300 dark:border-slate-700 hover:border-teal-600 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                    >
                      <CreditCard className="w-4 h-4" />
                      <span>Settle &amp; Check Out</span>
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Create Folio Modal */}
      {isCreateOpen && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md max-h-[90dvh] overflow-y-auto shadow-2xl">
            <div className="px-5 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
              <h3 className="font-semibold text-slate-900 dark:text-slate-100 text-sm">Open Guest Folio</h3>
              <button
                type="button"
                onClick={() => setIsCreateOpen(false)}
                aria-label="Close modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <form onSubmit={handleCreateFolio} className="p-5 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Room / Villa Number</label>
                <input
                  type="text"
                  required
                  value={roomNumber}
                  onChange={(e) => setRoomNumber(e.target.value)}
                  placeholder="e.g. 104 or VILLA-02"
                  className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-teal-500"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Guest Full Name</label>
                <input
                  type="text"
                  required
                  value={guestName}
                  onChange={(e) => setGuestName(e.target.value)}
                  placeholder="e.g. Budi Santoso"
                  className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-teal-500"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Room Charge Credit Limit (IDR)</label>
                <input
                  type="number"
                  min={0}
                  step={50000}
                  required
                  value={creditLimit}
                  onChange={(e) => setCreditLimit(e.target.value)}
                  className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm font-mono focus:outline-none focus:border-teal-500"
                />
              </div>
              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(false)}
                  className="flex-1 min-h-[44px] rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-xs font-semibold"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="flex-1 min-h-[44px] rounded-xl bg-teal-600 hover:bg-teal-700 dark:hover:bg-teal-500 text-white text-xs font-semibold disabled:opacity-50"
                >
                  {isSubmitting ? 'Opening...' : 'Open Folio'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Settle Confirmation Modal */}
      {settlingFolio && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-sm p-5 space-y-4 shadow-2xl">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-teal-50 dark:bg-teal-500/10 text-teal-700 dark:text-teal-400 flex items-center justify-center">
                <CheckCircle2 className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 text-sm">Settle Room {settlingFolio.room_number}?</h3>
                <p className="text-xs text-slate-600 dark:text-slate-400">{settlingFolio.guest_name}</p>
              </div>
            </div>
            <p className="text-xs text-slate-700 dark:text-slate-300 bg-slate-50 dark:bg-slate-950 p-3 rounded-xl border border-slate-200 dark:border-slate-800">
              Total balance due: <strong className="font-mono text-teal-700 dark:text-teal-400">{formatCurrency(settlingFolio.current_total)}</strong>
            </p>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setSettlingFolio(null)}
                className="flex-1 min-h-[44px] rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmSettle}
                disabled={isSubmitting}
                className="flex-1 min-h-[44px] rounded-xl bg-teal-600 hover:bg-teal-700 dark:hover:bg-teal-500 text-white text-xs font-semibold disabled:opacity-50"
              >
                {isSubmitting ? 'Settling...' : 'Confirm Settle'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
