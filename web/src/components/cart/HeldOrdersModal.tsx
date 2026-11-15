import React, { useEffect, useState } from 'react';
import { X, PauseCircle, PlayCircle, Clock } from 'lucide-react';
import { Order } from '../../domain/types';
import { api } from '../../services/api';
import { useCartStore } from '../../store/useCartStore';

interface HeldOrdersModalProps {
  isOpen: boolean;
  onClose: () => void;
  onHeldCountChange: (count: number) => void;
}

export const HeldOrdersModal: React.FC<HeldOrdersModalProps> = ({
  isOpen,
  onClose,
  onHeldCountChange,
}) => {
  const [heldOrders, setHeldOrders] = useState<Order[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { loadHeldOrder } = useCartStore();

  const loadOrders = async () => {
    setIsLoading(true);
    try {
      const orders = await api.getHeldOrders();
      setHeldOrders(orders);
      onHeldCountChange(orders.length);
    } catch (err: any) {
      console.error('Failed to load held orders', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadOrders();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const handleResume = (order: Order) => {
    loadHeldOrder(order);
    onClose();
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
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg overflow-hidden shadow-2xl">
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-amber-500/20 text-amber-400 flex items-center justify-center">
              <PauseCircle className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-100 text-sm">Parked / Held Orders</h3>
              <p className="text-xs text-slate-400">Restore parked carts back into register</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-6 max-h-96 overflow-y-auto space-y-3">
          {isLoading ? (
            <div className="h-32 flex items-center justify-center text-slate-500 text-xs">
              Loading parked orders...
            </div>
          ) : heldOrders.length === 0 ? (
            <div className="h-32 flex flex-col items-center justify-center text-slate-500 text-xs text-center border-2 border-dashed border-slate-800 rounded-xl p-4">
              <PauseCircle className="w-6 h-6 text-slate-600 mb-1" />
              <p className="font-medium text-slate-400">No parked orders</p>
              <p className="text-[11px] text-slate-600">Active cart can be parked anytime</p>
            </div>
          ) : (
            heldOrders.map((order) => (
              <div
                key={order.id}
                className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between gap-3"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono font-bold text-xs text-slate-100">
                      {order.order_number}
                    </span>
                    <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-800 text-slate-300 font-mono">
                      {order.items?.length || 0} items
                    </span>
                  </div>
                  {order.notes && (
                    <p className="text-xs text-amber-300/90 mt-1 italic">"{order.notes}"</p>
                  )}
                  <div className="text-[11px] text-slate-500 flex items-center gap-1 mt-1 font-mono">
                    <Clock className="w-3 h-3" />
                    {new Date(order.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    <span className="ml-2 font-bold text-slate-300">{formatIDR(order.total_amount)}</span>
                  </div>
                </div>

                <button
                  onClick={() => handleResume(order)}
                  className="px-3 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600 text-indigo-300 hover:text-white border border-indigo-500/30 text-xs font-medium flex items-center gap-1.5 transition-all shadow-sm"
                >
                  <PlayCircle className="w-4 h-4" />
                  <span>Resume</span>
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
