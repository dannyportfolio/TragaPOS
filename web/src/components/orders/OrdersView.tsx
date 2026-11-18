import React, { useState, useEffect } from 'react';
import {
  ClipboardList,
  Search,
  Filter,
  RefreshCw,
  Printer,
  Ban,
  CheckCircle2,
  AlertTriangle,
  X,
  FileText,
  Clock,
  Eye,
  Download,
} from 'lucide-react';
import { Order, OrderStatus } from '../../domain/types';
import { api } from '../../services/api';

const STATUS_FILTERS: { key: OrderStatus | 'ALL'; label: string; color: string }[] = [
  { key: 'ALL', label: 'All Orders', color: 'bg-slate-100 text-slate-700 border-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700' },
  { key: 'PAID', label: 'Paid', color: 'bg-emerald-50 text-emerald-800 border-emerald-200 dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30' },
  { key: 'HELD', label: 'Parked / Held', color: 'bg-amber-50 text-amber-800 border-amber-200 dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30' },
  { key: 'PREPARING', label: 'Preparing', color: 'bg-blue-50 text-blue-800 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30' },
  { key: 'READY', label: 'Ready', color: 'bg-indigo-50 text-indigo-800 border-indigo-200 dark:bg-indigo-500/20 dark:text-indigo-300 dark:border-indigo-500/30' },
  { key: 'COMPLETED', label: 'Completed', color: 'bg-slate-100 text-slate-800 border-slate-300 dark:bg-slate-500/20 dark:text-slate-300 dark:border-slate-500/30' },
  { key: 'CANCELLED', label: 'Cancelled / Void', color: 'bg-rose-50 text-rose-800 border-rose-200 dark:bg-rose-500/20 dark:text-rose-300 dark:border-rose-500/30' },
];

