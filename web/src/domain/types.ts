export type ItemSector = 'FB' | 'RETAIL' | 'TOURISM' | 'HOSPITALITY';

export interface ItemModifierOption {
  name: string;
  price_delta: number;
}

export interface ModifierGroup {
  name: string;
  options: ItemModifierOption[];
}

export interface Item {
  id: string;
  tenant_id: string;
  outlet_id: string;
  sku: string;
  barcode?: string;
  name: string;
  sector: ItemSector;
  base_price: number;
  currency: string;
  tax_rate: number;
  sector_traits: Record<string, any>;
  version: number;
  is_active: boolean;
}

export interface InventoryBalance {
  tenant_id: string;
  outlet_id: string;
  sku: string;
  quantity: number;
  uom: string;
  last_updated: string;
}

export interface AdjustStockPayload {
  tenant_id?: string;
  outlet_id?: string;
  sku: string;
  delta_qty: number;
  reason: string;
  ref_id?: string;
}

export interface CreateItemPayload {
  tenant_id?: string;
  outlet_id?: string;
  sku: string;
  barcode?: string;
  name: string;
  sector: ItemSector;
  base_price: number;
  currency?: string;
  tax_rate?: number;
  sector_traits?: Record<string, any>;
  is_active?: boolean;
}

export interface UpdateItemPayload {
  sku?: string;
  barcode?: string;
  name?: string;
  sector?: ItemSector;
  base_price?: number;
  currency?: string;
  tax_rate?: number;
  sector_traits?: Record<string, any>;
  is_active?: boolean;
}

export interface CartItem {
  item: Item;
  quantity: number;
  selectedModifiers: ItemModifierOption[];
  sectorPayload?: Record<string, any>;
  unitPrice: number;
  subtotal: number;
  taxAmount: number;
  totalAmount: number;
}

export type PaymentMethod = 'CASH' | 'CARD' | 'QRIS' | 'ROOM_CHARGE';

export interface PaymentInput {
  method: PaymentMethod;
  amount: number;
  reference?: string;
  room_number?: string;
}

export interface CheckoutRequest {
  order_id?: string;
  tenant_id: string;
  outlet_id: string;
  notes?: string;
  metadata?: Record<string, any>;
  items: {
    sku: string;
    barcode?: string;
    quantity: number;
    uom?: string;
    selected_modifiers?: ItemModifierOption[];
    sector_payload?: Record<string, any>;
  }[];
  payment?: PaymentInput;
  payments?: PaymentInput[];
}

export type OrderStatus = 'PENDING' | 'PAID' | 'HELD' | 'PREPARING' | 'READY' | 'COMPLETED' | 'CANCELLED' | 'REFUNDED';

export interface OrderItem {
  id: string;
  order_id: string;
  item_id: string;
  sku: string;
  name: string;
  sector: ItemSector;
  quantity: number;
  uom: string;
  unit_price: number;
  subtotal: number;
  tax_amount: number;
  total_amount: number;
  selected_modifiers?: ItemModifierOption[];
}

export interface PaymentRecord {
  id: string;
  order_id: string;
  method: PaymentMethod;
  amount: number;
  reference?: string;
  paid_at: string;
}

export interface Order {
  id: string;
  tenant_id: string;
  outlet_id: string;
  order_number: string;
  status: OrderStatus;
  subtotal: number;
  tax_amount: number;
  discount_amount: number;
  total_amount: number;
  currency: string;
  notes?: string;
  items: OrderItem[];
  payments?: PaymentRecord[];
  created_at: string;
  updated_at: string;
}

export type ShiftStatus = 'OPEN' | 'CLOSED';
export type CashMovementType = 'CASH_IN' | 'CASH_OUT' | 'FLOAT_ADJUST';

export interface CashMovement {
  id: string;
  shift_id: string;
  type: CashMovementType;
  amount: number;
  reason: string;
  created_at: string;
}

export interface ShiftSession {
  id: string;
  tenant_id: string;
  outlet_id: string;
  cashier_id: string;
  cashier_name: string;
  status: ShiftStatus;
  opening_float: number;
  expected_cash: number;
  actual_cash: number;
  variance: number;
  notes?: string;
  opened_at: string;
  closed_at?: string;
}

export interface ShiftSummary extends ShiftSession {
  shift_id: string;
  cash_in_total: number;
  cash_out_total: number;
  cash_sales_total: number;
  non_cash_sales_total: number;
  total_sales: number;
  movements?: CashMovement[];
}

export interface GuestFolio {
  id: string;
  tenant_id: string;
  outlet_id: string;
  room_number: string;
  guest_name: string;
  credit_limit: number;
  current_total: number;
  is_open: boolean;
  check_in_at: string;
  check_out_at: string;
}

export interface TicketPayload {
  tid: string;
  ten: string;
  cat: string;
  pax: number;
  exp: number;
  iat: number;
  oid?: string;
}

export interface SignedTicketQR {
  payload: TicketPayload;
  signature: string;
}

export interface IssuedTicketResponse {
  ticket: SignedTicketQR;
  qr_string: string;
}
