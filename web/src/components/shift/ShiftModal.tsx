import React, { useState } from 'react';
import { X, KeyRound, ArrowDownLeft, ArrowUpRight, DollarSign, AlertTriangle, CheckCircle2, Calculator } from 'lucide-react';
import { useShiftStore } from '../../store/useShiftStore';
import { toast } from '../../store/useToastStore';
import { CashMovementType } from '../../domain/types';

interface ShiftModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ShiftModal: React.FC<ShiftModalProps> = ({ isOpen, onClose }) => {
  const { shift, openShift, recordCashMovement, closeShift, isLoading } = useShiftStore();

  const [openingFloat, setOpeningFloat] = useState(100000);
  const [cashierName, setCashierName] = useState('Cashier 1');

  // Movement Form
  const [showMovementForm, setShowMovementForm] = useState(false);
  const [movementType, setMovementType] = useState<CashMovementType>('CASH_IN');
  const [movementAmount, setMovementAmount] = useState(50000);
  const [movementReason, setMovementReason] = useState('');

  // Close Shift Form
  const [showCloseForm, setShowCloseForm] = useState(false);
  const [actualCash, setActualCash] = useState(0);
  const [closeNotes, setCloseNotes] = useState('');
  const [summaryResult, setSummaryResult] = useState<any>(null);

  if (!isOpen) return null;