export const OrdersView: React.FC = () => {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [selectedStatus, setSelectedStatus] = useState<OrderStatus | 'ALL'>('ALL');

  // Modals state
  const [receiptOrder, setReceiptOrder] = useState<Order | null>(null);
  const [receiptText, setReceiptText] = useState<string>('');
  const [receiptLoading, setReceiptLoading] = useState(false);

  const [voidingOrder, setVoidingOrder] = useState<Order | null>(null);
  const [voidReason, setVoidReason] = useState('Customer requested return');

  const [viewingOrder, setViewingOrder] = useState<Order | null>(null);

  const loadOrders = async () => {
    try {
      setLoading(true);
      setError(null);
      const params = selectedStatus === 'ALL' ? {} : { status: selectedStatus };
      const data = await api.getOrders({ ...params, limit: 100 });
      setOrders(data);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch orders');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOrders();
  }, [selectedStatus]);

  const showNotification = (msg: string) => {
    setSuccessMsg(msg);
    setTimeout(() => setSuccessMsg(null), 3500);
  };

  const handleOpenReceipt = async (order: Order) => {
    setReceiptOrder(order);
    setReceiptLoading(true);
    try {
      const text = await api.getReceipt(order.id);
      setReceiptText(text);
    } catch (err: any) {
      setReceiptText(`Failed to load receipt: ${err.message}`);
    } finally {
      setReceiptLoading(false);
    }
  };

  const handleVoidOrder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!voidingOrder) return;
    try {
      setError(null);
      await api.voidOrder(voidingOrder.id, voidReason);
      showNotification(`Order #${voidingOrder.order_number} has been voided & inventory restored`);
      setVoidingOrder(null);
      await loadOrders();
    } catch (err: any) {
      setError(err.message || 'Failed to void order');
    }
  };

  const filteredOrders = orders.filter((ord) => {
    const q = search.toLowerCase();
    return (
      !q ||
      ord.order_number.toLowerCase().includes(q) ||
      ord.id.toLowerCase().includes(q) ||
      (ord.notes && ord.notes.toLowerCase().includes(q))
    );
  });

  const exportCsv = () => {
    const header = ['OrderNumber', 'Status', 'Currency', 'TotalAmount', 'ItemsCount', 'CreatedAt', 'Notes'];
    const rows = filteredOrders.map((o) => [
      o.order_number,
      o.status,
      o.currency || 'IDR',
      String(o.total_amount),
      String(o.items ? o.items.length : 0),
      o.created_at,
      `"${(o.notes || '').replace(/"/g, '""')}"`,
    ]);
    const csv = [header.join(','), ...rows.map((r) => r.join(','))].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `orders-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const paidRevenue = filteredOrders
    .filter((o) => o.status !== 'CANCELLED' && o.status !== 'REFUNDED' && o.status !== 'HELD')
    .reduce((sum, o) => sum + Number(o.total_amount || 0), 0);
  const completedCount = filteredOrders.filter(
    (o) => o.status !== 'CANCELLED' && o.status !== 'REFUNDED' && o.status !== 'HELD'
  ).length;
  const avgTicket = completedCount > 0 ? Math.round(paidRevenue / completedCount) : 0;

  return (
    <div className="flex-1 flex flex-col bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 overflow-hidden transition-colors">
      {/* Top Bar */}
      <div className="p-4 sm:p-6 border-b border-slate-200 dark:border-slate-800 flex flex-wrap items-center justify-between gap-4 bg-white dark:bg-slate-900/60">
        <div>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <ClipboardList className="w-5 h-5 text-cyan-700 dark:text-cyan-400" />
            Order History &amp; Sales Audit
          </h1>
          <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
            Browse completed, parked, and fulfillment orders with receipt printing and inventory-safe voiding
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={exportCsv}
            disabled={filteredOrders.length === 0}
            className="min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700 text-xs font-semibold flex items-center gap-1.5 transition disabled:opacity-50"
          >
            <Download className="w-4 h-4 text-cyan-700 dark:text-cyan-400" />
            <span>Export CSV</span>
          </button>
          <button
            type="button"
            onClick={loadOrders}
            disabled={loading}
            className="min-h-[44px] min-w-[44px] p-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 flex items-center justify-center transition"
            title="Refresh orders"
            aria-label="Refresh orders"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Analytics Summary Strip */}
      <div className="px-4 sm:px-6 py-3 grid grid-cols-2 lg:grid-cols-4 gap-3 border-b border-slate-200 dark:border-slate-800/80 bg-white/70 dark:bg-slate-900/20">
        <div className="p-3 rounded-xl bg-white dark:bg-slate-900/70 border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="text-[11px] text-slate-600 dark:text-slate-400">Filtered Orders</div>
          <div className="text-base font-bold text-slate-900 dark:text-slate-100 mt-0.5">{filteredOrders.length}</div>
        </div>
        <div className="p-3 rounded-xl bg-white dark:bg-slate-900/70 border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="text-[11px] text-slate-600 dark:text-slate-400">Active / Paid Count</div>
          <div className="text-base font-bold text-emerald-700 dark:text-emerald-400 mt-0.5">{completedCount}</div>
        </div>
        <div className="p-3 rounded-xl bg-white dark:bg-slate-900/70 border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="text-[11px] text-slate-600 dark:text-slate-400">Gross Sales</div>
          <div className="text-base font-bold font-mono text-cyan-700 dark:text-cyan-400 mt-0.5">
            IDR {paidRevenue.toLocaleString()}
          </div>
        </div>
        <div className="p-3 rounded-xl bg-white dark:bg-slate-900/70 border border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="text-[11px] text-slate-600 dark:text-slate-400">Avg. Order Value</div>
          <div className="text-base font-bold font-mono text-slate-900 dark:text-slate-200 mt-0.5">
            IDR {avgTicket.toLocaleString()}
          </div>
        </div>
      </div>

      {/* Notifications */}
      {successMsg && (
        <div className="mx-4 sm:mx-6 mt-4 p-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/15 border border-emerald-200 dark:border-emerald-500/30 text-emerald-800 dark:text-emerald-300 text-xs sm:text-sm flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
          <span>{successMsg}</span>
        </div>
      )}
      {error && (
        <div className="mx-4 sm:mx-6 mt-4 p-3 rounded-xl bg-rose-50 dark:bg-rose-500/15 border border-rose-200 dark:border-rose-500/30 text-rose-800 dark:text-rose-300 text-xs sm:text-sm flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
          <button type="button" onClick={() => setError(null)} className="ml-auto text-rose-600 dark:text-rose-400 hover:text-rose-800 dark:hover:text-rose-200">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Filters & Search */}
      <div className="px-4 sm:px-6 py-3 sm:py-4 flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 dark:border-slate-800/80 bg-white/70 dark:bg-slate-900/30">
        <div className="relative flex-1 min-w-[220px] max-w-md">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500 dark:text-slate-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by order # or notes..."
            className="w-full min-h-[44px] pl-10 pr-4 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-500 focus:outline-none focus:border-cyan-500 transition"
          />
        </div>

        <div className="flex flex-wrap gap-1.5 items-center">
          <Filter className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400 mr-1" />
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              onClick={() => setSelectedStatus(f.key)}
              className={`min-h-[36px] px-3 py-1.5 rounded-lg text-xs font-semibold border transition-all ${
                selectedStatus === f.key
                  ? 'bg-cyan-700 border-cyan-700 text-white shadow-sm'
                  : 'bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-800 text-slate-700 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:border-slate-400 dark:hover:border-slate-700'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {/* Table Container */}
      <div className="flex-1 overflow-auto p-3 sm:p-4 lg:p-6">
        <div className="border border-slate-200 dark:border-slate-800 rounded-2xl overflow-x-auto bg-white dark:bg-slate-900/40 shadow-sm">
          <table className="w-full min-w-[700px] text-left border-collapse">
            <thead>
              <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/80 text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                <th className="px-4 py-3">Order Number</th>
                <th className="px-4 py-3">Timestamp</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Items Count</th>
                <th className="px-4 py-3">Total Amount</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800/60 text-sm">
              {filteredOrders.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-slate-600 dark:text-slate-400">
                    {loading ? 'Loading orders...' : 'No orders found for the selected filter.'}
                  </td>
                </tr>
              ) : (
                filteredOrders.map((ord) => {
                  const statusObj = STATUS_FILTERS.find((s) => s.key === ord.status);
                  const canVoid = ord.status !== 'CANCELLED' && ord.status !== 'REFUNDED';
                  return (
                    <tr key={ord.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 transition-colors">
                      <td className="px-4 py-3.5">
                        <div className="font-mono font-semibold text-cyan-700 dark:text-cyan-400">#{ord.order_number}</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 truncate max-w-xs">{ord.notes || 'No customer notes'}</div>
                      </td>
                      <td className="px-4 py-3.5 text-xs text-slate-600 dark:text-slate-400">
                        <div className="flex items-center gap-1.5">
                          <Clock className="w-3.5 h-3.5 text-slate-500" />
                          <span>{new Date(ord.created_at).toLocaleTimeString()}</span>
                        </div>
                        <div className="text-[11px] text-slate-600 dark:text-slate-400 mt-0.5">
                          {new Date(ord.created_at).toLocaleDateString()}
                        </div>
                      </td>
                      <td className="px-4 py-3.5">
                        <span
                          className={`inline-block px-2 py-0.5 rounded text-[11px] font-semibold border ${
                            statusObj ? statusObj.color : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                          }`}
                        >
                          {ord.status}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 text-slate-700 dark:text-slate-300 font-medium">
                        {ord.items ? ord.items.length : 0} items
                      </td>
                      <td className="px-4 py-3.5 font-bold text-slate-900 dark:text-slate-100">
                        {ord.currency || 'IDR'} {Number(ord.total_amount).toLocaleString()}
                      </td>
                      <td className="px-4 py-3.5 text-right space-x-2 whitespace-nowrap">
                        <button
                          type="button"
                          onClick={() => setViewingOrder(ord)}
                          className="min-h-[36px] min-w-[36px] p-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 inline-flex items-center justify-center transition"
                          title="View order details"
                          aria-label={`View order ${ord.order_number}`}
                        >
                          <Eye className="w-3.5 h-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleOpenReceipt(ord)}
                          className="min-h-[36px] px-2.5 py-1 rounded-lg bg-cyan-50 dark:bg-slate-800 hover:bg-cyan-100 dark:hover:bg-slate-700 text-cyan-800 dark:text-cyan-300 border border-cyan-200 dark:border-slate-700 text-xs font-semibold inline-flex items-center gap-1.5 transition"
                          title="Print thermal receipt"
                        >
                          <Printer className="w-3.5 h-3.5" />
                          <span>Receipt</span>
                        </button>
                        {canVoid && (
                          <button
                            type="button"
                            onClick={() => {
                              setVoidingOrder(ord);
                              setVoidReason('Customer requested cancellation / return');
                            }}
                            className="min-h-[36px] px-2.5 py-1 rounded-lg bg-rose-50 dark:bg-rose-500/10 hover:bg-rose-100 dark:hover:bg-rose-500/20 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-500/30 text-xs font-semibold inline-flex items-center gap-1.5 transition"
                            title="Void order & restore inventory"
                          >
                            <Ban className="w-3.5 h-3.5" />
                            <span>Void</span>
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Order Details Modal */}
      {viewingOrder && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-lg max-h-[90dvh] overflow-y-auto p-5 sm:p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
              <div>
                <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100 font-mono">Order #{viewingOrder.order_number}</h2>
                <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">ID: {viewingOrder.id}</div>
              </div>
              <button
                type="button"
                onClick={() => setViewingOrder(null)}
                aria-label="Close modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 space-y-4 max-h-[60vh] overflow-y-auto pr-1">
              <div className="border border-slate-200 dark:border-slate-800 rounded-xl p-3 bg-slate-50 dark:bg-slate-950/60">
                <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider mb-2">Order Items</div>
                <div className="space-y-2">
                  {viewingOrder.items?.map((it, idx) => (
                    <div key={idx} className="flex justify-between items-center text-sm">
                      <div>
                        <div className="font-semibold text-slate-900 dark:text-slate-200">{it.name}</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 font-mono">
                          {it.quantity} {it.uom || 'x'} @ {viewingOrder.currency} {Number(it.unit_price).toLocaleString()}
                        </div>
                      </div>
                      <div className="font-semibold text-slate-900 dark:text-slate-200">
                        {viewingOrder.currency} {Number(it.total_amount).toLocaleString()}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="border border-slate-200 dark:border-slate-800 rounded-xl p-3 bg-slate-50 dark:bg-slate-950/60">
                <div className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider mb-2">Financials</div>
                <div className="space-y-1 text-xs">
                  <div className="flex justify-between text-slate-600 dark:text-slate-400">
                    <span>Subtotal</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.subtotal).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-slate-600 dark:text-slate-400">
                    <span>Tax</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.tax_amount).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-slate-600 dark:text-slate-400">
                    <span>Discount</span>
                    <span>- {viewingOrder.currency} {Number(viewingOrder.discount_amount || 0).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm font-bold text-slate-900 dark:text-slate-100 pt-2 border-t border-slate-200 dark:border-slate-800">
                    <span>Total Amount</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.total_amount).toLocaleString()}</span>
                  </div>
                </div>
              </div>

              {viewingOrder.notes && (
                <div className="p-3 rounded-lg bg-slate-100 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800 text-xs text-slate-700 dark:text-slate-300">
                  <span className="font-semibold text-slate-600 dark:text-slate-400 block mb-0.5">Audit / Notes:</span>
                  {viewingOrder.notes}
                </div>
              )}
            </div>

            <div className="mt-6 flex justify-end">
              <button
                type="button"
                onClick={() => setViewingOrder(null)}
                className="min-h-[44px] px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-sm font-semibold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Receipt Thermal Preview Modal */}
      {receiptOrder && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md max-h-[90dvh] overflow-y-auto p-5 sm:p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
              <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <Printer className="w-5 h-5 text-cyan-700 dark:text-cyan-400" />
                Thermal Receipt
              </h2>
              <button
                type="button"
                onClick={() => setReceiptOrder(null)}
                aria-label="Close modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 p-4 rounded-xl bg-slate-50 dark:bg-white border border-slate-200 text-slate-900 font-mono text-xs whitespace-pre shadow-inner max-h-[60vh] overflow-y-auto">
              {receiptLoading ? 'Generating receipt...' : receiptText}
            </div>

            <div className="mt-4 flex justify-between items-center gap-2">
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard.writeText(receiptText);
                  showNotification('Receipt copied to clipboard');
                }}
                className="min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-xs font-semibold transition inline-flex items-center gap-1.5"
              >
                <FileText className="w-4 h-4" />
                <span>Copy Text</span>
              </button>

              <button
                type="button"
                onClick={() => setReceiptOrder(null)}
                className="min-h-[44px] px-4 py-2 rounded-xl bg-cyan-700 hover:bg-cyan-800 dark:bg-cyan-600 dark:hover:bg-cyan-500 text-white text-sm font-semibold transition"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Void Confirmation Modal */}
      {voidingOrder && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md max-h-[90dvh] overflow-y-auto p-5 sm:p-6 shadow-2xl">
            <div className="flex items-center gap-3 text-rose-700 dark:text-rose-400 mb-3">
              <div className="w-10 h-10 rounded-full bg-rose-50 dark:bg-rose-500/15 border border-rose-200 dark:border-rose-500/30 flex items-center justify-center shrink-0">
                <Ban className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-bold text-slate-900 dark:text-slate-100 text-lg">Void Order #{voidingOrder.order_number}</h3>
                <div className="text-xs text-rose-700 dark:text-rose-400">Inventory reversal will be automatically applied</div>
              </div>
            </div>

            <p className="text-sm text-slate-700 dark:text-slate-300 mb-4">
              Voiding this order will mark it as <span className="text-rose-700 dark:text-rose-400 font-bold">CANCELLED</span> and
              restore all deducted stock balances via compensating inventory movements.
            </p>

            <form onSubmit={handleVoidOrder} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Reason for Void *</label>
                <textarea
                  required
                  rows={3}
                  value={voidReason}
                  onChange={(e) => setVoidReason(e.target.value)}
                  placeholder="e.g. Customer returned goods, incorrect billing amount..."
                  className="w-full px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-rose-500 focus:outline-none"
                />
              </div>

              <div className="flex flex-wrap justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setVoidingOrder(null)}
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-sm font-semibold transition"
                >
                  Keep Order
                </button>
                <button
                  type="submit"
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 dark:hover:bg-rose-500 text-white text-sm font-semibold shadow-lg shadow-rose-600/20 transition"
                >
                  Confirm Void &amp; Restore Stock
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
