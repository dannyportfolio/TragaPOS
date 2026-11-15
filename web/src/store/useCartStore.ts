import { create } from 'zustand';
import { Item, CartItem, ItemModifierOption, Order } from '../domain/types';

interface CartState {
  items: CartItem[];
  heldOrderId?: string;
  notes: string;
  
  addItem: (item: Item, modifiers?: ItemModifierOption[]) => void;
  updateQuantity: (index: number, delta: number) => void;
  removeItem: (index: number) => void;
  clearCart: () => void;
  setNotes: (notes: string) => void;
  loadHeldOrder: (order: Order) => void;

  getSubtotal: () => number;
  getTaxAmount: () => number;
  getTotalAmount: () => number;
}

export const useCartStore = create<CartState>((set, get) => ({
  items: [],
  heldOrderId: undefined,
  notes: '',

  addItem: (item, modifiers = []) => {
    set((state) => {
      const modDelta = modifiers.reduce((acc, m) => acc + m.price_delta, 0);
      const unitPrice = item.base_price + modDelta;

      // Check if identical item with identical modifiers already exists
      const existingIdx = state.items.findIndex(
        (ci) =>
          ci.item.id === item.id &&
          JSON.stringify(ci.selectedModifiers) === JSON.stringify(modifiers)
      );

      if (existingIdx >= 0) {
        const nextItems = [...state.items];
        const existing = nextItems[existingIdx];
        const newQty = existing.quantity + 1;
        const subtotal = unitPrice * newQty;
        const taxAmount = Math.round(subtotal * item.tax_rate);

        nextItems[existingIdx] = {
          ...existing,
          quantity: newQty,
          subtotal,
          taxAmount,
          totalAmount: subtotal + taxAmount,
        };
        return { items: nextItems };
      }

      const quantity = 1;
      const subtotal = unitPrice * quantity;
      const taxAmount = Math.round(subtotal * item.tax_rate);

      return {
        items: [
          ...state.items,
          {
            item,
            quantity,
            selectedModifiers: modifiers,
            unitPrice,
            subtotal,
            taxAmount,
            totalAmount: subtotal + taxAmount,
          },
        ],
      };
    });
  },

  updateQuantity: (index, delta) => {
    set((state) => {
      const nextItems = [...state.items];
      const target = nextItems[index];
      if (!target) return state;

      const nextQty = target.quantity + delta;
      if (nextQty <= 0) {
        return { items: nextItems.filter((_, i) => i !== index) };
      }

      const subtotal = target.unitPrice * nextQty;
      const taxAmount = Math.round(subtotal * target.item.tax_rate);

      nextItems[index] = {
        ...target,
        quantity: nextQty,
        subtotal,
        taxAmount,
        totalAmount: subtotal + taxAmount,
      };

      return { items: nextItems };
    });
  },

  removeItem: (index) => {
    set((state) => ({
      items: state.items.filter((_, i) => i !== index),
    }));
  },

  clearCart: () => {
    set({ items: [], heldOrderId: undefined, notes: '' });
  },

  setNotes: (notes) => set({ notes }),

  loadHeldOrder: (order) => {
    const cartItems: CartItem[] = order.items.map((oi) => ({
      item: {
        id: oi.item_id,
        tenant_id: order.tenant_id,
        outlet_id: order.outlet_id,
        sku: oi.sku,
        name: oi.name,
        sector: oi.sector,
        base_price: oi.unit_price,
        currency: order.currency,
        tax_rate: oi.tax_amount > 0 && oi.subtotal > 0 ? oi.tax_amount / oi.subtotal : 0,
        sector_traits: {},
        version: 1,
        is_active: true,
      },
      quantity: oi.quantity,
      selectedModifiers: oi.selected_modifiers || [],
      unitPrice: oi.unit_price,
      subtotal: oi.subtotal,
      taxAmount: oi.tax_amount,
      totalAmount: oi.total_amount,
    }));

    set({
      items: cartItems,
      heldOrderId: order.id,
      notes: order.notes || '',
    });
  },

  getSubtotal: () => {
    return get().items.reduce((sum, ci) => sum + ci.subtotal, 0);
  },

  getTaxAmount: () => {
    return get().items.reduce((sum, ci) => sum + ci.taxAmount, 0);
  },

  getTotalAmount: () => {
    return get().items.reduce((sum, ci) => sum + ci.totalAmount, 0);
  },
}));