  const handleOpenShift = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await openShift(openingFloat, cashierName);
      toast.success(`Shift opened for ${cashierName}`);
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const handleRecordMovement = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!movementReason.trim()) {
      toast.error('Reason is required');
      return;
    }
    try {
      await recordCashMovement(movementType, movementAmount, movementReason);
      setShowMovementForm(false);
      setMovementReason('');
      toast.success('Cash movement recorded');
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const handleCloseShift = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const summary = await closeShift(actualCash, closeNotes);
      setSummaryResult(summary);
      setShowCloseForm(false);
      toast.success('Shift closed successfully');
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const formatIDR = (val: number) => {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
    }).format(val);
  };

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-xl overflow-hidden shadow-2xl">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-indigo-500/20 text-indigo-400 flex items-center justify-center">
              <KeyRound className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-100 text-sm">Cash Drawer & Shift Session</h3>
              <p className="text-xs text-slate-400">Front-of-house cash balancing and shift audit</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {summaryResult ? (
            /* Z-Report Summary after close */
            <div className="space-y-4">
              <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-center">
                <CheckCircle2 className="w-8 h-8 text-emerald-400 mx-auto mb-2" />
                <h4 className="font-bold text-slate-100 text-base">Shift Closed Successfully (Z-Report)</h4>
                <p className="text-xs text-slate-400">Cash drawer reconciled and archived</p>
              </div>

              <div className="bg-slate-950 rounded-xl p-4 border border-slate-800 space-y-2 text-xs font-mono">
                <div className="flex justify-between text-slate-300">
                  <span>Opening Float:</span>
                  <span>{formatIDR(summaryResult.opening_float)}</span>
                </div>
                <div className="flex justify-between text-slate-300">
                  <span>Cash In (Pay-ins):</span>
                  <span>+{formatIDR(summaryResult.cash_in_total)}</span>
                </div>
                <div className="flex justify-between text-slate-300">
                  <span>Cash Out (Drops):</span>
                  <span>-{formatIDR(summaryResult.cash_out_total)}</span>
                </div>
                <div className="flex justify-between text-slate-300">
                  <span>Cash Sales:</span>
                  <span>+{formatIDR(summaryResult.cash_sales_total)}</span>
                </div>
                <div className="flex justify-between text-slate-300">
                  <span>Non-Cash Sales (QRIS/Card):</span>
                  <span>+{formatIDR(summaryResult.non_cash_sales_total)}</span>
                </div>
                <div className="border-t border-slate-800 pt-2 flex justify-between font-bold text-slate-100">
                  <span>SYSTEM EXPECTED CASH:</span>
                  <span>{formatIDR(summaryResult.expected_cash)}</span>
                </div>
                <div className="flex justify-between font-bold text-slate-100">
                  <span>ACTUAL COUNTED CASH:</span>
                  <span>{formatIDR(summaryResult.actual_cash)}</span>
                </div>
                <div
                  className={`border-t border-slate-800 pt-2 flex justify-between font-bold ${
                    summaryResult.variance === 0
                      ? 'text-emerald-400'
                      : summaryResult.variance > 0
                      ? 'text-sky-400'
                      : 'text-rose-400'
                  }`}
                >
                  <span>CASH DISCREPANCY (OVER/SHORT):</span>
                  <span>{formatIDR(summaryResult.variance)}</span>
                </div>
              </div>

              <button
                onClick={() => {
                  setSummaryResult(null);
                  onClose();
                }}
                className="w-full py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-100 font-medium text-xs transition-colors"
              >
                Done
              </button>
            </div>
          ) : !shift ? (
            /* Open Shift Screen */
            <form onSubmit={handleOpenShift} className="space-y-4">
              <div className="p-4 rounded-xl bg-slate-800/50 border border-slate-800 text-xs text-slate-300 leading-relaxed">
                Before starting sales, declare the initial cash float placed in the register drawer.
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Cashier Name / ID</label>
                <input
                  type="text"
                  value={cashierName}
                  onChange={(e) => setCashierName(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Opening Float Amount (IDR)</label>
                <div className="relative">
                  <span className="absolute left-3.5 top-2.5 text-xs text-slate-500 font-bold">Rp</span>
                  <input
                    type="number"
                    value={openingFloat}
                    onChange={(e) => setOpeningFloat(Number(e.target.value))}
                    min={0}
                    step={1000}
                    className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-indigo-500 font-mono"
                    required
                  />
                </div>
                <div className="flex gap-2 mt-2">
                  {[50000, 100000, 200000, 500000].map((amt) => (
                    <button
                      key={amt}
                      type="button"
                      onClick={() => setOpeningFloat(amt)}
                      className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-[11px] text-slate-300 transition-colors font-mono"
                    >
                      {formatIDR(amt)}
                    </button>
                  ))}
                </div>
              </div>

              <button
                type="submit"
                disabled={isLoading}
                className="w-full py-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs transition-colors shadow-lg shadow-indigo-600/20 disabled:opacity-50 mt-2"
              >
                {isLoading ? 'Opening Shift...' : 'Open Register Shift'}
              </button>
            </form>
          ) : showCloseForm ? (
            /* Close Shift Screen */
            <form onSubmit={handleCloseShift} className="space-y-4">
              <div className="p-3.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-300 flex items-start gap-2.5">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>
                  Please count all physical cash inside the register drawer and enter the actual total amount.
                </span>
              </div>

              <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 text-xs space-y-1.5">
                <div className="flex justify-between text-slate-400">
                  <span>System Expected Cash:</span>
                  <span className="font-mono text-slate-200 font-semibold">{formatIDR(shift.expected_cash)}</span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Counted Actual Cash (IDR)</label>
                <div className="relative">
                  <span className="absolute left-3.5 top-2.5 text-xs text-slate-500 font-bold">Rp</span>
                  <input
                    type="number"
                    value={actualCash}
                    onChange={(e) => setActualCash(Number(e.target.value))}
                    min={0}
                    step={1000}
                    className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-indigo-500 font-mono"
                    required
                  />
                </div>
                <div className="mt-2 text-xs flex justify-between font-mono">
                  <span className="text-slate-400">Projected Variance:</span>
                  <span
                    className={
                      actualCash - shift.expected_cash === 0
                        ? 'text-emerald-400'
                        : actualCash - shift.expected_cash > 0
                        ? 'text-sky-400'
                        : 'text-rose-400'
                    }
                  >
                    {formatIDR(actualCash - shift.expected_cash)}
                  </span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Notes (Optional)</label>
                <input
                  type="text"
                  value={closeNotes}
                  onChange={(e) => setCloseNotes(e.target.value)}
                  placeholder="e.g. End of morning shift"
                  className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCloseForm(false)}
                  className="flex-1 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium text-xs transition-colors"
                >
                  Back
                </button>
                <button
                  type="submit"
                  disabled={isLoading}
                  className="flex-1 py-2.5 rounded-xl bg-rose-600 hover:bg-rose-500 text-white font-medium text-xs transition-colors shadow-lg shadow-rose-600/20 disabled:opacity-50"
                >
                  Confirm & Close Shift
                </button>
              </div>
            </form>
          ) : showMovementForm ? (
            /* Cash Movement Form (Cash In / Drop) */
            <form onSubmit={handleRecordMovement} className="space-y-4">
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setMovementType('CASH_IN')}
                  className={`flex-1 py-2 rounded-xl border text-xs font-medium flex items-center justify-center gap-1.5 ${
                    movementType === 'CASH_IN'
                      ? 'bg-emerald-500/10 border-emerald-500/40 text-emerald-300'
                      : 'bg-slate-800 border-slate-700 text-slate-400'
                  }`}
                >
                  <ArrowDownLeft className="w-3.5 h-3.5" /> Cash In (Pay-In)
                </button>
                <button
                  type="button"
                  onClick={() => setMovementType('CASH_OUT')}
                  className={`flex-1 py-2 rounded-xl border text-xs font-medium flex items-center justify-center gap-1.5 ${
                    movementType === 'CASH_OUT'
                      ? 'bg-rose-500/10 border-rose-500/40 text-rose-300'
                      : 'bg-slate-800 border-slate-700 text-slate-400'
                  }`}
                >
                  <ArrowUpRight className="w-3.5 h-3.5" /> Cash Out (Drop / Expense)
                </button>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Amount (IDR)</label>
                <input
                  type="number"
                  value={movementAmount}
                  onChange={(e) => setMovementAmount(Number(e.target.value))}
                  min={1000}
                  step={1000}
                  className="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-indigo-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Reason / Note</label>
                <input
                  type="text"
                  value={movementReason}
                  onChange={(e) => setMovementReason(e.target.value)}
                  placeholder="e.g. Change replenishment, petty cash coffee supplies"
                  className="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500"
                  required
                />
              </div>

              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowMovementForm(false)}
                  className="flex-1 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium text-xs transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isLoading}
                  className="flex-1 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs transition-colors shadow-lg shadow-indigo-600/20"
                >
                  Save Cash Movement
                </button>
              </div>
            </form>
          ) : (
            /* Active Shift Overview */
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
                  <div className="text-[11px] text-slate-400">Cashier</div>
                  <div className="text-sm font-semibold text-slate-100 mt-0.5">{shift.cashier_name}</div>
                </div>
                <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
                  <div className="text-[11px] text-slate-400">Started At</div>
                  <div className="text-xs font-semibold text-slate-100 mt-0.5 font-mono">
                    {new Date(shift.opened_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  </div>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2.5 text-xs">
                <div className="flex justify-between text-slate-400">
                  <span>Opening Float:</span>
                  <span className="font-mono text-slate-200">{formatIDR(shift.opening_float)}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Cash In Total:</span>
                  <span className="font-mono text-emerald-400">+{formatIDR(shift.cash_in_total)}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Cash Out Total:</span>
                  <span className="font-mono text-rose-400">-{formatIDR(shift.cash_out_total)}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Cash Sales:</span>
                  <span className="font-mono text-emerald-400">+{formatIDR(shift.cash_sales_total)}</span>
                </div>
                <div className="flex justify-between text-slate-400">
                  <span>Non-Cash Sales (QRIS/Card):</span>
                  <span className="font-mono text-slate-200">+{formatIDR(shift.non_cash_sales_total)}</span>
                </div>
                <div className="border-t border-slate-800 pt-2 flex justify-between font-bold text-slate-100">
                  <span className="flex items-center gap-1.5">
                    <Calculator className="w-3.5 h-3.5 text-indigo-400" />
                    EXPECTED DRAWER CASH:
                  </span>
                  <span className="text-indigo-300 font-mono text-sm">{formatIDR(shift.expected_cash)}</span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 pt-2">
                <button
                  onClick={() => setShowMovementForm(true)}
                  className="py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700/80 text-slate-200 border border-slate-700 text-xs font-medium flex items-center justify-center gap-2 transition-colors"
                >
                  <DollarSign className="w-3.5 h-3.5 text-indigo-400" />
                  Cash In / Out
                </button>
                <button
                  onClick={() => {
                    setActualCash(shift.expected_cash);
                    setShowCloseForm(true);
                  }}
                  className="py-2.5 rounded-xl bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/30 text-xs font-medium flex items-center justify-center gap-2 transition-colors"
                >
                  Close Shift (Z-Report)
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
