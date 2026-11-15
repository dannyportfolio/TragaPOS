import React, { useEffect, useState } from 'react';
import { ChefHat, Clock, Flame, CheckCircle2, Check, RefreshCw } from 'lucide-react';
import { Order } from '../../domain/types';
import { api, getAuthToken } from '../../services/api';
import { toast } from '../../store/useToastStore';

export const KdsView: React.FC = () => {
  const [orders, setOrders] = useState<Order[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const loadKdsOrders = async () => {
    setIsLoading(true);
    try {
      const activeOrders = await api.getKdsOrders();
      setOrders(activeOrders);
    } catch (err: any) {
      console.error('Failed to load KDS orders', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadKdsOrders();

    // Auto-poll every 5 seconds as fallback, plus WebSocket listener if connected
    const interval = setInterval(loadKdsOrders, 5000);

    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = getAuthToken();
    const wsQuery = token ? `?token=${encodeURIComponent(token)}` : '';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws${wsQuery}`;
    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(wsUrl);
      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.topic === 'kds' || msg.topic === 'orders') {
            loadKdsOrders();
          }
        } catch {
          // ignore
        }
      };
    } catch {
      // ignore
    }

    return () => {
      clearInterval(interval);
      if (socket) socket.close();
    };
  }, []);

  const handleUpdateStatus = async (orderId: string, nextStatus: string) => {
    try {
      await api.updateOrderStatus(orderId, nextStatus);
      await loadKdsOrders();
    } catch (err: any) {
      toast.error(`Failed to update status: ${err.message}`);
    }
  };

  const getElapsedTime = (createdAt: string) => {
    const mins = Math.floor((Date.now() - new Date(createdAt).getTime()) / 60000);
    return `${mins}m ago`;
  };

  const paidOrders = orders.filter((o) => o.status === 'PAID');
  const preparingOrders = orders.filter((o) => o.status === 'PREPARING');
  const readyOrders = orders.filter((o) => o.status === 'READY');

  return (
    <div className="flex-1 flex flex-col h-full bg-slate-950 p-6 overflow-hidden select-none">
      {/* Top Banner */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2.5">
          <div className="w-9 h-9 rounded-xl bg-amber-500/10 text-amber-400 border border-amber-500/20 flex items-center justify-center">
            <ChefHat className="w-5 h-5" />
          </div>
          <div>
            <h2 className="font-bold text-slate-100 text-sm flex items-center gap-2">
              Kitchen Display System (KDS)
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            </h2>
            <p className="text-xs text-slate-400">Real-time order fulfillment & prep queue</p>
          </div>
        </div>

        <button
          onClick={loadKdsOrders}
          disabled={isLoading}
          className="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-slate-100 text-xs font-medium flex items-center gap-1.5 transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
          <span>Refresh</span>
        </button>
      </div>

      {/* 3 Columns Kanban Queue */}
      <div className="flex-1 grid grid-cols-1 md:grid-cols-3 gap-4 overflow-hidden">
        {/* Column 1: Paid / In Queue */}
        <div className="flex flex-col bg-slate-900/60 rounded-2xl border border-slate-800/80 overflow-hidden">
          <div className="px-4 py-3 bg-slate-900 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-amber-400">
              <Clock className="w-4 h-4" />
              <span>WAITING QUEUE ({paidOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {paidOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 text-xs">
                No tickets waiting
              </div>
            ) : (
              paidOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-950 border border-amber-500/30 space-y-2.5 shadow-md shadow-slate-950/60"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono text-amber-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  {order.notes && (
                    <div className="p-1.5 rounded bg-amber-500/10 text-amber-300 text-[11px] italic font-medium">
                      "{order.notes}"
                    </div>
                  )}

                  <div className="space-y-1 text-xs border-t border-slate-900 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-200">
                        <span>
                          <strong className="text-amber-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    onClick={() => handleUpdateStatus(order.id, 'PREPARING')}
                    className="w-full py-2 rounded-lg bg-amber-500/20 hover:bg-amber-500 text-amber-300 hover:text-slate-950 font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors"
                  >
                    <Flame className="w-3.5 h-3.5" />
                    <span>Start Cooking</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 2: Preparing / In Kitchen */}
        <div className="flex flex-col bg-slate-900/60 rounded-2xl border border-slate-800/80 overflow-hidden">
          <div className="px-4 py-3 bg-slate-900 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-sky-400">
              <Flame className="w-4 h-4" />
              <span>PREPARING / COOKING ({preparingOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {preparingOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 text-xs">
                No orders currently in prep
              </div>
            ) : (
              preparingOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-950 border border-sky-500/30 space-y-2.5 shadow-md shadow-slate-950/60"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono text-sky-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  {order.notes && (
                    <div className="p-1.5 rounded bg-sky-500/10 text-sky-300 text-[11px] italic font-medium">
                      "{order.notes}"
                    </div>
                  )}

                  <div className="space-y-1 text-xs border-t border-slate-900 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-200">
                        <span>
                          <strong className="text-sky-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    onClick={() => handleUpdateStatus(order.id, 'READY')}
                    className="w-full py-2 rounded-lg bg-sky-500/20 hover:bg-sky-500 text-sky-300 hover:text-slate-950 font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors"
                  >
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    <span>Mark Ready for Pickup</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 3: Ready / Completed */}
        <div className="flex flex-col bg-slate-900/60 rounded-2xl border border-slate-800/80 overflow-hidden">
          <div className="px-4 py-3 bg-slate-900 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-emerald-400">
              <CheckCircle2 className="w-4 h-4" />
              <span>READY FOR PICKUP ({readyOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {readyOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 text-xs">
                No orders ready for pickup
              </div>
            ) : (
              readyOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-950 border border-emerald-500/30 space-y-2.5 shadow-md shadow-slate-950/60"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono text-emerald-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  <div className="space-y-1 text-xs border-t border-slate-900 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-200">
                        <span>
                          <strong className="text-emerald-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    onClick={() => handleUpdateStatus(order.id, 'COMPLETED')}
                    className="w-full py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors shadow-lg shadow-emerald-600/20"
                  >
                    <Check className="w-3.5 h-3.5" />
                    <span>Bump / Completed</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
