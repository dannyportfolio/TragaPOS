import {
  Item,
  CheckoutRequest,
  Order,
  OrderStatus,
  ShiftSession,
  ShiftSummary,
  CashMovement,
  CashMovementType,
  GuestFolio,
  AdjustStockPayload,
  CreateItemPayload,
  UpdateItemPayload,
  InventoryBalance,
  IssuedTicketResponse,
} from '../domain/types';

const AUTH_STORAGE_KEY = 'pos_api_auth_token';

export function getAuthToken(): string {
  try {
    return localStorage.getItem(AUTH_STORAGE_KEY) || '';
  } catch {
    return '';
  }
}

export function setAuthToken(token: string): void {
  try {
    if (token.trim()) {
      localStorage.setItem(AUTH_STORAGE_KEY, token.trim());
    } else {
      localStorage.removeItem(AUTH_STORAGE_KEY);
    }
  } catch {
    // ignore storage errors
  }
}

function generateIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'idem-' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
}

const API_BASE = '/v1';

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const token = getAuthToken();
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  // Inject Idempotency Key for mutating requests if not already provided
  if ((options.method === 'POST' || options.method === 'PUT') && !headers.has('X-Idempotency-Key')) {
    headers.set('X-Idempotency-Key', generateIdempotencyKey());
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    let errMsg = `Request failed with status ${response.status}`;
    try {
      const errData = await response.json();
      if (errData && errData.error) {
        errMsg = errData.error;
      }
    } catch {
      // ignore json parse error
    }
    throw new Error(errMsg);
  }

  // Handle empty responses
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return response.json();
  }
  return response.text() as unknown as T;
}

