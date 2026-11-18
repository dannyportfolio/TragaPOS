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
    <div className="flex-1 flex flex-col h-full bg-slate-50 dark:bg-slate-950 p-3 sm:p-4 lg:p-6 overflow-y-auto md:overflow-hidden select-none transition-colors">
      {/* Top Banner */}
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4 shrink-0">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400 border border-amber-200 dark:border-amber-500/20 flex items-center justify-center shrink-0">
            <ChefHat className="w-5 h-5" />
          </div>
          <div>
            <h2 className="font-bold text-slate-900 dark:text-slate-100 text-sm sm:text-base flex items-center gap-2">
              Kitchen Display System (KDS)
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" aria-hidden="true"></span>
            </h2>
            <p className="text-xs text-slate-600 dark:text-slate-400">Real-time order fulfillment &amp; prep queue</p>
          </div>
        </div>

        <button
          type="button"
          onClick={loadKdsOrders}
          disabled={isLoading}
          className="min-h-[44px] px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-800 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 text-xs font-semibold flex items-center gap-1.5 transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
          <span>Refresh</span>
        </button>
      </div>

      {/* 3 Columns Kanban Queue */}
      <div className="flex-1 grid grid-cols-1 md:grid-cols-3 gap-4 md:min-h-0 md:overflow-hidden">
        {/* Column 1: Paid / In Queue */}
        <div className="flex flex-col bg-white dark:bg-slate-900/60 rounded-2xl border border-slate-200 dark:border-slate-800/80 min-h-[260px] md:min-h-0 overflow-hidden shadow-sm">
          <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-amber-800 dark:text-amber-400">
              <Clock className="w-4 h-4" />
              <span>WAITING QUEUE ({paidOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {paidOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 dark:text-slate-400 text-xs">
                No tickets waiting
              </div>
            ) : (
              paidOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-amber-300 dark:border-amber-500/30 space-y-2.5 shadow-sm"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-900 dark:text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono font-semibold text-amber-800 dark:text-amber-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  {order.notes && (
                    <div className="p-2 rounded-lg bg-amber-50 dark:bg-amber-500/10 text-amber-900 dark:text-amber-300 text-xs italic font-medium border border-amber-200 dark:border-amber-500/20">
                      "{order.notes}"
                    </div>
                  )}

                  <div className="space-y-1 text-xs border-t border-slate-200 dark:border-slate-800 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-800 dark:text-slate-200">
                        <span>
                          <strong className="text-amber-700 dark:text-amber-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    type="button"
                    onClick={() => handleUpdateStatus(order.id, 'PREPARING')}
                    className="w-full min-h-[44px] py-2 rounded-xl bg-amber-100 dark:bg-amber-500/20 hover:bg-amber-500 text-amber-900 dark:text-amber-300 hover:text-white dark:hover:text-slate-950 font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors"
                  >
                    <Flame className="w-4 h-4" />
                    <span>Start Cooking</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 2: Preparing / In Kitchen */}
        <div className="flex flex-col bg-white dark:bg-slate-900/60 rounded-2xl border border-slate-200 dark:border-slate-800/80 min-h-[260px] md:min-h-0 overflow-hidden shadow-sm">
          <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-sky-800 dark:text-sky-400">
              <Flame className="w-4 h-4" />
              <span>PREPARING / COOKING ({preparingOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {preparingOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 dark:text-slate-400 text-xs">
                No orders currently in prep
              </div>
            ) : (
              preparingOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-sky-300 dark:border-sky-500/30 space-y-2.5 shadow-sm"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-900 dark:text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono font-semibold text-sky-800 dark:text-sky-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  {order.notes && (
                    <div className="p-2 rounded-lg bg-sky-50 dark:bg-sky-500/10 text-sky-900 dark:text-sky-300 text-xs italic font-medium border border-sky-200 dark:border-sky-500/20">
                      "{order.notes}"
                    </div>
                  )}

                  <div className="space-y-1 text-xs border-t border-slate-200 dark:border-slate-800 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-800 dark:text-slate-200">
                        <span>
                          <strong className="text-sky-700 dark:text-sky-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    type="button"
                    onClick={() => handleUpdateStatus(order.id, 'READY')}
                    className="w-full min-h-[44px] py-2 rounded-xl bg-sky-100 dark:bg-sky-500/20 hover:bg-sky-600 dark:hover:bg-sky-500 text-sky-900 dark:text-sky-300 hover:text-white dark:hover:text-slate-950 font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors"
                  >
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Mark Ready for Pickup</span>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Column 3: Ready / Completed */}
        <div className="flex flex-col bg-white dark:bg-slate-900/60 rounded-2xl border border-slate-200 dark:border-slate-800/80 min-h-[260px] md:min-h-0 overflow-hidden shadow-sm">
          <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-emerald-800 dark:text-emerald-400">
              <CheckCircle2 className="w-4 h-4" />
              <span>READY FOR PICKUP ({readyOrders.length})</span>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-3 space-y-3">
            {readyOrders.length === 0 ? (
              <div className="h-40 flex items-center justify-center text-slate-600 dark:text-slate-400 text-xs">
                No orders ready for pickup
              </div>
            ) : (
              readyOrders.map((order) => (
                <div
                  key={order.id}
                  className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-emerald-300 dark:border-emerald-500/30 space-y-2.5 shadow-sm"
                >
                  <div className="flex justify-between items-center text-xs">
                    <span className="font-mono font-bold text-slate-900 dark:text-slate-100">{order.order_number}</span>
                    <span className="text-[11px] font-mono font-semibold text-emerald-800 dark:text-emerald-400">{getElapsedTime(order.created_at)}</span>
                  </div>

                  <div className="space-y-1 text-xs border-t border-slate-200 dark:border-slate-800 pt-2">
                    {order.items.map((it, idx) => (
                      <div key={idx} className="flex justify-between text-slate-800 dark:text-slate-200">
                        <span>
                          <strong className="text-emerald-700 dark:text-emerald-400 font-mono">{it.quantity}x</strong> {it.name}
                        </span>
                      </div>
                    ))}
                  </div>

                  <button
                    type="button"
                    onClick={() => handleUpdateStatus(order.id, 'COMPLETED')}
                    className="w-full min-h-[44px] py-2 rounded-xl bg-emerald-600 hover:bg-emerald-700 dark:hover:bg-emerald-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 transition-colors shadow-md shadow-emerald-600/20"
                  >
                    <Check className="w-4 h-4" />
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
