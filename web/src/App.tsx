import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { Header, AppView } from './components/layout/Header';
import { CatalogGrid } from './components/register/CatalogGrid';
import { CartDrawer } from './components/cart/CartDrawer';
import { ShiftModal } from './components/shift/ShiftModal';
import { HeldOrdersModal } from './components/cart/HeldOrdersModal';
import { ModifierModal } from './components/register/ModifierModal';
import { PaymentModal } from './components/payment/PaymentModal';
import { ReceiptModal } from './components/receipt/ReceiptModal';
import { KdsView } from './components/kds/KdsView';
import { GateScannerView } from './components/gate/GateScannerView';
import { ItemsView } from './components/catalog/ItemsView';
import { OrdersView } from './components/orders/OrdersView';
import { FoliosView } from './components/folios/FoliosView';
import { TicketsView } from './components/tickets/TicketsView';
import { ToastContainer } from './components/ui/ToastContainer';
import { useShiftStore } from './store/useShiftStore';
import { useCartStore } from './store/useCartStore';
import { toast } from './store/useToastStore';
import { Item, ItemModifierOption, Order } from './domain/types';
import { api } from './services/api';

export const App: React.FC = () => {
  const { shift, fetchShift } = useShiftStore();
  const { items: cartItems, addItem, notes, clearCart, heldOrderId } = useCartStore();

  const [activeView, setActiveView] = useState<AppView>('POS');
  const [catalogItems, setCatalogItems] = useState<Item[]>([]);
  const [isLoadingCatalog, setIsLoadingCatalog] = useState(false);
  const [isMobileCartOpen, setIsMobileCartOpen] = useState(false);

  // Modals state
  const [isShiftModalOpen, setIsShiftModalOpen] = useState(false);
  const [isHeldModalOpen, setIsHeldModalOpen] = useState(false);
  const [isPaymentModalOpen, setIsPaymentModalOpen] = useState(false);
  const [isReceiptModalOpen, setIsReceiptModalOpen] = useState(false);
  const [completedOrder, setCompletedOrder] = useState<Order | null>(null);

  // Modifier modal state
  const [modifierItem, setModifierItem] = useState<Item | null>(null);
  const [isModifierModalOpen, setIsModifierModalOpen] = useState(false);

  // Parked orders count
  const [heldCount, setHeldCount] = useState(0);
  const [isHolding, setIsHolding] = useState(false);

  const totalCartCount = cartItems.reduce((sum, ci) => sum + ci.quantity, 0);

  const loadCatalog = async () => {
    setIsLoadingCatalog(true);
    try {
      const items = await api.getCatalogItems();
      setCatalogItems(items);
    } catch (err: any) {
      toast.error(err.message || 'Failed to load catalog');
    } finally {
      setIsLoadingCatalog(false);
    }
  };

  const updateHeldCount = async () => {
    try {
      const orders = await api.getHeldOrders();
      setHeldCount(orders.length);
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    fetchShift();
    loadCatalog();
    updateHeldCount();
  }, []);

  // Global keyboard shortcuts: Escape closes modals/drawers
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsMobileCartOpen(false);
        setIsShiftModalOpen(false);
        setIsHeldModalOpen(false);
        setIsModifierModalOpen(false);
        setIsPaymentModalOpen(false);
        setIsReceiptModalOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleSelectItem = (item: Item) => {
    if (!shift) {
      toast.info('Please open a cashier shift before ringing up items');
      setIsShiftModalOpen(true);
      return;
    }

    if (item.sector === 'FB') {
      setModifierItem(item);
      setIsModifierModalOpen(true);
    } else {
      addItem(item);
      toast.success(`Added ${item.name} to cart`);
    }
  };

  const handleConfirmModifiers = (item: Item, modifiers: ItemModifierOption[]) => {
    addItem(item, modifiers);
    toast.success(`Added ${item.name} to cart`);
  };

  const handleBarcodeScan = async (barcode: string) => {
    if (!shift) {
      toast.info('Please open a cashier shift before scanning items');
      setIsShiftModalOpen(true);
      return;
    }
    try {
      const item = await api.getItemByBarcode(barcode);
      addItem(item);
      toast.success(`Scanned ${item.name}`);
    } catch (err: any) {
      toast.error(`Barcode "${barcode}" not found: ${err.message}`);
    }
  };

  const handleHoldOrder = async () => {
    if (cartItems.length === 0) return;
    setIsHolding(true);
    try {
      await api.holdOrder({
        order_id: heldOrderId,
        tenant_id: '',
        outlet_id: '',
        notes: notes || 'Parked by cashier',
        items: cartItems.map((ci) => ({
          sku: ci.item.sku,
          barcode: ci.item.barcode,
          quantity: ci.quantity,
          selected_modifiers: ci.selectedModifiers,
          sector_payload: ci.sectorPayload,
        })),
      });
      clearCart();
      setIsMobileCartOpen(false);
      await updateHeldCount();
      toast.success('Order parked for later');
    } catch (err: any) {
      toast.error(`Failed to park order: ${err.message}`);
    } finally {
      setIsHolding(false);
    }
  };

  const handlePaymentSuccess = (order: Order) => {
    setCompletedOrder(order);
    setIsReceiptModalOpen(true);
    setIsMobileCartOpen(false);
    updateHeldCount();
    fetchShift();
    toast.success(`Order ${order.order_number} completed`);
  };

  const openPayment = () => {
    if (!shift) {
      toast.info('Open a shift before taking payment');
      setIsShiftModalOpen(true);
      return;
    }
    setIsPaymentModalOpen(true);
  };

  return (
    <div className="flex flex-col h-dvh w-full max-w-full overflow-hidden bg-slate-950 font-sans">
      {/* Top Header */}
      <Header
        activeView={activeView}
        onChangeView={setActiveView}
        onOpenShiftModal={() => setIsShiftModalOpen(true)}
        onOpenHeldModal={() => setIsHeldModalOpen(true)}
        onToggleMobileCart={() => setIsMobileCartOpen((v) => !v)}
        heldCount={heldCount}
        cartCount={totalCartCount}
      />

      {/* Main Body View */}
      <main className="flex-1 flex min-h-0 min-w-0 overflow-hidden relative">
        {activeView === 'POS' ? (
          <>
            <CatalogGrid
              items={catalogItems}
              onSelectItem={handleSelectItem}
              onBarcodeScan={handleBarcodeScan}
              isLoading={isLoadingCatalog}
            />

            {/* Desktop Persistent Cart */}
            <div className="hidden lg:flex h-full shrink-0">
              <CartDrawer
                onOpenPaymentModal={openPayment}
                onHoldOrder={handleHoldOrder}
                isHolding={isHolding}
              />
            </div>

            {/* Mobile/Tablet Slide-Over Cart */}
            {isMobileCartOpen && (
              <div className="lg:hidden fixed inset-0 z-40 bg-slate-950/75 backdrop-blur-sm flex justify-end">
                <div className="relative h-full w-full max-w-sm bg-slate-900 flex flex-col shadow-2xl">
                  <button
                    onClick={() => setIsMobileCartOpen(false)}
                    aria-label="Close cart"
                    className="absolute top-3 right-3 z-10 min-w-[36px] min-h-[36px] rounded-lg bg-slate-800 text-slate-300 flex items-center justify-center"
                  >
                    <X className="w-4 h-4" />
                  </button>
                  <CartDrawer
                    onOpenPaymentModal={openPayment}
                    onHoldOrder={handleHoldOrder}
                    isHolding={isHolding}
                  />
                </div>
              </div>
            )}
          </>
        ) : activeView === 'ITEMS' ? (
          <ItemsView />
        ) : activeView === 'ORDERS' ? (
          <OrdersView />
        ) : activeView === 'KDS' ? (
          <KdsView />
        ) : activeView === 'FOLIOS' ? (
          <FoliosView />
        ) : activeView === 'TICKETS' ? (
          <TicketsView />
        ) : (
          <GateScannerView />
        )}
      </main>

      {/* Modals */}
      <ShiftModal
        isOpen={isShiftModalOpen}
        onClose={() => setIsShiftModalOpen(false)}
      />

      <HeldOrdersModal
        isOpen={isHeldModalOpen}
        onClose={() => setIsHeldModalOpen(false)}
        onHeldCountChange={setHeldCount}
      />

      <ModifierModal
        item={modifierItem}
        isOpen={isModifierModalOpen}
        onClose={() => {
          setIsModifierModalOpen(false);
          setModifierItem(null);
        }}
        onConfirm={handleConfirmModifiers}
      />

      <PaymentModal
        isOpen={isPaymentModalOpen}
        onClose={() => setIsPaymentModalOpen(false)}
        onSuccess={handlePaymentSuccess}
      />

      <ReceiptModal
        order={completedOrder}
        isOpen={isReceiptModalOpen}
        onClose={() => {
          setIsReceiptModalOpen(false);
          setCompletedOrder(null);
        }}
      />

      <ToastContainer />
    </div>
  );
};
export default App;
