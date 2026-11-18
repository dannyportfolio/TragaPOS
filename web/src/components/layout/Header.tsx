import React, { useState } from 'react';
import {
  Store,
  ShieldCheck,
  UserCheck,
  PauseCircle,
  KeyRound,
  ChefHat,
  ScanLine,
  Package,
  ClipboardList,
  BedDouble,
  Ticket,
  Menu,
  X,
  ShoppingCart,
  RefreshCw,
  Lock,
  Sun,
  Moon,
} from 'lucide-react';
import { useShiftStore } from '../../store/useShiftStore';
import { useThemeStore } from '../../store/useThemeStore';
import { api, getAuthToken, setAuthToken } from '../../services/api';
import { toast } from '../../store/useToastStore';

export type AppView = 'POS' | 'ITEMS' | 'ORDERS' | 'KDS' | 'GATE' | 'FOLIOS' | 'TICKETS';

interface HeaderProps {
  activeView: AppView;
  onChangeView: (view: AppView) => void;
  onOpenShiftModal: () => void;
  onOpenHeldModal: () => void;
  onToggleMobileCart?: () => void;
  heldCount: number;
  cartCount?: number;
}

const NAV_ITEMS: { key: AppView; label: string; activeClass: string; icon: React.ReactNode }[] = [
  { key: 'POS', label: 'POS Register', activeClass: 'bg-indigo-600 text-white shadow-sm', icon: <Store className="w-4 h-4 shrink-0" /> },
  { key: 'ITEMS', label: 'Products', activeClass: 'bg-blue-600 text-white shadow-sm', icon: <Package className="w-4 h-4 shrink-0" /> },
  { key: 'ORDERS', label: 'Orders', activeClass: 'bg-cyan-700 text-white shadow-sm', icon: <ClipboardList className="w-4 h-4 shrink-0" /> },
  { key: 'KDS', label: 'Kitchen KDS', activeClass: 'bg-amber-600 text-white shadow-sm', icon: <ChefHat className="w-4 h-4 shrink-0" /> },
  { key: 'FOLIOS', label: 'Folios', activeClass: 'bg-teal-600 text-white shadow-sm', icon: <BedDouble className="w-4 h-4 shrink-0" /> },
  { key: 'TICKETS', label: 'Tickets', activeClass: 'bg-violet-600 text-white shadow-sm', icon: <Ticket className="w-4 h-4 shrink-0" /> },
  { key: 'GATE', label: 'Gate', activeClass: 'bg-purple-600 text-white shadow-sm', icon: <ScanLine className="w-4 h-4 shrink-0" /> },
];

