import React, { useState, useEffect } from 'react';
import {
  Package,
  Plus,
  Search,
  Filter,
  Edit2,
  Trash2,
  TrendingUp,
  RefreshCw,
  AlertTriangle,
  X,
  CheckCircle2,
} from 'lucide-react';
import { Item, ItemSector, CreateItemPayload, UpdateItemPayload } from '../../domain/types';
import { api } from '../../services/api';

const SECTORS: { key: ItemSector | 'ALL'; label: string; color: string }[] = [
  { key: 'ALL', label: 'All Sectors', color: 'bg-slate-100 text-slate-700 border-slate-300 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700' },
  { key: 'FB', label: 'F&B (Kitchen / Bar)', color: 'bg-amber-50 text-amber-800 border-amber-200 dark:bg-amber-500/20 dark:text-amber-300 dark:border-amber-500/30' },
  { key: 'RETAIL', label: 'Retail (Multi-UOM)', color: 'bg-blue-50 text-blue-800 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/30' },
  { key: 'TOURISM', label: 'Tourism (Tickets)', color: 'bg-purple-50 text-purple-800 border-purple-200 dark:bg-purple-500/20 dark:text-purple-300 dark:border-purple-500/30' },
  { key: 'HOSPITALITY', label: 'Hospitality (Folio)', color: 'bg-emerald-50 text-emerald-800 border-emerald-200 dark:bg-emerald-500/20 dark:text-emerald-300 dark:border-emerald-500/30' },
];

