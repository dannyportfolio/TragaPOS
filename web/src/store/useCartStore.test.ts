import { describe, it, expect, beforeEach } from 'vitest';
import { useCartStore } from './useCartStore';
import { Item } from '../domain/types';

describe('useCartStore', () => {
  beforeEach(() => {
    useCartStore.getState().clearCart();
  });

  const sampleLatte: Item = {
    id: 'item-1',
    tenant_id: 'tenant-1',
    outlet_id: 'outlet-1',
    sku: 'COF-LATTE-01',
    name: 'Caffe Latte',
    sector: 'FB',
    base_price: 30000,
    currency: 'IDR',
    tax_rate: 0.1,
    sector_traits: {},
    version: 1,
    is_active: true,
  };

  const sampleChips: Item = {
    id: 'item-2',
    tenant_id: 'tenant-1',
    outlet_id: 'outlet-1',
    sku: 'SNK-CHIPS-01',
    name: 'Potato Chips',
    sector: 'RETAIL',
    base_price: 15000,
    currency: 'IDR',
    tax_rate: 0.1,
    sector_traits: {},
    version: 1,
    is_active: true,
  };

  it('adds item to cart with accurate price and tax calculations', () => {
    const store = useCartStore.getState();
    store.addItem(sampleChips);

    const items = useCartStore.getState().items;
    expect(items.length).toBe(1);
    expect(items[0].quantity).toBe(1);
    expect(items[0].subtotal).toBe(15000);
    expect(items[0].taxAmount).toBe(1500);
    expect(items[0].totalAmount).toBe(16500);

    expect(useCartStore.getState().getSubtotal()).toBe(15000);
    expect(useCartStore.getState().getTaxAmount()).toBe(1500);
    expect(useCartStore.getState().getTotalAmount()).toBe(16500);
  });

  it('correctly calculates item modifiers price delta', () => {
    const store = useCartStore.getState();
    store.addItem(sampleLatte, [
      { name: 'Extra Shot', price_delta: 5000 },
      { name: 'Oat Milk', price_delta: 8000 },
    ]);

    const items = useCartStore.getState().items;
    expect(items.length).toBe(1);
    // Base 30,000 + 5,000 + 8,000 = 43,000
    expect(items[0].unitPrice).toBe(43000);
    expect(items[0].subtotal).toBe(43000);
    expect(items[0].taxAmount).toBe(4300);
    expect(items[0].totalAmount).toBe(47300);
  });

  it('updates quantity and recalculates totals', () => {
    const store = useCartStore.getState();
    store.addItem(sampleChips);
    store.updateQuantity(0, 2); // now 3

    const items = useCartStore.getState().items;
    expect(items[0].quantity).toBe(3);
    expect(items[0].subtotal).toBe(45000);
    expect(items[0].taxAmount).toBe(4500);
    expect(items[0].totalAmount).toBe(49500);
  });

  it('removes item when quantity reaches zero', () => {
    const store = useCartStore.getState();
    store.addItem(sampleChips);
    store.updateQuantity(0, -1); // 1 -> 0, should be removed

    expect(useCartStore.getState().items.length).toBe(0);
    expect(useCartStore.getState().getTotalAmount()).toBe(0);
  });
});