export const Header: React.FC<HeaderProps> = ({
  activeView,
  onChangeView,
  onOpenShiftModal,
  onOpenHeldModal,
  onToggleMobileCart,
  heldCount,
  cartCount = 0,
}) => {
  const { shift } = useShiftStore();
  const { theme, toggleTheme } = useThemeStore();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
  const [tokenInput, setTokenInput] = useState(() => getAuthToken());

  const handleSyncNow = async () => {
    setIsSyncing(true);
    try {
      await api.triggerSync();
      toast.success('Background sync cycle triggered');
    } catch (err: any) {
      toast.info(err.message || 'Sync trigger unavailable on this node');
    } finally {
      setIsSyncing(false);
    }
  };

  const handleSaveToken = (e: React.FormEvent) => {
    e.preventDefault();
    setAuthToken(tokenInput);
    setIsTokenModalOpen(false);
    toast.success(tokenInput.trim() ? 'API bearer token saved' : 'API bearer token cleared');
  };

  const handleSelectView = (view: AppView) => {
    onChangeView(view);
    setMobileMenuOpen(false);
  };

  return (
    <>
      <header className="min-h-[64px] bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 px-3 sm:px-5 py-2 flex items-center justify-between gap-2 select-none shrink-0 transition-colors">
        <div className="flex items-center gap-2 sm:gap-3 min-w-0">
          {/* Mobile Nav Toggle */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen((v) => !v)}
            aria-label="Toggle navigation menu"
            aria-expanded={mobileMenuOpen}
            className="xl:hidden min-w-[44px] min-h-[44px] px-2.5 rounded-xl bg-slate-100 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-800 flex items-center justify-center gap-1.5 shrink-0 transition-colors"
          >
            {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            <span className="hidden xs:inline text-xs font-semibold">Menu</span>
          </button>

          {/* Brand */}
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-9 h-9 rounded-lg bg-indigo-50 dark:bg-indigo-600/20 text-indigo-700 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-500/30 flex items-center justify-center font-bold shrink-0">
              <Store className="w-5 h-5" />
            </div>
            <div className="hidden sm:block truncate">
              <div className="font-semibold text-slate-900 dark:text-slate-100 flex items-center gap-2 text-sm leading-tight">
                Commerce Engine
                <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/30">
                  EDGE
                </span>
              </div>
              <div className="text-[11px] text-slate-600 dark:text-slate-400 flex items-center gap-1 mt-0.5">
                <ShieldCheck className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                Offline-First Hub
              </div>
            </div>
          </div>

          {/* Desktop View Switcher Tabs */}
          <nav aria-label="Main navigation" className="hidden xl:flex bg-slate-100 dark:bg-slate-950 p-1 rounded-xl border border-slate-200 dark:border-slate-800 gap-0.5">
            {NAV_ITEMS.map((item) => (
              <button
                key={item.key}
                type="button"
                onClick={() => handleSelectView(item.key)}
                className={`px-2.5 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all whitespace-nowrap ${
                  activeView === item.key
                    ? item.activeClass
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-white/70 dark:hover:bg-slate-900/60'
                }`}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            ))}
          </nav>
        </div>

        {/* Right Actions */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
          {/* Theme Toggle */}
          <button
            type="button"
            onClick={toggleTheme}
            title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            className="min-w-[44px] min-h-[44px] sm:min-w-0 sm:px-2.5 py-1.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-300 dark:border-slate-700 flex items-center justify-center gap-1.5 text-xs font-medium transition-colors"
          >
            {theme === 'dark' ? (
              <Sun className="w-4 h-4 text-amber-400" />
            ) : (
              <Moon className="w-4 h-4 text-indigo-600" />
            )}
            <span className="hidden 2xl:inline">{theme === 'dark' ? 'Light' : 'Dark'}</span>
          </button>

          {/* Sync Button */}
          <button
            type="button"
            onClick={handleSyncNow}
            disabled={isSyncing}
            title="Trigger Edge-to-Cloud Sync"
            aria-label="Sync now"
            className="min-w-[44px] min-h-[44px] sm:min-w-0 sm:px-3 py-1.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 flex items-center justify-center gap-1.5 text-xs font-medium transition-colors"
          >
            <RefreshCw className={`w-4 h-4 text-sky-600 dark:text-sky-400 ${isSyncing ? 'animate-spin' : ''}`} />
            <span className="hidden md:inline">Sync</span>
          </button>

          {/* API Token Settings */}
          <button
            type="button"
            onClick={() => setIsTokenModalOpen(true)}
            title="Configure API Auth Token"
            aria-label="API Auth Token"
            className={`min-w-[44px] min-h-[44px] sm:min-w-0 sm:px-2.5 py-1.5 rounded-xl border flex items-center justify-center text-xs transition-colors ${
              getAuthToken()
                ? 'bg-emerald-50 dark:bg-emerald-500/10 border-emerald-300 dark:border-emerald-500/30 text-emerald-700 dark:text-emerald-400'
                : 'bg-slate-100 dark:bg-slate-800 border-slate-300 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
            }`}
          >
            <Lock className="w-4 h-4" />
          </button>

          {/* Held Orders Button (Visible on POS view) */}
          {activeView === 'POS' && (
            <button
              type="button"
              onClick={onOpenHeldModal}
              aria-label={`Parked orders (${heldCount})`}
              className="relative min-w-[44px] min-h-[44px] px-2.5 sm:px-3 py-1.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700 flex items-center justify-center gap-2 text-xs font-medium transition-colors"
            >
              <PauseCircle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0" />
              <span className="hidden sm:inline">Parked</span>
              {heldCount > 0 && (
                <span className="w-5 h-5 rounded-full bg-amber-500 text-slate-950 font-bold text-[11px] flex items-center justify-center">
                  {heldCount}
                </span>
              )}
            </button>
          )}

          {/* Mobile Cart Drawer Toggle (Visible on < lg in POS view) */}
          {activeView === 'POS' && onToggleMobileCart && (
            <button
              type="button"
              onClick={onToggleMobileCart}
              aria-label={`Open cart (${cartCount} items)`}
              className="lg:hidden relative min-w-[44px] min-h-[44px] px-3 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-700 dark:hover:bg-indigo-500 text-white flex items-center justify-center gap-1.5 text-xs font-semibold transition-colors"
            >
              <ShoppingCart className="w-4 h-4" />
              {cartCount > 0 && (
                <span className="px-1.5 py-0.5 rounded-full bg-white text-indigo-700 font-bold text-[10px]">
                  {cartCount}
                </span>
              )}
            </button>
          )}

          {/* Shift Management Button */}
          <button
            type="button"
            onClick={onOpenShiftModal}
            className={`min-h-[44px] min-w-[44px] px-2.5 sm:px-3 py-1.5 rounded-xl border flex items-center justify-center gap-2 text-xs font-medium transition-all ${
              shift
                ? 'bg-emerald-50 dark:bg-slate-800 border-emerald-300 dark:border-emerald-500/40 text-emerald-800 dark:text-emerald-300 hover:bg-emerald-100 dark:hover:bg-slate-700'
                : 'bg-rose-50 dark:bg-rose-500/10 border-rose-300 dark:border-rose-500/30 text-rose-700 dark:text-rose-300 hover:bg-rose-100 dark:hover:bg-rose-500/20'
            }`}
          >
            {shift ? (
              <>
                <UserCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
                <span className="hidden md:inline truncate max-w-[110px]">Shift ({shift.cashier_name})</span>
                <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" aria-hidden="true"></span>
              </>
            ) : (
              <>
                <KeyRound className="w-4 h-4 text-rose-600 dark:text-rose-400 shrink-0" />
                <span className="hidden sm:inline">Open Shift</span>
              </>
            )}
          </button>
        </div>
      </header>

      {/* Mobile/Tablet Navigation Drawer */}
      {mobileMenuOpen && (
        <nav
          aria-label="Mobile navigation"
          className="xl:hidden bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 px-3 sm:px-4 py-3 grid grid-cols-2 sm:grid-cols-4 gap-2 shrink-0 shadow-sm"
        >
          {NAV_ITEMS.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => handleSelectView(item.key)}
              className={`min-h-[44px] px-3 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all ${
                activeView === item.key
                  ? item.activeClass
                  : 'bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-800'
              }`}
            >
              {item.icon}
              <span>{item.label}</span>
            </button>
          ))}
        </nav>
      )}

      {/* API Token Modal */}
      {isTokenModalOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="token-modal-title"
          className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4"
        >
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-2xl w-full max-w-sm p-5 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between">
              <h3 id="token-modal-title" className="font-semibold text-slate-900 dark:text-slate-100 text-sm flex items-center gap-2">
                <Lock className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
                API Bearer Token
              </h3>
              <button
                type="button"
                onClick={() => setIsTokenModalOpen(false)}
                aria-label="Close token modal"
                className="min-w-[40px] min-h-[40px] rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-500 dark:text-slate-400 flex items-center justify-center"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              If the backend enforces <code className="text-indigo-700 dark:text-indigo-400 font-mono">API_AUTH_TOKEN</code>, enter your bearer token here.
            </p>
            <form onSubmit={handleSaveToken} className="space-y-3">
              <input
                type="password"
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder="Enter bearer token (leave empty to clear)"
                className="w-full min-h-[44px] px-3.5 py-2 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-800 text-slate-900 dark:text-slate-100 text-xs font-mono focus:outline-none focus:border-indigo-500"
              />
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setIsTokenModalOpen(false)}
                  className="flex-1 min-h-[44px] rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 text-xs font-semibold transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="flex-1 min-h-[44px] rounded-xl bg-indigo-600 hover:bg-indigo-700 dark:hover:bg-indigo-500 text-white text-xs font-semibold transition-colors"
                >
                  Save Token
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
};