export const ItemsView: React.FC = () => {
  const [items, setItems] = useState<Item[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [selectedSector, setSelectedSector] = useState<ItemSector | 'ALL'>('ALL');

  // Modals state
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<Item | null>(null);
  const [adjustingItem, setAdjustingItem] = useState<Item | null>(null);
  const [deletingItem, setDeletingItem] = useState<Item | null>(null);

  // Form states
  const [itemForm, setItemForm] = useState<CreateItemPayload>({
    sku: '',
    barcode: '',
    name: '',
    sector: 'RETAIL',
    base_price: 10000,
    currency: 'IDR',
    tax_rate: 0.11,
    sector_traits: {},
    is_active: true,
  });

  const [adjustForm, setAdjustForm] = useState({
    delta_qty: 10,
    reason: 'RESTOCK',
    ref_id: '',
  });

  const loadItems = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getCatalogItems(undefined, 200);
      setItems(data);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch items');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadItems();
  }, []);

  const showNotification = (msg: string) => {
    setSuccessMsg(msg);
    setTimeout(() => setSuccessMsg(null), 3500);
  };

  const handleOpenAdd = () => {
    setItemForm({
      sku: '',
      barcode: '',
      name: '',
      sector: 'RETAIL',
      base_price: 15000,
      currency: 'IDR',
      tax_rate: 0.11,
      sector_traits: {},
      is_active: true,
    });
    setIsAddOpen(true);
  };

  const handleOpenEdit = (item: Item) => {
    setEditingItem(item);
    setItemForm({
      sku: item.sku,
      barcode: item.barcode || '',
      name: item.name,
      sector: item.sector,
      base_price: item.base_price,
      currency: item.currency || 'IDR',
      tax_rate: item.tax_rate ?? 0.11,
      sector_traits: item.sector_traits || {},
      is_active: item.is_active,
    });
  };

  const handleSaveItem = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setError(null);
      if (!itemForm.sku.trim() || !itemForm.name.trim()) {
        setError('SKU and Product Name are required');
        return;
      }

      if (editingItem) {
        const updatePayload: UpdateItemPayload = {
          barcode: itemForm.barcode,
          name: itemForm.name,
          base_price: Number(itemForm.base_price),
          tax_rate: Number(itemForm.tax_rate),
          sector_traits: itemForm.sector_traits,
          is_active: itemForm.is_active,
        };
        await api.updateItem(editingItem.id, updatePayload);
        showNotification(`Item "${itemForm.name}" updated successfully`);
        setEditingItem(null);
      } else {
        await api.createItem({
          ...itemForm,
          base_price: Number(itemForm.base_price),
          tax_rate: Number(itemForm.tax_rate),
        });
        showNotification(`Product "${itemForm.name}" created successfully`);
        setIsAddOpen(false);
      }
      await loadItems();
    } catch (err: any) {
      setError(err.message || 'Operation failed');
    }
  };

  const handleDelete = async () => {
    if (!deletingItem) return;
    try {
      setError(null);
      await api.deleteItem(deletingItem.id);
      showNotification(`Item "${deletingItem.name}" deleted successfully`);
      setDeletingItem(null);
      await loadItems();
    } catch (err: any) {
      setError(err.message || 'Failed to delete item');
    }
  };

  const handleStockAdjust = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!adjustingItem) return;
    try {
      setError(null);
      const res = await api.adjustStock({
        sku: adjustingItem.sku,
        delta_qty: Number(adjustForm.delta_qty),
        reason: adjustForm.reason,
        ref_id: adjustForm.ref_id || undefined,
      });
      showNotification(`Stock adjusted: ${adjustingItem.sku} now has ${res.quantity} on hand`);
      setAdjustingItem(null);
      await loadItems();
    } catch (err: any) {
      setError(err.message || 'Failed to adjust stock');
    }
  };

  const filteredItems = items.filter((it) => {
    const matchesSector = selectedSector === 'ALL' || it.sector === selectedSector;
    const q = search.toLowerCase();
    const matchesSearch =
      !q ||
      it.name.toLowerCase().includes(q) ||
      it.sku.toLowerCase().includes(q) ||
      (it.barcode && it.barcode.toLowerCase().includes(q));
    return matchesSector && matchesSearch;
  });

  return (
    <div className="flex-1 flex flex-col bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 overflow-hidden transition-colors">
      {/* Top Bar */}
      <div className="p-4 sm:p-6 border-b border-slate-200 dark:border-slate-800 flex flex-wrap items-center justify-between gap-4 bg-white dark:bg-slate-900/60">
        <div>
          <h1 className="text-lg sm:text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <Package className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            Catalog &amp; Inventory Management
          </h1>
          <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
            Manage multi-sector items, atomic pricing, barcodes, and inventory adjustments
          </p>
        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          <button
            type="button"
            onClick={loadItems}
            disabled={loading}
            className="min-h-[44px] min-w-[44px] p-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 flex items-center justify-center transition"
            title="Refresh items"
            aria-label="Refresh items"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </button>
          <button
            type="button"
            onClick={handleOpenAdd}
            className="min-h-[44px] px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-500 text-white font-semibold text-xs sm:text-sm flex items-center gap-2 shadow-lg shadow-blue-600/20 transition"
          >
            <Plus className="w-4 h-4" />
            <span>Add Product</span>
          </button>
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
            placeholder="Search by name, SKU, or barcode..."
            className="w-full min-h-[44px] pl-10 pr-4 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition"
          />
        </div>

        <div className="flex flex-wrap gap-1.5 items-center">
          <Filter className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400 mr-1" />
          {SECTORS.map((sec) => (
            <button
              key={sec.key}
              type="button"
              onClick={() => setSelectedSector(sec.key)}
              className={`min-h-[36px] px-3 py-1.5 rounded-lg text-xs font-semibold border transition-all ${
                selectedSector === sec.key
                  ? 'bg-blue-600 border-blue-600 text-white shadow-sm'
                  : 'bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-800 text-slate-700 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:border-slate-400 dark:hover:border-slate-700'
              }`}
            >
              {sec.label}
            </button>
          ))}
        </div>
      </div>

      {/* Table Container */}
      <div className="flex-1 overflow-auto p-3 sm:p-4 lg:p-6">
        <div className="border border-slate-200 dark:border-slate-800 rounded-2xl overflow-x-auto bg-white dark:bg-slate-900/40 shadow-sm">
          <table className="w-full min-w-[680px] text-left border-collapse">
            <thead>
              <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/80 text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                <th className="px-4 py-3">Product / SKU</th>
                <th className="px-4 py-3">Sector</th>
                <th className="px-4 py-3">Barcode</th>
                <th className="px-4 py-3">Price</th>
                <th className="px-4 py-3">Tax</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800/60 text-sm">
              {filteredItems.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-slate-600 dark:text-slate-400">
                    {loading ? 'Loading catalog items...' : 'No catalog items found matching your filter.'}
                  </td>
                </tr>
              ) : (
                filteredItems.map((item) => {
                  const secBadge = SECTORS.find((s) => s.key === item.sector);
                  return (
                    <tr key={item.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/40 transition-colors">
                      <td className="px-4 py-3.5">
                        <div className="font-semibold text-slate-900 dark:text-slate-200">{item.name}</div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 font-mono mt-0.5">{item.sku}</div>
                      </td>
                      <td className="px-4 py-3.5">
                        <span
                          className={`inline-block px-2 py-0.5 rounded text-[11px] font-semibold border ${
                            secBadge ? secBadge.color : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                          }`}
                        >
                          {item.sector}
                        </span>
                      </td>
                      <td className="px-4 py-3.5 font-mono text-xs text-slate-600 dark:text-slate-400">
                        {item.barcode || <span className="text-slate-400 dark:text-slate-600">—</span>}
                      </td>
                      <td className="px-4 py-3.5 font-semibold text-slate-900 dark:text-slate-200">
                        {item.currency || 'IDR'} {Number(item.base_price).toLocaleString()}
                      </td>
                      <td className="px-4 py-3.5 text-xs text-slate-600 dark:text-slate-400">
                        {((item.tax_rate ?? 0.11) * 100).toFixed(0)}%
                      </td>
                      <td className="px-4 py-3.5 text-right space-x-2 whitespace-nowrap">
                        <button
                          type="button"
                          onClick={() => {
                            setAdjustingItem(item);
                            setAdjustForm({ delta_qty: 10, reason: 'RESTOCK', ref_id: '' });
                          }}
                          className="min-h-[36px] px-2.5 py-1 rounded-lg bg-amber-50 dark:bg-slate-800 hover:bg-amber-100 dark:hover:bg-slate-700 text-amber-800 dark:text-amber-300 border border-amber-200 dark:border-slate-700 text-xs font-semibold inline-flex items-center gap-1.5 transition"
                          title="Adjust stock"
                        >
                          <TrendingUp className="w-3.5 h-3.5" />
                          <span>Stock</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => handleOpenEdit(item)}
                          className="min-h-[36px] min-w-[36px] p-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 inline-flex items-center justify-center transition"
                          title="Edit product"
                          aria-label={`Edit ${item.name}`}
                        >
                          <Edit2 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeletingItem(item)}
                          className="min-h-[36px] min-w-[36px] p-2 rounded-lg bg-rose-50 dark:bg-rose-500/10 hover:bg-rose-100 dark:hover:bg-rose-500/20 text-rose-700 dark:text-rose-400 border border-rose-200 dark:border-rose-500/30 inline-flex items-center justify-center transition"
                          title="Delete product"
                          aria-label={`Delete ${item.name}`}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add / Edit Product Modal */}
      {(isAddOpen || editingItem) && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-lg max-h-[90dvh] overflow-y-auto p-5 sm:p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
              <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <Package className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                {editingItem ? 'Edit Product' : 'Add New Product'}
              </h2>
              <button
                type="button"
                onClick={() => {
                  setIsAddOpen(false);
                  setEditingItem(null);
                }}
                aria-label="Close modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSaveItem} className="mt-4 space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">SKU *</label>
                  <input
                    type="text"
                    required
                    disabled={!!editingItem}
                    value={itemForm.sku}
                    onChange={(e) => setItemForm({ ...itemForm, sku: e.target.value.toUpperCase() })}
                    placeholder="e.g. COFFEE-BEANS-01"
                    className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 font-mono focus:border-blue-500 focus:outline-none disabled:opacity-50"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Barcode (UPC/EAN)</label>
                  <input
                    type="text"
                    value={itemForm.barcode || ''}
                    onChange={(e) => setItemForm({ ...itemForm, barcode: e.target.value })}
                    placeholder="e.g. 8991234567890"
                    className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 font-mono focus:border-blue-500 focus:outline-none"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Product Name *</label>
                <input
                  type="text"
                  required
                  value={itemForm.name}
                  onChange={(e) => setItemForm({ ...itemForm, name: e.target.value })}
                  placeholder="e.g. House Blend Espresso Beans"
                  className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-blue-500 focus:outline-none"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Sector</label>
                  <select
                    disabled={!!editingItem}
                    value={itemForm.sector}
                    onChange={(e) => setItemForm({ ...itemForm, sector: e.target.value as ItemSector })}
                    className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-blue-500 focus:outline-none disabled:opacity-50"
                  >
                    <option value="RETAIL">Retail</option>
                    <option value="FB">F&amp;B</option>
                    <option value="TOURISM">Tourism</option>
                    <option value="HOSPITALITY">Hospitality</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Base Price *</label>
                  <input
                    type="number"
                    required
                    min={0}
                    step={100}
                    value={itemForm.base_price}
                    onChange={(e) => setItemForm({ ...itemForm, base_price: Number(e.target.value) })}
                    className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-blue-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Tax Rate</label>
                  <input
                    type="number"
                    step={0.01}
                    min={0}
                    max={1}
                    value={itemForm.tax_rate ?? 0.11}
                    onChange={(e) => setItemForm({ ...itemForm, tax_rate: Number(e.target.value) })}
                    className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-blue-500 focus:outline-none"
                  />
                </div>
              </div>

              <div className="pt-4 border-t border-slate-200 dark:border-slate-800 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setIsAddOpen(false);
                    setEditingItem(null);
                  }}
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-sm font-semibold transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-500 text-white text-sm font-semibold shadow-lg shadow-blue-600/20 transition"
                >
                  {editingItem ? 'Save Changes' : 'Create Product'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Adjust Stock Modal */}
      {adjustingItem && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md max-h-[90dvh] overflow-y-auto p-5 sm:p-6 shadow-2xl">
            <div className="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
              <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <TrendingUp className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                Adjust Stock
              </h2>
              <button
                type="button"
                onClick={() => setAdjustingItem(null)}
                aria-label="Close modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-3 p-3 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-200 dark:border-slate-800">
              <div className="font-semibold text-slate-900 dark:text-slate-200">{adjustingItem.name}</div>
              <div className="text-xs text-slate-600 dark:text-slate-400 font-mono mt-0.5">SKU: {adjustingItem.sku}</div>
            </div>

            <form onSubmit={handleStockAdjust} className="mt-4 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Reason</label>
                <select
                  value={adjustForm.reason}
                  onChange={(e) => setAdjustForm({ ...adjustForm, reason: e.target.value })}
                  className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 focus:border-amber-500 focus:outline-none"
                >
                  <option value="RESTOCK">RESTOCK (Inbound Delivery)</option>
                  <option value="AUDIT">AUDIT (Physical Stocktake Count)</option>
                  <option value="DAMAGE">DAMAGE (Damaged / Broken)</option>
                  <option value="RETURN">RETURN (Customer Return)</option>
                  <option value="TRANSFER">TRANSFER (Outlet Transfer)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">
                  Delta Quantity (Negative for deductions, Positive for additions) *
                </label>
                <input
                  type="number"
                  step="any"
                  required
                  value={adjustForm.delta_qty}
                  onChange={(e) => setAdjustForm({ ...adjustForm, delta_qty: Number(e.target.value) })}
                  className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 font-mono focus:border-amber-500 focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-400 mb-1">Reference ID / PO Number</label>
                <input
                  type="text"
                  value={adjustForm.ref_id}
                  onChange={(e) => setAdjustForm({ ...adjustForm, ref_id: e.target.value })}
                  placeholder="e.g. PO-2026-0042"
                  className="w-full min-h-[44px] px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-sm text-slate-900 dark:text-slate-100 font-mono focus:border-amber-500 focus:outline-none"
                />
              </div>

              <div className="pt-4 border-t border-slate-200 dark:border-slate-800 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setAdjustingItem(null)}
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-sm font-semibold transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="min-h-[44px] px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-700 dark:hover:bg-amber-500 text-white text-sm font-semibold shadow-lg shadow-amber-600/20 transition"
                >
                  Confirm Adjustment
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deletingItem && (
        <div role="dialog" aria-modal="true" className="fixed inset-0 bg-slate-950/60 dark:bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-sm p-6 shadow-2xl">
            <div className="flex items-center gap-3 text-rose-700 dark:text-rose-400 mb-3">
              <div className="w-10 h-10 rounded-full bg-rose-50 dark:bg-rose-500/15 border border-rose-200 dark:border-rose-500/30 flex items-center justify-center shrink-0">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <h3 className="font-bold text-slate-900 dark:text-slate-100 text-lg">Delete Product</h3>
            </div>
            <p className="text-sm text-slate-700 dark:text-slate-300">
              Are you sure you want to deactivate and remove <span className="font-semibold text-slate-900 dark:text-white">"{deletingItem.name}"</span>?
              Downstream edge nodes will synchronize the deletion without breaking past order records.
            </p>
            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setDeletingItem(null)}
                className="min-h-[44px] px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 text-sm font-semibold transition"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDelete}
                className="min-h-[44px] px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-700 dark:hover:bg-rose-500 text-white text-sm font-semibold shadow-lg shadow-rose-600/20 transition"
              >
                Delete Product
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
