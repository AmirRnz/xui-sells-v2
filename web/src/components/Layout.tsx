import React, { useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Server,
  CreditCard,
  Bot,
  MessageSquare,
  Settings,
  LogOut,
  Menu,
  X,
  Wallet,
  Globe,
  Sparkles,
  ShieldCheck,
  User,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { formatCurrency, cn } from '../lib/utils';

export const Layout: React.FC = () => {
  const { user, logout } = useAuth();
  const { lang, setLang, t, isRTL } = useLanguage();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const toggleLanguage = () => {
    setLang(lang === 'fa' ? 'en' : 'fa');
  };

  const navItems = [
    { to: '/', label: t('nav.dashboard'), icon: LayoutDashboard },
    { to: '/services', label: t('nav.services'), icon: Server },
    { to: '/orders', label: t('nav.orders'), icon: CreditCard },
    { to: '/child-bots', label: t('nav.childBots'), icon: Bot },
    { to: '/tickets', label: t('nav.tickets'), icon: MessageSquare },
    { to: '/settings', label: t('nav.settings'), icon: Settings },
  ];

  const getTierBadgeClass = (tier?: string) => {
    const tLower = tier?.toLowerCase();
    if (tLower === 'ultimate') {
      return 'bg-gradient-to-r from-amber-500/20 to-purple-500/20 text-amber-300 border-amber-500/30';
    }
    if (tLower === 'pro') {
      return 'bg-blue-500/20 text-blue-300 border-blue-500/30';
    }
    return 'bg-slate-700/50 text-slate-300 border-slate-600/50';
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col antialiased">
      {/* Top Navbar */}
      <header className="sticky top-0 z-40 h-16 border-b border-slate-800/80 bg-slate-900/90 backdrop-blur-md px-4 sm:px-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition lg:hidden"
            aria-label="Toggle Navigation"
          >
            {sidebarOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>

          <div className="flex items-center gap-2">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/20">
              <ShieldCheck className="w-5 h-5 text-white" />
            </div>
            <div>
              <span className="font-bold tracking-tight text-white flex items-center gap-1.5 text-base sm:text-lg">
                {user?.service_name || 'XUI Reseller'}
              </span>
              <span className="text-[10px] text-slate-400 hidden sm:block">
                Powered by XUI-Sells
              </span>
            </div>
          </div>
        </div>

        {/* Right / End items in Navbar */}
        <div className="flex items-center gap-2 sm:gap-4">
          {/* Tier Badge */}
          {user && (
            <div
              className={cn(
                'hidden sm:flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold border shadow-sm',
                getTierBadgeClass(user.tier)
              )}
            >
              <Sparkles className="w-3.5 h-3.5" />
              <span>{user.tier.toUpperCase()}</span>
            </div>
          )}

          {/* Wallet Balance */}
          {user && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/80 border border-slate-700/60 text-emerald-400 text-xs sm:text-sm font-medium">
              <Wallet className="w-4 h-4 text-emerald-400 shrink-0" />
              <div className="flex flex-col">
                <span className="text-[10px] text-slate-400 sm:hidden">{t('header.wallet')}</span>
                <span className="font-semibold tabular-nums">
                  {formatCurrency(user.wallet_balance, 'IRT', lang)}
                </span>
              </div>
            </div>
          )}

          {/* Language Toggle */}
          <button
            type="button"
            onClick={toggleLanguage}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700/80 text-slate-200 hover:text-white transition text-xs font-medium border border-slate-700/60"
            title="Switch Language"
          >
            <Globe className="w-3.5 h-3.5 text-indigo-400" />
            <span>{t('header.switchLang')}</span>
          </button>

          {/* User profile & Logout */}
          <div className="flex items-center gap-2 border-slate-800 ps-2 sm:ps-3">
            <div className="hidden md:flex items-center gap-2">
              <div className="w-8 h-8 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-slate-300">
                <User className="w-4 h-4" />
              </div>
              <div className="flex flex-col text-xs text-start">
                <span className="font-medium text-slate-200">{user?.first_name || user?.username}</span>
                <span className="text-slate-400 text-[10px]">ID: {user?.tg_id}</span>
              </div>
            </div>

            <button
              type="button"
              onClick={handleLogout}
              className="p-2 rounded-lg text-rose-400 hover:bg-rose-500/10 hover:text-rose-300 transition"
              title={t('nav.logout')}
            >
              <LogOut className={cn("w-4 h-4", isRTL ? "rotate-180" : "")} />
            </button>
          </div>
        </div>
      </header>

      {/* Main Container with Sidebar + Content */}
      <div className="flex flex-1 relative overflow-hidden">
        {/* Mobile Backdrop */}
        {sidebarOpen && (
          <div
            className="fixed inset-0 z-30 bg-slate-950/80 backdrop-blur-xs lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        {/* Sidebar */}
        <aside
          className={cn(
            'fixed inset-y-16 z-30 w-64 bg-slate-900/95 border-slate-800/80 p-4 transition-transform duration-200 ease-in-out lg:static lg:translate-x-0 flex flex-col justify-between shrink-0 backdrop-blur-sm',
            isRTL ? 'border-s right-0' : 'border-r left-0',
            sidebarOpen ? 'translate-x-0' : (isRTL ? 'translate-x-full lg:translate-x-0' : '-translate-x-full lg:translate-x-0')
          )}
        >
          {/* Navigation Links */}
          <nav className="space-y-1.5">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === '/'}
                  onClick={() => setSidebarOpen(false)}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all group',
                      isActive
                        ? 'bg-gradient-to-r from-indigo-600/30 to-purple-600/20 text-indigo-300 border border-indigo-500/30 shadow-sm'
                        : 'text-slate-400 hover:text-slate-100 hover:bg-slate-800/60'
                    )
                  }
                >
                  <Icon className="w-4 h-4 shrink-0 transition-colors group-hover:text-indigo-400" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>

          {/* Sidebar Footer Info */}
          <div className="pt-4 border-t border-slate-800/80 space-y-3">
            <div className="p-3 rounded-xl bg-slate-800/40 border border-slate-800 text-xs space-y-1.5">
              <div className="flex justify-between items-center text-slate-400">
                <span>{t('header.service')}</span>
                <span className="font-semibold text-slate-200">{user?.service_name}</span>
              </div>
              <div className="flex justify-between items-center text-slate-400">
                <span>{t('header.tier')}</span>
                <span className="text-amber-400 font-semibold">{user?.tier}</span>
              </div>
              <div className="flex justify-between items-center text-slate-400">
                <span>TG ID</span>
                <span className="text-slate-300 font-mono">{user?.tg_id}</span>
              </div>
            </div>
            <div className="text-[11px] text-center text-slate-400">
              v2.0 &bull; Reseller Panel
            </div>
          </div>
        </aside>

        {/* Page Content Viewport */}
        <main className="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8 bg-slate-950/50">
          <div className="max-w-7xl mx-auto space-y-6">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
};
