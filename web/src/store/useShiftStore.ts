import { create } from 'zustand';
import { ShiftSummary, CashMovementType } from '../domain/types';
import { api } from '../services/api';

interface ShiftState {
  shift: ShiftSummary | null;
  isLoading: boolean;
  error: string | null;

  fetchShift: () => Promise<void>;
  openShift: (openingFloat: number, cashierName: string) => Promise<void>;
  recordCashMovement: (type: CashMovementType, amount: number, reason: string) => Promise<void>;
  closeShift: (actualCash: number, notes?: string) => Promise<ShiftSummary>;
}

export const useShiftStore = create<ShiftState>((set, get) => ({
  shift: null,
  isLoading: false,
  error: null,

  fetchShift: async () => {
    set({ isLoading: true, error: null });
    try {
      const shift = await api.getCurrentShift('default_cashier');
      set({ shift, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  openShift: async (openingFloat: number, cashierName: string) => {
    set({ isLoading: true, error: null });
    try {
      await api.openShift({
        cashier_id: 'default_cashier',
        cashier_name: cashierName,
        opening_float: openingFloat,
      });
      await get().fetchShift();
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  recordCashMovement: async (type: CashMovementType, amount: number, reason: string) => {
    set({ isLoading: true, error: null });
    try {
      await api.recordCashMovement({
        type,
        amount,
        reason,
        cashier_id: 'default_cashier',
      });
      await get().fetchShift();
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  closeShift: async (actualCash: number, notes?: string) => {
    set({ isLoading: true, error: null });
    try {
      const summary = await api.closeShift({
        actual_cash: actualCash,
        notes,
        cashier_id: 'default_cashier',
      });
      set({ shift: null, isLoading: false });
      return summary;
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },
}));
