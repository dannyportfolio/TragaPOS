import React from 'react';
import { ShoppingCart, Plus, Minus, Trash2, PauseCircle, CreditCard, FileText } from 'lucide-react';
import { useCartStore } from '../../store/useCartStore';

interface CartDrawerProps {
  onOpenPaymentModal: () => void;
  onHoldOrder: () => void;
  isHolding: boolean;
}

export const CartDrawer: React.FC<CartDrawerProps> = ({
  onOpenPaymentModal,
  onHoldOrder,
  isHolding,
}) => {
  const {
    items,
    heldOrderId,
    notes,
    updateQuantity,
    removeItem,
    clearCart,
    setNotes,
    getSubtotal,
    getTaxAmount,
    getTotalAmount,
  } = useCartStore();

  const subtotal = getSubtotal();
  const taxAmount = getTaxAmount();
  const totalAmount = getTotalAmount();

  const formatIDR = (val: number) => {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
    }).format(val);
  };

  return (
    <div className="w-80 md:w-96 bg-slate-900 border-l border-slate-800 flex flex-col h-full select-none">
      {/* Cart Header */}
      <div className="px-5 py-4 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ShoppingCart className="w-4 h-4 text-indigo-400" />
          <h3 className="font-semibold text-slate-100 text-sm">Active Register Cart</h3>
          <span className="w-5 h-5 rounded-full bg-slate-800 text-slate-300 font-mono text-[11px] font-bold flex items-center justify-center">
            {items.reduce((sum, i) => sum + i.quantity, 0)}
          </span>
        </div>
        {items.length > 0 && (
          <button
            onClick={clearCart}
            className="text-[11px] text-slate-500 hover:text-rose-400 font-medium transition-colors"
          >
            Clear All
          </button>
        )}
      </div>

      {heldOrderId && (
        <div className="px-5 py-2 bg-amber-500/10 border-b border-amber-500/20 text-xs text-amber-300 flex items-center justify-between">
          <span>Resumed Parked Order</span>
          <span className="font-mono text-[10px] text-amber-400 font-bold">{heldOrderId.substring(0, 8)}</span>
        </div>
      )}

      {/* Cart Item List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2.5">
        {items.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-slate-500 text-xs text-center p-6">
            <ShoppingCart className="w-8 h-8 text-slate-700 mb-2" />
            <p className="font-medium text-slate-400">Cart is empty</p>
            <p className="text-[11px] text-slate-600 mt-0.5">Click catalog items or scan barcode to add</p>
          </div>
        ) : (
          items.map((cartItem, idx) => (
            <div
              key={`${cartItem.item.id}-${idx}`}
              className="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80 space-y-2"
            >
              <div className="flex items-start justify-between gap-2">
                <div>
                  <h4 className="text-xs font-medium text-slate-100">{cartItem.item.name}</h4>
                  <div className="text-[10px] text-slate-500 font-mono">
                    {cartItem.item.sku} • {formatIDR(cartItem.unitPrice)}
                  </div>
                </div>
                <button
                  onClick={() => removeItem(idx)}
                  className="text-slate-600 hover:text-rose-400 transition-colors p-1"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>

              {/* Modifiers List */}
              {cartItem.selectedModifiers.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {cartItem.selectedModifiers.map((mod, mIdx) => (
                    <span
                      key={mIdx}
                      className="px-1.5 py-0.5 rounded text-[10px] bg-slate-800 text-slate-300 font-mono"
                    >
                      +{mod.name}
                    </span>
                  ))}
                </div>
              )}

              {/* Quantity Stepper & Subtotal */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-2 bg-slate-900 border border-slate-800 rounded-lg p-0.5">
                  <button
                    onClick={() => updateQuantity(idx, -1)}
                    className="w-6 h-6 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 flex items-center justify-center transition-colors"
                  >
                    <Minus className="w-3 h-3" />
                  </button>
                  <span className="text-xs font-mono font-bold text-slate-100 px-1.5">
                    {cartItem.quantity}
                  </span>
                  <button
                    onClick={() => updateQuantity(idx, 1)}
                    className="w-6 h-6 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 flex items-center justify-center transition-colors"
                  >
                    <Plus className="w-3 h-3" />
                  </button>
                </div>
                <div className="text-xs font-mono font-bold text-slate-200">
                  {formatIDR(cartItem.totalAmount)}
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Cart Summary & Actions */}
      <div className="p-4 border-t border-slate-800 bg-slate-900 space-y-3">
        {/* Notes Input */}
        <div className="relative">
          <FileText className="w-3.5 h-3.5 absolute left-3 top-2.5 text-slate-500" />
          <input
            type="text"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Order notes / Table number..."
            className="w-full pl-8 pr-3 py-1.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-200 text-xs focus:outline-none focus:border-indigo-500 placeholder:text-slate-600"
          />
        </div>

        {/* Totals Breakdown */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-slate-400">
            <span>Subtotal:</span>
            <span className="font-mono text-slate-200">{formatIDR(subtotal)}</span>
          </div>
          {taxAmount > 0 && (
            <div className="flex justify-between text-slate-400">
              <span>Tax (PPN):</span>
              <span className="font-mono text-slate-200">{formatIDR(taxAmount)}</span>
            </div>
          )}
          <div className="border-t border-slate-800 pt-1.5 flex justify-between font-bold text-sm text-slate-100">
            <span>TOTAL DUE:</span>
            <span className="text-indigo-400 font-mono text-base">{formatIDR(totalAmount)}</span>
          </div>
        </div>

        {/* Buttons */}
        <div className="grid grid-cols-3 gap-2 pt-1">
          <button
            onClick={onHoldOrder}
            disabled={items.length === 0 || isHolding}
            className="py-2.5 rounded-xl bg-slate-800 hover:bg-slate-750 text-slate-300 border border-slate-700 text-xs font-medium flex items-center justify-center gap-1.5 transition-colors disabled:opacity-50"
            title="Park order to serve next customer"
          >
            <PauseCircle className="w-3.5 h-3.5 text-amber-400" />
            <span>Park</span>
          </button>

          <button
            onClick={onOpenPaymentModal}
            disabled={items.length === 0}
            className="col-span-2 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center justify-center gap-2 transition-all shadow-lg shadow-indigo-600/25 disabled:opacity-50"
          >
            <CreditCard className="w-4 h-4" />
            <span>Checkout / Pay</span>
          </button>
        </div>
      </div>
    </div>
  );
};