export const api = {
  // Catalog CRUD
  async getCatalogItems(tenantId?: string, limit = 100): Promise<Item[]> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (tenantId) params.append('tenant_id', tenantId);
    const data = await request<{ items: Item[]; count: number }>(`/catalog/items?${params.toString()}`);
    return data.items || [];
  },

  async getItem(idOrSku: string, tenantId?: string): Promise<Item> {
    const params = tenantId ? `?tenant_id=${tenantId}` : '';
    return request<Item>(`/catalog/items/${encodeURIComponent(idOrSku)}${params}`);
  },

  async getItemByBarcode(barcode: string, tenantId?: string): Promise<Item> {
    const params = tenantId ? `?tenant_id=${tenantId}` : '';
    return request<Item>(`/catalog/barcode/${encodeURIComponent(barcode)}${params}`);
  },

  async createItem(payload: CreateItemPayload): Promise<Item> {
    return request<Item>('/catalog/items', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  async updateItem(id: string, payload: UpdateItemPayload): Promise<{ status: string; id: string }> {
    return request<{ status: string; id: string }>(`/catalog/items/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  },

  async deleteItem(id: string, tenantId?: string): Promise<{ status: string; id: string }> {
    const params = tenantId ? `?tenant_id=${tenantId}` : '';
    return request<{ status: string; id: string }>(`/catalog/items/${encodeURIComponent(id)}${params}`, {
      method: 'DELETE',
    });
  },

  async getStock(sku: string, tenantId?: string, outletId?: string): Promise<InventoryBalance> {
    const params = new URLSearchParams();
    if (tenantId) params.append('tenant_id', tenantId);
    if (outletId) params.append('outlet_id', outletId);
    const query = params.toString() ? `?${params.toString()}` : '';
    return request<InventoryBalance>(`/catalog/stock/${encodeURIComponent(sku)}${query}`);
  },

  async adjustStock(payload: AdjustStockPayload): Promise<InventoryBalance> {
    return request<InventoryBalance>('/catalog/stock/adjust', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  // Checkout & Orders CRUD
  async checkout(payload: CheckoutRequest): Promise<Order> {
    return request<Order>('/checkout', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  async holdOrder(payload: CheckoutRequest): Promise<Order> {
    return request<Order>('/orders/hold', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  async getOrders(params?: {
    status?: OrderStatus | string;
    tenant_id?: string;
    outlet_id?: string;
    limit?: number;
    offset?: number;
  }): Promise<Order[]> {
    const query = new URLSearchParams();
    if (params?.status) query.append('status', params.status);
    if (params?.tenant_id) query.append('tenant_id', params.tenant_id);
    if (params?.outlet_id) query.append('outlet_id', params.outlet_id);
    if (params?.limit) query.append('limit', String(params.limit));
    if (params?.offset) query.append('offset', String(params.offset));
    const res = await request<{ orders: Order[]; count: number }>(`/orders?${query.toString()}`);
    return res.orders || [];
  },

  async voidOrder(orderId: string, reason: string): Promise<{ status: string; order_id: string }> {
    return request<{ status: string; order_id: string }>(`/orders/${encodeURIComponent(orderId)}/void`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  },

  async getHeldOrders(tenantId?: string, outletId?: string): Promise<Order[]> {
    const params = new URLSearchParams();
    if (tenantId) params.append('tenant_id', tenantId);
    if (outletId) params.append('outlet_id', outletId);
    const res = await request<{ orders: Order[]; count: number }>(`/orders/held?${params.toString()}`);
    return res.orders || [];
  },

  async getOrder(id: string): Promise<Order> {
    return request<Order>(`/orders/${encodeURIComponent(id)}`);
  },

  async getReceipt(id: string): Promise<string> {
    return request<string>(`/orders/${encodeURIComponent(id)}/receipt?format=text`);
  },

  // Shift Management
  async getShifts(limit = 50, offset = 0, tenantId?: string, outletId?: string): Promise<ShiftSession[]> {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (tenantId) params.append('tenant_id', tenantId);
    if (outletId) params.append('outlet_id', outletId);
    const res = await request<{ shifts: ShiftSession[]; count: number }>(`/shifts?${params.toString()}`);
    return res.shifts || [];
  },

  async openShift(data: {
    cashier_id: string;
    cashier_name: string;
    opening_float: number;
    notes?: string;
    tenant_id?: string;
    outlet_id?: string;
  }): Promise<ShiftSession> {
    return request<ShiftSession>('/shifts/open', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async getCurrentShift(cashierId = 'default_cashier'): Promise<ShiftSummary | null> {
    try {
      return await request<ShiftSummary>(`/shifts/current?cashier_id=${encodeURIComponent(cashierId)}`);
    } catch (err: any) {
      if (err.message && err.message.includes('no active shift')) {
        return null;
      }
      throw err;
    }
  },

  async recordCashMovement(data: {
    type: CashMovementType;
    amount: number;
    reason: string;
    cashier_id?: string;
  }): Promise<CashMovement> {
    return request<CashMovement>('/shifts/movements', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async closeShift(data: {
    actual_cash: number;
    notes?: string;
    cashier_id?: string;
  }): Promise<ShiftSummary> {
    return request<ShiftSummary>('/shifts/close', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // Hospitality Folios
  async createFolio(data: {
    room_number: string;
    guest_name: string;
    credit_limit: number;
    tenant_id?: string;
    outlet_id?: string;
  }): Promise<GuestFolio> {
    return request<GuestFolio>('/folios', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async getFolios(openOnly = false, limit = 50, offset = 0, tenantId?: string): Promise<GuestFolio[]> {
    const params = new URLSearchParams({
      open_only: String(openOnly),
      limit: String(limit),
      offset: String(offset),
    });
    if (tenantId) params.append('tenant_id', tenantId);
    const res = await request<{ folios: GuestFolio[]; count: number }>(`/folios?${params.toString()}`);
    return res.folios || [];
  },

  async settleFolio(folioId: string, tenantId?: string): Promise<{ status: string; folio_id: string }> {
    const params = tenantId ? `?tenant_id=${tenantId}` : '';
    return request<{ status: string; folio_id: string }>(`/folios/${encodeURIComponent(folioId)}/settle${params}`, {
      method: 'POST',
    });
  },

  async getRoomFolio(roomNumber: string, tenantId?: string): Promise<GuestFolio> {
    const params = tenantId ? `?tenant_id=${tenantId}` : '';
    return request<GuestFolio>(`/folios/room/${encodeURIComponent(roomNumber)}${params}`);
  },

  // KDS & Order Fulfillment (Option 2)
  async getKdsOrders(tenantId?: string, outletId?: string): Promise<Order[]> {
    const params = new URLSearchParams();
    if (tenantId) params.append('tenant_id', tenantId);
    if (outletId) params.append('outlet_id', outletId);
    const res = await request<{ orders: Order[]; count: number }>(`/kds/orders?${params.toString()}`);
    return res.orders || [];
  },

  async updateOrderStatus(orderId: string, status: string): Promise<Order> {
    return request<Order>(`/orders/${encodeURIComponent(orderId)}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    });
  },

  // Gate Turnstile Verification & Issuance
  async issueTicket(data: {
    category: string;
    pax: number;
    valid_hours: number;
    order_id?: string;
    tenant_id?: string;
  }): Promise<IssuedTicketResponse> {
    return request<IssuedTicketResponse>('/tickets/issue', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async triggerSync(): Promise<{ status: string }> {
    return request<{ status: string }>('/sync/trigger', {
      method: 'POST',
    });
  },

  async redeemTicket(data: {
    qr_string: string;
    gate_zone?: string;
    device_id?: string;
    tenant_id?: string;
    outlet_id?: string;
  }): Promise<{ id: string; ticket_id: string; pax_redeemed: number; gate_zone: string }> {
    return request<{ id: string; ticket_id: string; pax_redeemed: number; gate_zone: string }>('/tickets/redeem', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },
};
