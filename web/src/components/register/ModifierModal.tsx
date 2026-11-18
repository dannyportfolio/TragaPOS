import React, { useState } from 'react';
import { X, Check } from 'lucide-react';
import { Item, ItemModifierOption } from '../../domain/types';

interface ModifierModalProps {
  item: Item | null;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (item: Item, selectedModifiers: ItemModifierOption[]) => void;
}

export const ModifierModal: React.FC<ModifierModalProps> = ({
  item,
  isOpen,
  onClose,
  onConfirm,
}) => {
  const [selectedMods, setSelectedMods] = useState<ItemModifierOption[]>([]);

  if (!isOpen || !item) return null;

  const availableModifiers: ItemModifierOption[] =
    item.sector_traits?.modifiers || [
      { name: 'Extra Shot', price_delta: 5000 },
      { name: 'Oat Milk Sub', price_delta: 8000 },
      { name: 'Vanilla Syrup', price_delta: 4000 },
      { name: 'Less Sugar (Normal)', price_delta: 0 },
      { name: 'Less Ice (70%)', price_delta: 0 },
    ];

  const toggleModifier = (mod: ItemModifierOption) => {
    if (selectedMods.some((m) => m.name === mod.name)) {
      setSelectedMods(selectedMods.filter((m) => m.name !== mod.name));
    } else {
      setSelectedMods([...selectedMods, mod]);
    }
  };

  const deltaTotal = selectedMods.reduce((acc, m) => acc + m.price_delta, 0);
  const finalUnitPrice = item.base_price + deltaTotal;

  const formatIDR = (val: number) => {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
    }).format(val);
  };

  return (
    <div role="dialog" aria-modal="true" className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-md max-h-[90dvh] flex flex-col overflow-hidden shadow-2xl">
        <div className="px-5 sm:px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-slate-900 dark:text-slate-100 text-sm">Customize: {item.name}</h3>
            <p className="text-xs text-slate-600 dark:text-slate-400">Select recipe modifiers &amp; variants</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close modal"
            className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 sm:p-6 space-y-4 overflow-y-auto">
          <div className="space-y-2">
            {availableModifiers.map((mod) => {
              const isSelected = selectedMods.some((m) => m.name === mod.name);
              return (
                <button
                  key={mod.name}
                  type="button"
                  onClick={() => toggleModifier(mod)}
                  className={`w-full min-h-[44px] p-3 rounded-xl border flex items-center justify-between text-xs transition-colors ${
                    isSelected
                      ? 'bg-indigo-50 dark:bg-indigo-600/15 border-indigo-400 dark:border-indigo-500/50 text-indigo-950 dark:text-indigo-200'
                      : 'bg-slate-50 dark:bg-slate-950/60 border-slate-200 dark:border-slate-800 text-slate-800 dark:text-slate-300 hover:border-slate-300 dark:hover:border-slate-700'
                  }`}
                >
                  <div className="flex items-center gap-2.5">
                    <div
                      className={`w-4 h-4 rounded border flex items-center justify-center ${
                        isSelected
                          ? 'bg-indigo-600 border-indigo-600 text-white'
                          : 'border-slate-400 dark:border-slate-700 bg-white dark:bg-slate-900'
                      }`}
                    >
                      {isSelected && <Check className="w-3 h-3" />}
                    </div>
                    <span className="font-semibold">{mod.name}</span>
                  </div>
                  <span className="font-mono text-slate-600 dark:text-slate-400">
                    {mod.price_delta > 0 ? `+${formatIDR(mod.price_delta)}` : 'Free'}
                  </span>
                </button>
              );
            })}
          </div>

          <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 flex justify-between items-center text-xs">
            <span className="text-slate-600 dark:text-slate-400 font-medium">Updated Price:</span>
            <span className="font-mono font-bold text-slate-900 dark:text-slate-100 text-sm">{formatIDR(finalUnitPrice)}</span>
          </div>

          <div className="flex gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 min-h-[44px] py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-300 font-semibold text-xs transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                onConfirm(item, selectedMods);
                onClose();
              }}
              className="flex-1 min-h-[44px] py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 dark:hover:bg-indigo-500 text-white font-semibold text-xs transition-colors shadow-lg shadow-indigo-600/20"
            >
              Add to Cart
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
