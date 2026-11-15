import React, { useState } from 'react';
import { Search, Barcode, Coffee, ShoppingBag, Ticket, BedDouble } from 'lucide-react';
import { Item, ItemSector } from '../../domain/types';

interface CatalogGridProps {
  items: Item[];
  onSelectItem: (item: Item) => void;
  onBarcodeScan: (barcode: string) => void;
  isLoading: boolean;
}

export const CatalogGrid: React.FC<CatalogGridProps> = ({
  items,
  onSelectItem,
  onBarcodeScan,
  isLoading,
}) => {
  const [selectedSector, setSelectedSector] = useState<ItemSector | 'ALL'>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [barcodeInput, setBarcodeInput] = useState('');

  const sectors: { id: ItemSector | 'ALL'; label: string; icon: any }[] = [
    { id: 'ALL', label: 'All Items', icon: null },
    { id: 'FB', label: 'F&B (Cafe/Resto)', icon: Coffee },
    { id: 'RETAIL', label: 'Retail (Mart)', icon: ShoppingBag },
    { id: 'TOURISM', label: 'Tourism (Gate)', icon: Ticket },
    { id: 'HOSPITALITY', label: 'Hotel & Resort', icon: BedDouble },
  ];

  const filteredItems = items.filter((item) => {
    const matchesSector = selectedSector === 'ALL' || item.sector === selectedSector;
    const matchesQuery =
      searchQuery === '' ||
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.sku.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (item.barcode && item.barcode.includes(searchQuery));
    return matchesSector && matchesQuery;
  });

  const handleBarcodeSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (barcodeInput.trim()) {
      onBarcodeScan(barcodeInput.trim());
      setBarcodeInput('');
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
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-slate-950 p-6">
      {/* Search & Barcode Bar */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
        {/* Keyword Search */}
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3.5 top-3 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search items by name, SKU, or tag..."
            className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-200 text-xs focus:outline-none focus:border-indigo-500 placeholder:text-slate-600 transition-colors"
          />
        </div>

        {/* Rapid Barcode Scanner Listener */}
        <form onSubmit={handleBarcodeSubmit} className="relative">
          <Barcode className="w-4 h-4 absolute left-3.5 top-3 text-emerald-400" />
          <input
            type="text"
            value={barcodeInput}
            onChange={(e) => setBarcodeInput(e.target.value)}
            placeholder="Scan barcode scanner or type & press Enter..."
            className="w-full pl-10 pr-3.5 py-2.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-200 text-xs focus:outline-none focus:border-emerald-500 font-mono placeholder:text-slate-600 transition-colors"
          />
        </form>
      </div>

      {/* Sector Filter Tabs */}
      <div className="flex gap-2 overflow-x-auto pb-2 mb-4 scrollbar-none">
        {sectors.map((s) => {
          const Icon = s.icon;
          const isSelected = selectedSector === s.id;
          return (
            <button
              key={s.id}
              onClick={() => setSelectedSector(s.id)}
              className={`px-3 py-2 rounded-xl border text-xs font-medium whitespace-nowrap flex items-center gap-2 transition-all ${
                isSelected
                  ? 'bg-indigo-600 border-indigo-500 text-white shadow-md shadow-indigo-600/20'
                  : 'bg-slate-900 border-slate-800 text-slate-400 hover:text-slate-200 hover:border-slate-700'
              }`}
            >
              {Icon && <Icon className="w-3.5 h-3.5" />}
              <span>{s.label}</span>
            </button>
          );
        })}
      </div>

      {/* Item Grid */}
      <div className="flex-1 overflow-y-auto pr-1">
        {isLoading ? (
          <div className="h-64 flex items-center justify-center text-slate-500 text-xs">
            Loading catalog items from local Edge SQLite database...
          </div>
        ) : filteredItems.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-slate-500 text-xs text-center border-2 border-dashed border-slate-800/80 rounded-2xl p-6">
            <ShoppingBag className="w-8 h-8 text-slate-600 mb-2" />
            <p className="font-medium text-slate-400">No items found</p>
            <p className="text-[11px] text-slate-600 mt-0.5">Try searching with a different SKU or barcode</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {filteredItems.map((item) => {
              const sectorColor =
                item.sector === 'FB'
                  ? 'text-amber-400 bg-amber-400/10 border-amber-400/20'
                  : item.sector === 'RETAIL'
                  ? 'text-sky-400 bg-sky-400/10 border-sky-400/20'
                  : item.sector === 'TOURISM'
                  ? 'text-emerald-400 bg-emerald-400/10 border-emerald-400/20'
                  : 'text-purple-400 bg-purple-400/10 border-purple-400/20';

              return (
                <button
                  key={item.id}
                  onClick={() => onSelectItem(item)}
                  className="bg-slate-900 border border-slate-800 hover:border-indigo-500/50 rounded-xl p-3.5 flex flex-col justify-between text-left transition-all hover:shadow-lg hover:shadow-slate-950/40 group active:scale-[0.98]"
                >
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${sectorColor}`}>
                        {item.sector}
                      </span>
                      {item.barcode && (
                        <span className="text-[10px] text-slate-500 font-mono flex items-center gap-1">
                          <Barcode className="w-3 h-3 text-slate-600" />
                          {item.barcode}
                        </span>
                      )}
                    </div>
                    <h4 className="font-medium text-slate-100 text-xs line-clamp-2 group-hover:text-indigo-300 transition-colors">
                      {item.name}
                    </h4>
                    <div className="text-[10px] text-slate-500 font-mono mt-0.5">{item.sku}</div>
                  </div>

                  <div className="mt-3 pt-2.5 border-t border-slate-800/80 flex items-center justify-between">
                    <span className="text-xs font-mono font-bold text-slate-200">
                      {formatIDR(item.base_price)}
                    </span>
                    <span className="text-[10px] text-indigo-400 font-medium group-hover:underline">
                      + Add
                    </span>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
