import React, { useState } from 'react';
import { X, CreditCard, Banknote, QrCode, Building, Plus, Trash2, CheckCircle2, AlertCircle } from 'lucide-react';
import { PaymentMethod, PaymentInput, Order } from '../../domain/types';
import { useCartStore } from '../../store/useCartStore';
import { api } from '../../services/api';

interface PaymentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (order: Order) => void;
}

export const PaymentModal: React.FC<PaymentModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const { items, heldOrderId, notes, getTotalAmount, clearCart } = useCartStore();
  const totalAmount = getTotalAmount();

  const [isSplitMode, setIsSplitMode] = useState(false);
  const [singleMethod, setSingleMethod] = useState<PaymentMethod>('CASH');
  const [cashTendered, setCashTendered] = useState(totalAmount);
  const [roomNumber, setRoomNumber] = useState('');
  const [cardReference, setCardReference] = useState('');

  // Split tenders
  const [splitTenders, setSplitTenders] = useState<PaymentInput[]>([
    { method: 'CASH', amount: Math.floor(totalAmount / 2) },
    { method: 'QRIS', amount: totalAmount - Math.floor(totalAmount / 2) },
  ]);

  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  if (!isOpen) return null;

  const splitTotal = splitTenders.reduce((sum, t) => sum + t.amount, 0);
  const remainingDue = totalAmount - splitTotal;

  const formatIDR = (val: number) => {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
    }).format(val);
  };

  const handleQuickCash = (amt: number) => {
    setCashTendered(amt);
  };

  const addSplitTender = (method: PaymentMethod) => {
    const defaultAmt = remainingDue > 0 ? remainingDue : 10000;
    setSplitTenders([...splitTenders, { method, amount: defaultAmt }]);
  };

  const removeSplitTender = (idx: number) => {
    setSplitTenders(splitTenders.filter((_, i) => i !== idx));
  };

  const updateSplitAmount = (idx: number, amt: number) => {
    const next = [...splitTenders];
    next[idx] = { ...next[idx], amount: amt };
    setSplitTenders(next);
  };

  const handleSubmit = async () => {
    setErrorMsg(null);
    setIsLoading(true);

    try {
      let paymentsPayload: PaymentInput[];

      if (isSplitMode) {
        if (splitTotal !== totalAmount) {
          throw new Error(`Split total (${formatIDR(splitTotal)}) must equal order total (${formatIDR(totalAmount)})`);
        }
        paymentsPayload = splitTenders;
      } else {
        if (singleMethod === 'CASH' && cashTendered < totalAmount) {
          throw new Error(`Cash tendered is less than total amount due`);
        }

        paymentsPayload = [
          {
            method: singleMethod,
            amount: singleMethod === 'CASH' ? cashTendered : totalAmount,
            reference: singleMethod === 'CARD' ? cardReference : singleMethod === 'QRIS' ? 'QRIS-DYNAMIC' : undefined,
            room_number: singleMethod === 'ROOM_CHARGE' ? roomNumber : undefined,
          },
        ];
      }

      const checkoutPayload = {
        order_id: heldOrderId,
        tenant_id: '',
        outlet_id: '',
        notes,
        items: items.map((ci) => ({
          sku: ci.item.sku,
          barcode: ci.item.barcode,
          quantity: ci.quantity,
          selected_modifiers: ci.selectedModifiers,
          sector_payload: ci.sectorPayload,
        })),
        payments: paymentsPayload,
      };

      const order = await api.checkout(checkoutPayload);
      clearCart();
      onSuccess(order);
      onClose();
    } catch (err: any) {
      setErrorMsg(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-xl overflow-hidden shadow-2xl">
        {/* Modal Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-slate-100 text-sm">Tender Payment & Settlement</h3>
            <p className="text-xs text-slate-400">Total Due: <span className="text-indigo-400 font-mono font-bold">{formatIDR(totalAmount)}</span></p>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {errorMsg && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs text-rose-300 flex items-start gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          {/* Mode Switch (Single vs Split Tender) */}
          <div className="flex gap-2 p-1 bg-slate-950 rounded-xl border border-slate-800">
            <button
              type="button"
              onClick={() => setIsSplitMode(false)}
              className={`flex-1 py-1.5 rounded-lg text-xs font-medium transition-all ${
                !isSplitMode ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Single Tender
            </button>
            <button
              type="button"
              onClick={() => setIsSplitMode(true)}
              className={`flex-1 py-1.5 rounded-lg text-xs font-medium transition-all ${
                isSplitMode ? 'bg-indigo-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Split-Tender (Multi-Payment)
            </button>
          </div>

          {!isSplitMode ? (
            /* Single Tender Mode */
            <div className="space-y-4">
              {/* Method Selection Grid */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {[
                  { id: 'CASH', label: 'Cash', icon: Banknote },
                  { id: 'QRIS', label: 'QRIS', icon: QrCode },
                  { id: 'CARD', label: 'Card / EDC', icon: CreditCard },
                  { id: 'ROOM_CHARGE', label: 'Room Charge', icon: Building },
                ].map((m) => {
                  const Icon = m.icon;
                  const isSelected = singleMethod === m.id;
                  return (
                    <button
                      key={m.id}
                      type="button"
                      onClick={() => setSingleMethod(m.id as PaymentMethod)}
                      className={`p-3 rounded-xl border flex flex-col items-center gap-1.5 text-xs font-medium transition-all ${
                        isSelected
                          ? 'bg-indigo-600/10 border-indigo-500 text-indigo-200 shadow-sm'
                          : 'bg-slate-950/60 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
                      }`}
                    >
                      <Icon className="w-5 h-5" />
                      <span>{m.label}</span>
                    </button>
                  );
                })}
              </div>

              {/* Method Specific Inputs */}
              {singleMethod === 'CASH' && (
                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
                  <div className="flex justify-between items-center text-xs">
                    <span className="text-slate-400 font-medium">Cash Tendered:</span>
                    <input
                      type="number"
                      value={cashTendered}
                      onChange={(e) => setCashTendered(Number(e.target.value))}
                      step={1000}
                      className="w-40 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-right font-mono font-bold text-slate-100 text-sm focus:outline-none focus:border-indigo-500"
                    />
                  </div>

                  {/* Quick Cash Buttons */}
                  <div className="flex flex-wrap gap-1.5">
                    {[
                      { label: 'Exact', amt: totalAmount },
                      { label: '+50k', amt: Math.ceil(totalAmount / 50000) * 50000 },
                      { label: '100,000', amt: 100000 },
                      { label: '200,000', amt: 200000 },
                    ].map((btn, idx) => (
                      <button
                        key={idx}
                        type="button"
                        onClick={() => handleQuickCash(btn.amt)}
                        className="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-mono transition-colors"
                      >
                        {btn.label}
                      </button>
                    ))}
                  </div>

                  {/* Change calculation */}
                  <div className="border-t border-slate-800 pt-2 flex justify-between items-center text-xs">
                    <span className="text-slate-400">Change Due:</span>
                    <span className="font-mono font-bold text-emerald-400 text-sm">
                      {formatIDR(Math.max(0, cashTendered - totalAmount))}
                    </span>
                  </div>
                </div>
              )}

              {singleMethod === 'ROOM_CHARGE' && (
                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
                  <label className="block text-xs font-medium text-slate-300">Guest Room Number</label>
                  <input
                    type="text"
                    value={roomNumber}
                    onChange={(e) => setRoomNumber(e.target.value)}
                    placeholder="e.g. 101, VILLA-2"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-900 border border-slate-700 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 font-mono"
                    required
                  />
                  <p className="text-[11px] text-slate-500">
                    Will check folio balance and credit limit automatically upon submission.
                  </p>
                </div>
              )}

              {singleMethod === 'CARD' && (
                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
                  <label className="block text-xs font-medium text-slate-300">EDC Approval Code / Trace</label>
                  <input
                    type="text"
                    value={cardReference}
                    onChange={(e) => setCardReference(e.target.value)}
                    placeholder="e.g. 982144"
                    className="w-full px-3.5 py-2.5 rounded-xl bg-slate-900 border border-slate-700 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 font-mono"
                  />
                </div>
              )}

              {singleMethod === 'QRIS' && (
                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 text-center space-y-2">
                  <QrCode className="w-12 h-12 text-indigo-400 mx-auto" />
                  <p className="text-xs text-slate-300 font-medium">Dynamic QRIS Ready for Settlement</p>
                  <p className="text-[11px] text-slate-500">Simulating instant local settlement for offline/online hub</p>
                </div>
              )}
            </div>
          ) : (
            /* Split Tender Mode */
            <div className="space-y-3">
              <div className="space-y-2 max-h-56 overflow-y-auto pr-1">
                {splitTenders.map((tender, idx) => (
                  <div
                    key={idx}
                    className="p-3 rounded-xl bg-slate-950 border border-slate-800 flex items-center gap-2"
                  >
                    <select
                      value={tender.method}
                      onChange={(e) => {
                        const next = [...splitTenders];
                        next[idx].method = e.target.value as PaymentMethod;
                        setSplitTenders(next);
                      }}
                      className="px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-slate-200 text-xs focus:outline-none"
                    >
                      <option value="CASH">Cash</option>
                      <option value="QRIS">QRIS</option>
                      <option value="CARD">Card</option>
                      <option value="ROOM_CHARGE">Room Charge</option>
                    </select>

                    <div className="relative flex-1">
                      <span className="absolute left-2.5 top-2 text-[10px] text-slate-500 font-mono">Rp</span>
                      <input
                        type="number"
                        value={tender.amount}
                        onChange={(e) => updateSplitAmount(idx, Number(e.target.value))}
                        step={1000}
                        min={0}
                        className="w-full pl-8 pr-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-slate-100 text-xs font-mono font-bold text-right focus:outline-none focus:border-indigo-500"
                      />
                    </div>

                    {tender.method === 'ROOM_CHARGE' && (
                      <input
                        type="text"
                        value={tender.room_number || ''}
                        onChange={(e) => {
                          const next = [...splitTenders];
                          next[idx].room_number = e.target.value;
                          setSplitTenders(next);
                        }}
                        placeholder="Room #"
                        className="w-20 px-2 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-slate-100 text-xs font-mono focus:outline-none"
                      />
                    )}

                    <button
                      type="button"
                      onClick={() => removeSplitTender(idx)}
                      className="text-slate-600 hover:text-rose-400 p-1"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>

              {/* Add Split Tender buttons */}
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => addSplitTender('CASH')}
                  className="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium flex items-center gap-1 transition-colors"
                >
                  <Plus className="w-3.5 h-3.5" /> + Cash
                </button>
                <button
                  type="button"
                  onClick={() => addSplitTender('QRIS')}
                  className="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium flex items-center gap-1 transition-colors"
                >
                  <Plus className="w-3.5 h-3.5" /> + QRIS
                </button>
                <button
                  type="button"
                  onClick={() => addSplitTender('CARD')}
                  className="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium flex items-center gap-1 transition-colors"
                >
                  <Plus className="w-3.5 h-3.5" /> + Card
                </button>
              </div>

              {/* Split Balance Overview */}
              <div className="p-3 rounded-xl bg-slate-950 border border-slate-800 text-xs space-y-1 font-mono">
                <div className="flex justify-between text-slate-400">
                  <span>Split Tendered Total:</span>
                  <span className="text-slate-200 font-bold">{formatIDR(splitTotal)}</span>
                </div>
                <div
                  className={`flex justify-between font-bold ${
                    remainingDue === 0 ? 'text-emerald-400' : 'text-amber-400'
                  }`}
                >
                  <span>Remaining Due:</span>
                  <span>{formatIDR(remainingDue)}</span>
                </div>
              </div>
            </div>
          )}

          {/* Submit Action */}
          <button
            onClick={handleSubmit}
            disabled={isLoading || (isSplitMode && remainingDue !== 0)}
            className="w-full py-3 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg shadow-emerald-600/20 disabled:opacity-50 flex items-center justify-center gap-2"
          >
            <CheckCircle2 className="w-4 h-4" />
            <span>{isLoading ? 'Processing Checkout...' : `Complete Payment (${formatIDR(totalAmount)})`}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
