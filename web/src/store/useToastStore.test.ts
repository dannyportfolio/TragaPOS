import { describe, it, expect, beforeEach } from 'vitest';
import { useToastStore, toast } from './useToastStore';

describe('useToastStore', () => {
  beforeEach(() => {
    useToastStore.getState().clearAll();
  });

  it('adds and removes toast notifications', () => {
    toast.success('Order completed');
    toast.error('Payment failed');

    const toasts = useToastStore.getState().toasts;
    expect(toasts).toHaveLength(2);
    expect(toasts[0].type).toBe('success');
    expect(toasts[1].type).toBe('error');

    useToastStore.getState().removeToast(toasts[0].id);
    expect(useToastStore.getState().toasts).toHaveLength(1);
  });
});
