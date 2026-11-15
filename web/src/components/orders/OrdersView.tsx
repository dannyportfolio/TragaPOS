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
  { key: 'ALL', label: 'All Orders', color: 'bg-slate-800 text-slate-300' },
  { key: 'PAID', label: 'Paid', color: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' },
  { key: 'HELD', label: 'Parked / Held', color: 'bg-amber-500/20 text-amber-300 border-amber-500/30' },
  { key: 'PREPARING', label: 'Preparing', color: 'bg-blue-500/20 text-blue-300 border-blue-500/30' },
  { key: 'READY', label: 'Ready', color: 'bg-indigo-500/20 text-indigo-300 border-indigo-500/30' },
  { key: 'COMPLETED', label: 'Completed', color: 'bg-slate-500/20 text-slate-300 border-slate-500/30' },
  { key: 'CANCELLED', label: 'Cancelled / Void', color: 'bg-rose-500/20 text-rose-300 border-rose-500/30' },
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
    <div className="flex-1 flex flex-col bg-slate-950 text-slate-100 overflow-hidden">
      {/* Top Bar */}
      <div className="p-4 sm:p-6 border-b border-slate-800 flex flex-wrap items-center justify-between gap-4 bg-slate-900/60">
        <div>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2 text-slate-100">
            <ClipboardList className="w-5 h-5 text-cyan-400" />
            Order History & Sales Audit
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Browse completed, parked, and fulfillment orders with receipt printing and inventory-safe voiding
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={exportCsv}
            disabled={filteredOrders.length === 0}
            className="min-h-[40px] px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 text-xs font-semibold flex items-center gap-1.5 transition disabled:opacity-50"
          >
            <Download className="w-4 h-4 text-cyan-400" />
            <span>Export CSV</span>
          </button>
          <button
            onClick={loadOrders}
            disabled={loading}
            className="min-h-[40px] p-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition"
            title="Refresh orders"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Analytics Summary Strip */}
      <div className="px-4 sm:px-6 py-3 grid grid-cols-2 lg:grid-cols-4 gap-3 border-b border-slate-800/80 bg-slate-900/20">
        <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800">
          <div className="text-[11px] text-slate-400">Filtered Orders</div>
          <div className="text-base font-bold text-slate-100 mt-0.5">{filteredOrders.length}</div>
        </div>
        <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800">
          <div className="text-[11px] text-slate-400">Active / Paid Count</div>
          <div className="text-base font-bold text-emerald-400 mt-0.5">{completedCount}</div>
        </div>
        <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800">
          <div className="text-[11px] text-slate-400">Gross Sales</div>
          <div className="text-base font-bold font-mono text-cyan-400 mt-0.5">
            IDR {paidRevenue.toLocaleString()}
          </div>
        </div>
        <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800">
          <div className="text-[11px] text-slate-400">Avg. Order Value</div>
          <div className="text-base font-bold font-mono text-slate-200 mt-0.5">
            IDR {avgTicket.toLocaleString()}
          </div>
        </div>
      </div>

      {/* Notifications */}
      {successMsg && (
        <div className="mx-6 mt-4 p-3 rounded-lg bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-sm flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
          <span>{successMsg}</span>
        </div>
      )}
      {error && (
        <div className="mx-6 mt-4 p-3 rounded-lg bg-rose-500/15 border border-rose-500/30 text-rose-300 text-sm flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
          <button onClick={() => setError(null)} className="ml-auto text-rose-400 hover:text-rose-200">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Filters & Search */}
      <div className="px-6 py-4 flex flex-wrap items-center justify-between gap-4 border-b border-slate-800/80 bg-slate-900/30">
        <div className="relative flex-1 min-w-[260px] max-w-md">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by order # or notes..."
            className="w-full pl-9 pr-4 py-2 rounded-lg bg-slate-900 border border-slate-800 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-cyan-500 transition"
          />
        </div>

        <div className="flex flex-wrap gap-1.5 items-center">
          <Filter className="w-3.5 h-3.5 text-slate-400 mr-1" />
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setSelectedStatus(f.key)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-all ${
                selectedStatus === f.key
                  ? 'bg-cyan-600 border-cyan-500 text-white shadow-sm'
                  : 'bg-slate-900 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {/* Table Container */}
      <div className="flex-1 overflow-auto p-6">
        <div className="border border-slate-800 rounded-xl overflow-hidden bg-slate-900/40">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-900/80 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <th className="px-4 py-3">Order Number</th>
                <th className="px-4 py-3">Timestamp</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Items Count</th>
                <th className="px-4 py-3">Total Amount</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 text-sm">
              {filteredOrders.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-slate-500">
                    {loading ? 'Loading orders...' : 'No orders found for the selected filter.'}
                  </td>
                </tr>
              ) : (
                filteredOrders.map((ord) => {
                  const statusObj = STATUS_FILTERS.find((s) => s.key === ord.status);
                  const canVoid = ord.status !== 'CANCELLED' && ord.status !== 'REFUNDED';
                  return (
                    <tr key={ord.id} className="hover:bg-slate-800/40 transition-colors">
                      <td className="px-4 py-3.5">
                        <div className="font-mono font-semibold text-cyan-400">#{ord.order_number}</div>
                        <div className="text-xs text-slate-400 truncate max-w-xs">{ord.notes || 'No customer notes'}</div>
                      </td>
                      <td className="px-4 py-3.5 text-xs text-slate-400">
                        <div className="flex items-center gap-1.5">
                          <Clock className="w-3.5 h-3.5 text-slate-500" />
                          <span>{new Date(ord.created_at).toLocaleTimeString()}</span>
                        </div>
                        <div className="text-[11px] text-slate-400 mt-0.5">
                          {new Date(ord.created_at).toLocaleDateString()}
                        </div>
                      </td>
                      <td className="px-4 py-3.5">
                        <span
                          className={`inline-block px-2 py-0.5 rounded text-[11px] font-semibold border ${
                            statusObj ? statusObj.color : 'bg-slate-800 text-slate-300'
                          }`}
                        >
                          {ord.status}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 text-slate-300 font-medium">
                        {ord.items ? ord.items.length : 0} items
                      </td>
                      <td className="px-4 py-3.5 font-bold text-slate-100">
                        {ord.currency || 'IDR'} {Number(ord.total_amount).toLocaleString()}
                      </td>
                      <td className="px-4 py-3.5 text-right space-x-2">
                        <button
                          onClick={() => setViewingOrder(ord)}
                          className="p-1.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 inline-flex transition"
                          title="View order details"
                        >
                          <Eye className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => handleOpenReceipt(ord)}
                          className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-cyan-300 border border-slate-700 text-xs font-medium inline-flex items-center gap-1.5 transition"
                          title="Print thermal receipt"
                        >
                          <Printer className="w-3.5 h-3.5" />
                          <span>Receipt</span>
                        </button>
                        {canVoid && (
                          <button
                            onClick={() => {
                              setVoidingOrder(ord);
                              setVoidReason('Customer requested cancellation / return');
                            }}
                            className="px-2.5 py-1 rounded bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/30 text-xs font-medium inline-flex items-center gap-1.5 transition"
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
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-800">
              <div>
                <h2 className="text-lg font-bold text-slate-100 font-mono">Order #{viewingOrder.order_number}</h2>
                <div className="text-xs text-slate-400 mt-0.5">ID: {viewingOrder.id}</div>
              </div>
              <button onClick={() => setViewingOrder(null)} className="text-slate-400 hover:text-slate-200">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 space-y-4 max-h-[60vh] overflow-y-auto pr-1">
              <div className="border border-slate-800 rounded-xl p-3 bg-slate-950/60">
                <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Order Items</div>
                <div className="space-y-2">
                  {viewingOrder.items?.map((it, idx) => (
                    <div key={idx} className="flex justify-between items-center text-sm">
                      <div>
                        <div className="font-medium text-slate-200">{it.name}</div>
                        <div className="text-xs text-slate-400 font-mono">
                          {it.quantity} {it.uom || 'x'} @ {viewingOrder.currency} {Number(it.unit_price).toLocaleString()}
                        </div>
                      </div>
                      <div className="font-semibold text-slate-200">
                        {viewingOrder.currency} {Number(it.total_amount).toLocaleString()}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="border border-slate-800 rounded-xl p-3 bg-slate-950/60">
                <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Financials</div>
                <div className="space-y-1 text-xs">
                  <div className="flex justify-between text-slate-400">
                    <span>Subtotal</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.subtotal).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-slate-400">
                    <span>Tax</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.tax_amount).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-slate-400">
                    <span>Discount</span>
                    <span>- {viewingOrder.currency} {Number(viewingOrder.discount_amount || 0).toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between text-sm font-bold text-slate-100 pt-2 border-t border-slate-800">
                    <span>Total Amount</span>
                    <span>{viewingOrder.currency} {Number(viewingOrder.total_amount).toLocaleString()}</span>
                  </div>
                </div>
              </div>

              {viewingOrder.notes && (
                <div className="p-3 rounded-lg bg-slate-800/40 border border-slate-800 text-xs text-slate-300">
                  <span className="font-semibold text-slate-400 block mb-0.5">Audit / Notes:</span>
                  {viewingOrder.notes}
                </div>
              )}
            </div>

            <div className="mt-6 flex justify-end">
              <button
                onClick={() => setViewingOrder(null)}
                className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Receipt Thermal Preview Modal */}
      {receiptOrder && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-800">
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                <Printer className="w-5 h-5 text-cyan-400" />
                Thermal Receipt
              </h2>
              <button onClick={() => setReceiptOrder(null)} className="text-slate-400 hover:text-slate-200">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 p-4 rounded-xl bg-white text-slate-900 font-mono text-xs whitespace-pre shadow-inner max-h-[60vh] overflow-y-auto">
              {receiptLoading ? 'Generating receipt...' : receiptText}
            </div>

            <div className="mt-4 flex justify-between items-center">
              <button
                onClick={() => {
                  navigator.clipboard.writeText(receiptText);
                  showNotification('Receipt copied to clipboard');
                }}
                className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium transition inline-flex items-center gap-1.5"
              >
                <FileText className="w-3.5 h-3.5" />
                <span>Copy Text</span>
              </button>

              <button
                onClick={() => setReceiptOrder(null)}
                className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-sm font-semibold transition"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Void Confirmation Modal */}
      {voidingOrder && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md p-6 shadow-2xl">
            <div className="flex items-center gap-3 text-rose-400 mb-3">
              <div className="w-10 h-10 rounded-full bg-rose-500/15 border border-rose-500/30 flex items-center justify-center">
                <Ban className="w-5 h-5" />
              </div>
              <div>
                <h3 className="font-bold text-slate-100 text-lg">Void Order #{voidingOrder.order_number}</h3>
                <div className="text-xs text-rose-400">Inventory reversal will be automatically applied</div>
              </div>
            </div>

            <p className="text-sm text-slate-300 mb-4">
              Voiding this order will mark it as <span className="text-rose-400 font-bold">CANCELLED</span> and
              restore all deducted stock balances via compensating inventory movements.
            </p>

            <form onSubmit={handleVoidOrder} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">Reason for Void *</label>
                <textarea
                  required
                  rows={3}
                  value={voidReason}
                  onChange={(e) => setVoidReason(e.target.value)}
                  placeholder="e.g. Customer returned goods, incorrect billing amount..."
                  className="w-full px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-sm text-slate-100 focus:border-rose-500 focus:outline-none"
                />
              </div>

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setVoidingOrder(null)}
                  className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium transition"
                >
                  Keep Order
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-sm font-semibold shadow-lg shadow-rose-600/20 transition"
                >
                  Confirm Void & Restore Stock
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
