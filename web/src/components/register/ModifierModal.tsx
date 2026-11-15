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

  // Extract modifiers from sector traits if present, otherwise default common modifiers
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
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md overflow-hidden shadow-2xl">
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
          <div>
            <h3 className="font-semibold text-slate-100 text-sm">Customize: {item.name}</h3>
            <p className="text-xs text-slate-400">Select recipe modifiers & variants</p>
          </div>
          <button
            onClick={onClose}
            className="w-8 h-8 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div className="space-y-2">
            {availableModifiers.map((mod) => {
              const isSelected = selectedMods.some((m) => m.name === mod.name);
              return (
                <button
                  key={mod.name}
                  onClick={() => toggleModifier(mod)}
                  className={`w-full p-3 rounded-xl border flex items-center justify-between text-xs transition-colors ${
                    isSelected
                      ? 'bg-indigo-600/10 border-indigo-500/50 text-indigo-200'
                      : 'bg-slate-950/60 border-slate-800 text-slate-300 hover:border-slate-700'
                  }`}
                >
                  <div className="flex items-center gap-2.5">
                    <div
                      className={`w-4 h-4 rounded border flex items-center justify-center ${
                        isSelected
                          ? 'bg-indigo-600 border-indigo-500 text-white'
                          : 'border-slate-700 bg-slate-900'
                      }`}
                    >
                      {isSelected && <Check className="w-3 h-3" />}
                    </div>
                    <span className="font-medium">{mod.name}</span>
                  </div>
                  <span className="font-mono text-slate-400">
                    {mod.price_delta > 0 ? `+${formatIDR(mod.price_delta)}` : 'Free'}
                  </span>
                </button>
              );
            })}
          </div>

          <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex justify-between items-center text-xs">
            <span className="text-slate-400">Updated Price:</span>
            <span className="font-mono font-bold text-slate-100 text-sm">{formatIDR(finalUnitPrice)}</span>
          </div>

          <div className="flex gap-2 pt-2">
            <button
              onClick={onClose}
              className="flex-1 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium text-xs transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                onConfirm(item, selectedMods);
                onClose();
              }}
              className="flex-1 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs transition-colors shadow-lg shadow-indigo-600/20"
            >
              Add to Cart
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
