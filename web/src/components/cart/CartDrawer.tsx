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
    <div className="w-full lg:w-80 xl:w-96 bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800 flex flex-col h-full select-none transition-colors">
      {/* Cart Header */}
      <div className="px-4 sm:px-5 py-3.5 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between pr-14 lg:pr-5">
        <div className="flex items-center gap-2">
          <ShoppingCart className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
          <h3 className="font-semibold text-slate-900 dark:text-slate-100 text-sm">Active Register Cart</h3>
          <span className="w-5 h-5 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200 font-mono text-[11px] font-bold flex items-center justify-center border border-slate-200 dark:border-slate-700">
            {items.reduce((sum, i) => sum + i.quantity, 0)}
          </span>
        </div>
        {items.length > 0 && (
          <button
            type="button"
            onClick={clearCart}
            className="min-h-[36px] px-2 rounded-lg text-xs text-slate-600 dark:text-slate-400 hover:text-rose-700 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 font-medium transition-colors"
          >
            Clear All
          </button>
        )}
      </div>

      {heldOrderId && (
        <div className="px-4 sm:px-5 py-2 bg-amber-50 dark:bg-amber-500/10 border-b border-amber-200 dark:border-amber-500/20 text-xs text-amber-900 dark:text-amber-300 flex items-center justify-between">
          <span className="font-medium">Resumed Parked Order</span>
          <span className="font-mono text-[11px] text-amber-800 dark:text-amber-400 font-bold">{heldOrderId.substring(0, 8)}</span>
        </div>
      )}

      {/* Cart Item List */}
      <div className="flex-1 overflow-y-auto p-3.5 sm:p-4 space-y-2.5">
        {items.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-slate-600 dark:text-slate-400 text-xs text-center p-6">
            <ShoppingCart className="w-8 h-8 text-slate-400 dark:text-slate-600 mb-2" />
            <p className="font-semibold text-slate-800 dark:text-slate-300 text-sm">Cart is empty</p>
            <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">Click catalog items or scan barcode to add</p>
          </div>
        ) : (
          items.map((cartItem, idx) => (
            <div
              key={`${cartItem.item.id}-${idx}`}
              className="p-3 rounded-xl bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800/80 space-y-2"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <h4 className="text-xs sm:text-sm font-semibold text-slate-900 dark:text-slate-100 truncate">{cartItem.item.name}</h4>
                  <div className="text-[11px] text-slate-600 dark:text-slate-400 font-mono">
                    {cartItem.item.sku} • {formatIDR(cartItem.unitPrice)}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => removeItem(idx)}
                  aria-label={`Remove ${cartItem.item.name}`}
                  className="min-w-[36px] min-h-[36px] rounded-lg text-slate-500 dark:text-slate-400 hover:text-rose-700 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors flex items-center justify-center shrink-0"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>

              {/* Modifiers List */}
              {cartItem.selectedModifiers.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {cartItem.selectedModifiers.map((mod, mIdx) => (
                    <span
                      key={mIdx}
                      className="px-1.5 py-0.5 rounded text-[10px] bg-slate-200 dark:bg-slate-800 text-slate-800 dark:text-slate-300 font-mono"
                    >
                      +{mod.name}
                    </span>
                  ))}
                </div>
              )}

              {/* Quantity Stepper & Subtotal */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-1.5 bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 rounded-lg p-0.5">
                  <button
                    type="button"
                    onClick={() => updateQuantity(idx, -1)}
                    aria-label={`Decrease quantity of ${cartItem.item.name}`}
                    className="w-8 h-8 rounded-md bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 flex items-center justify-center transition-colors"
                  >
                    <Minus className="w-3.5 h-3.5" />
                  </button>
                  <span className="text-xs font-mono font-bold text-slate-900 dark:text-slate-100 px-2">
                    {cartItem.quantity}
                  </span>
                  <button
                    type="button"
                    onClick={() => updateQuantity(idx, 1)}
                    aria-label={`Increase quantity of ${cartItem.item.name}`}
                    className="w-8 h-8 rounded-md bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 flex items-center justify-center transition-colors"
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </button>
                </div>
                <div className="text-xs sm:text-sm font-mono font-bold text-slate-900 dark:text-slate-100">
                  {formatIDR(cartItem.totalAmount)}
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Cart Summary & Actions */}
      <div className="p-4 border-t border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 space-y-3">
        {/* Notes Input */}
        <div className="relative">
          <FileText className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500 dark:text-slate-400 pointer-events-none" />
          <input
            type="text"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Order notes / Table number..."
            aria-label="Order notes"
            className="w-full min-h-[40px] pl-8 pr-3 py-1.5 rounded-lg bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs focus:outline-none focus:border-indigo-500 placeholder:text-slate-500"
          />
        </div>

        {/* Totals Breakdown */}
        <div className="space-y-1.5 text-xs">
          <div className="flex justify-between text-slate-600 dark:text-slate-400">
            <span>Subtotal:</span>
            <span className="font-mono font-medium text-slate-900 dark:text-slate-200">{formatIDR(subtotal)}</span>
          </div>
          {taxAmount > 0 && (
            <div className="flex justify-between text-slate-600 dark:text-slate-400">
              <span>Tax (PPN):</span>
              <span className="font-mono font-medium text-slate-900 dark:text-slate-200">{formatIDR(taxAmount)}</span>
            </div>
          )}
          <div className="border-t border-slate-200 dark:border-slate-800 pt-2 flex justify-between font-bold text-sm text-slate-900 dark:text-slate-100">
            <span>TOTAL DUE:</span>
            <span className="text-indigo-700 dark:text-indigo-400 font-mono text-base">{formatIDR(totalAmount)}</span>
          </div>
        </div>

        {/* Buttons */}
        <div className="grid grid-cols-3 gap-2 pt-1">
          <button
            type="button"
            onClick={onHoldOrder}
            disabled={items.length === 0 || isHolding}
            className="min-h-[44px] py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700 text-xs font-semibold flex items-center justify-center gap-1.5 transition-colors disabled:opacity-50"
            title="Park order to serve next customer"
          >
            <PauseCircle className="w-4 h-4 text-amber-600 dark:text-amber-400" />
            <span>{isHolding ? 'Saving...' : 'Park'}</span>
          </button>

          <button
            type="button"
            onClick={onOpenPaymentModal}
            disabled={items.length === 0}
            className="col-span-2 min-h-[44px] py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 dark:hover:bg-indigo-500 text-white font-semibold text-xs sm:text-sm flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/20 transition-all disabled:opacity-50"
          >
            <CreditCard className="w-4 h-4" />
            <span>Pay Now</span>
          </button>
        </div>
      </div>
    </div>
  );
};
