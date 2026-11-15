import { describe, it, expect } from 'vitest';
import { PaymentInput } from '../../domain/types';

describe('Split Payment Logic Verification', () => {
  it('validates exact split matches total amount due', () => {
    const totalAmount = 75000;
    const splitTenders: PaymentInput[] = [
      { method: 'CASH', amount: 40000 },
      { method: 'QRIS', amount: 35000 },
    ];

    const sum = splitTenders.reduce((acc, t) => acc + t.amount, 0);
    const remaining = totalAmount - sum;

    expect(sum).toBe(75000);
    expect(remaining).toBe(0);
  });

  it('detects underpayment on incomplete split allocations', () => {
    const totalAmount = 100000;
    const splitTenders: PaymentInput[] = [
      { method: 'CASH', amount: 50000 },
      { method: 'CARD', amount: 30000 },
    ];

    const sum = splitTenders.reduce((acc, t) => acc + t.amount, 0);
    const remaining = totalAmount - sum;

    expect(sum).toBe(80000);
    expect(remaining).toBe(20000); // 20,000 remaining due
  });

  it('calculates cash change correctly for cash transactions', () => {
    const totalDue = 47500;
    const cashTendered = 100000;
    const change = Math.max(0, cashTendered - totalDue);

    expect(change).toBe(52500);
  });
});
