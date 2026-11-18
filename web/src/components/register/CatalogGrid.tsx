import React, { useState } from 'react';
import { Search, Barcode, Coffee, ShoppingBag, Ticket, BedDouble, X } from 'lucide-react';
import { Item, ItemSector } from '../../domain/types';

interface CatalogGridProps {
  items: Item[];
  onSelectItem: (item: Item) => void;
  onBarcodeScan: (barcode: string) => void;
  isLoading: boolean;
  hasMobileCartBar?: boolean;
}

export const CatalogGrid: React.FC<CatalogGridProps> = ({
  items,
  onSelectItem,
  onBarcodeScan,
  isLoading,
  hasMobileCartBar = false,
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
    <div className="flex-1 flex flex-col h-full min-w-0 overflow-hidden bg-slate-50 dark:bg-slate-950 p-3 sm:p-4 lg:p-6 transition-colors">
      {/* Search & Barcode Bar */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-2.5 sm:gap-3 mb-3 sm:mb-4">
        {/* Keyword Search */}
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500 dark:text-slate-400 pointer-events-none" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search items by name, SKU, or tag..."
            aria-label="Search catalog items"
            className="w-full min-h-[44px] pl-10 pr-10 py-2.5 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-indigo-500 placeholder:text-slate-500 dark:placeholder:text-slate-500 transition-colors"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => setSearchQuery('')}
              aria-label="Clear search"
              className="absolute right-1.5 top-1/2 -translate-y-1/2 min-w-[36px] min-h-[36px] rounded-lg text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 flex items-center justify-center"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Rapid Barcode Scanner Listener */}
        <form onSubmit={handleBarcodeSubmit} className="relative">
          <Barcode className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-emerald-600 dark:text-emerald-400 pointer-events-none" />
          <input
            type="text"
            value={barcodeInput}
            onChange={(e) => setBarcodeInput(e.target.value)}
            placeholder="Scan barcode or type & press Enter..."
            aria-label="Barcode scanner input"
            className="w-full min-h-[44px] pl-10 pr-3.5 py-2.5 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs sm:text-sm focus:outline-none focus:border-emerald-500 font-mono placeholder:text-slate-500 dark:placeholder:text-slate-500 transition-colors"
          />
        </form>
      </div>

      {/* Sector Filter Tabs */}
      <div className="flex gap-2 overflow-x-auto pb-2 mb-3 sm:mb-4 shrink-0">
        {sectors.map((s) => {
          const Icon = s.icon;
          const isSelected = selectedSector === s.id;
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => setSelectedSector(s.id)}
              className={`min-h-[44px] px-3.5 py-2 rounded-xl border text-xs font-semibold whitespace-nowrap flex items-center gap-2 transition-all shrink-0 ${
                isSelected
                  ? 'bg-indigo-600 border-indigo-600 text-white shadow-md shadow-indigo-600/20'
                  : 'bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 hover:border-slate-400 dark:hover:border-slate-700'
              }`}
            >
              {Icon && <Icon className="w-3.5 h-3.5" />}
              <span>{s.label}</span>
            </button>
          );
        })}
      </div>

      {/* Item Grid */}
      <div className={`flex-1 overflow-y-auto pr-1 ${hasMobileCartBar ? 'pb-20 lg:pb-2' : 'pb-2'}`}>
        {isLoading ? (
          <div className="h-64 flex items-center justify-center text-slate-600 dark:text-slate-400 text-xs sm:text-sm">
            Loading catalog items from local Edge SQLite database...
          </div>
        ) : filteredItems.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-slate-600 dark:text-slate-400 text-xs text-center border-2 border-dashed border-slate-300 dark:border-slate-800 rounded-2xl p-6 bg-white/60 dark:bg-slate-900/40">
            <ShoppingBag className="w-8 h-8 text-slate-500 dark:text-slate-500 mb-2" />
            <p className="font-semibold text-slate-800 dark:text-slate-200 text-sm">No items found</p>
            <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">Try searching with a different SKU, name, or barcode</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3">
            {filteredItems.map((item) => {
              const sectorColor =
                item.sector === 'FB'
                  ? 'text-amber-800 bg-amber-50 border-amber-200 dark:text-amber-300 dark:bg-amber-400/10 dark:border-amber-400/20'
                  : item.sector === 'RETAIL'
                  ? 'text-sky-800 bg-sky-50 border-sky-200 dark:text-sky-300 dark:bg-sky-400/10 dark:border-sky-400/20'
                  : item.sector === 'TOURISM'
                  ? 'text-emerald-800 bg-emerald-50 border-emerald-200 dark:text-emerald-300 dark:bg-emerald-400/10 dark:border-emerald-400/20'
                  : 'text-purple-800 bg-purple-50 border-purple-200 dark:text-purple-300 dark:bg-purple-400/10 dark:border-purple-400/20';

              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onSelectItem(item)}
                  className="min-h-[120px] bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-indigo-500 dark:hover:border-indigo-500/60 rounded-xl p-3.5 flex flex-col justify-between text-left transition-all hover:shadow-md dark:hover:shadow-slate-950/40 group active:scale-[0.99]"
                >
                  <div className="w-full">
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${sectorColor}`}>
                        {item.sector}
                      </span>
                      {item.barcode && (
                        <span className="text-[10px] text-slate-600 dark:text-slate-400 font-mono flex items-center gap-1 truncate">
                          <Barcode className="w-3 h-3 shrink-0" />
                          <span className="truncate">{item.barcode}</span>
                        </span>
                      )}
                    </div>
                    <h4 className="font-semibold text-slate-900 dark:text-slate-100 text-xs sm:text-sm line-clamp-2 group-hover:text-indigo-700 dark:group-hover:text-indigo-300 transition-colors">
                      {item.name}
                    </h4>
                    <div className="text-[11px] text-slate-600 dark:text-slate-400 font-mono mt-0.5">{item.sku}</div>
                  </div>

                  <div className="w-full mt-3 pt-2.5 border-t border-slate-100 dark:border-slate-800/80 flex items-center justify-between">
                    <span className="text-xs sm:text-sm font-mono font-bold text-slate-900 dark:text-slate-100">
                      {formatIDR(item.base_price)}
                    </span>
                    <span className="text-xs text-indigo-700 dark:text-indigo-400 font-semibold group-hover:underline">
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
