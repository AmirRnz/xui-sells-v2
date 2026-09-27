import React, { useEffect, useState } from 'react';
import {
  Bot,
  Plus,
  RefreshCw,
  Users,
  CreditCard,
  Globe,
  ExternalLink,
  ShieldCheck,
  Check,
  Copy,
  X,
} from 'lucide-react';
import { api } from '../api/client';
import type { ChildBot, BotStatus } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { cn } from '../lib/utils';

export const ChildBots: React.FC = () => {
  const { t, lang } = useLanguage();
  const [bots, setBots] = useState<ChildBot[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  // New Bot Modal State
  const [showAddModal, setShowAddModal] = useState<boolean>(false);
  const [botToken, setBotToken] = useState<string>('');
  const [cardNumber, setCardNumber] = useState<string>('');
  const [cardholderName, setCardholderName] = useState<string>('');
  const [defaultLang, setDefaultLang] = useState<'fa' | 'en'>('fa');
  const [saving, setSaving] = useState<boolean>(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const fetchBots = async () => {
    setLoading(true);
    try {
      const data = await api.getChildBots();
      setBots(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBots();
  }, []);

  const handleCopyLink = (username: string, id: string) => {
    navigator.clipboard.writeText(`https://t.me/${username}`);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleDeployBot = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!botToken.trim()) return;

    setSaving(true);
    try {
      // Simulate Bot Father validation & registration
      const botUsername = `Customer_${botToken.split(':')[0] || 'Bot'}`;
      const newBot: ChildBot = {
        id: `bot-${Date.now().toString(36)}`,
        bot_username: botUsername,
        bot_token_masked: `${botToken.substring(0, 8)}:AAH***********************xyz`,
        default_lang: defaultLang,
        active_users_count: 0,
        card_number: cardNumber || '6037-xxxx-xxxx-0000',
        cardholder_name: cardholderName || 'Reseller Admin',
        status: 'active',
        created_at: new Date().toISOString(),
      };

      setBots((prev) => [newBot, ...prev]);
      setShowAddModal(false);
      setBotToken('');
      setCardNumber('');
      setCardholderName('');
      setNotice(
        lang === 'fa'
          ? `ربات جدید @${botUsername} با موفقیت راه‌اندازی و به گروه سرور شما متصل شد.`
          : `New child bot @${botUsername} deployed and linked successfully.`
      );
    } catch {
      alert(t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  const toggleBotStatus = (botId: string) => {
    setBots((prev) =>
      prev.map((b) => {
        if (b.id === botId) {
          const newStatus: BotStatus = b.status === 'active' ? 'stopped' : 'active';
          return { ...b, status: newStatus };
        }
        return b;
      })
    );
  };

  const totalUsers = bots.reduce((acc, b) => acc + b.active_users_count, 0);

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            <Bot className="w-6 h-6 text-indigo-400" />
            <span>{t('childBots.title')}</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">{t('childBots.subtitle')}</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={fetchBots}
            disabled={loading}
            className="flex items-center gap-2 px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white text-sm transition"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
          </button>
          <button
            type="button"
            onClick={() => setShowAddModal(true)}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/25 transition"
          >
            <Plus className="w-4 h-4" />
            <span>{t('childBots.newBot')}</span>
          </button>
        </div>
      </div>

      {notice && (
        <div className="p-4 rounded-xl bg-indigo-500/10 border border-indigo-500/30 text-indigo-300 text-sm flex items-center justify-between">
          <span>{notice}</span>
          <button type="button" onClick={() => setNotice(null)} className="text-slate-400 hover:text-white">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Overview Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5">
          <span className="text-xs text-slate-400">{lang === 'fa' ? 'تعداد ربات‌های ثبت شده' : 'Registered Bots'}</span>
          <div className="mt-2 text-2xl font-bold text-white tracking-tight">{bots.length}</div>
        </div>
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5">
          <span className="text-xs text-slate-400">{lang === 'fa' ? 'مجموع کاربران فعال' : 'Total Active End-Users'}</span>
          <div className="mt-2 text-2xl font-bold text-purple-400 tracking-tight">{totalUsers}</div>
        </div>
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5">
          <span className="text-xs text-slate-400">{lang === 'fa' ? 'میانگین کاربر در ربات' : 'Avg Users / Bot'}</span>
          <div className="mt-2 text-2xl font-bold text-emerald-400 tracking-tight">
            {bots.length > 0 ? Math.round(totalUsers / bots.length) : 0}
          </div>
        </div>
      </div>

      {/* Child Bots Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        {bots.map((bot) => (
          <div
            key={bot.id}
            className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5 shadow-sm hover:border-slate-700 transition space-y-4"
          >
            {/* Bot Header */}
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
                  <Bot className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="font-bold text-white text-base flex items-center gap-1.5">
                    <span>@{bot.bot_username}</span>
                  </h3>
                  <p className="text-xs font-mono text-slate-400 mt-0.5">{bot.bot_token_masked}</p>
                </div>
              </div>

              {/* Status Badge */}
              <button
                type="button"
                onClick={() => toggleBotStatus(bot.id)}
                className={cn(
                  'px-3 py-1 rounded-full text-xs font-semibold border transition cursor-pointer',
                  bot.status === 'active'
                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/20'
                    : 'bg-rose-500/10 text-rose-400 border-rose-500/30 hover:bg-rose-500/20'
                )}
                title="Click to toggle status"
              >
                {bot.status === 'active' ? t('status.active') : t('status.disabled')}
              </button>
            </div>

            {/* Metrics & Details */}
            <div className="grid grid-cols-2 gap-3 text-xs bg-slate-950/60 p-3.5 rounded-xl border border-slate-800/80">
              <div className="flex items-center gap-2">
                <Users className="w-4 h-4 text-purple-400 shrink-0" />
                <div>
                  <span className="text-slate-400 block">{t('childBots.users')}</span>
                  <span className="font-semibold text-slate-200">{bot.active_users_count}</span>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Globe className="w-4 h-4 text-sky-400 shrink-0" />
                <div>
                  <span className="text-slate-400 block">{t('childBots.lang')}</span>
                  <span className="font-semibold text-slate-200 uppercase">{bot.default_lang}</span>
                </div>
              </div>
              <div className="col-span-2 pt-2 border-t border-slate-800/60 flex items-center gap-2">
                <CreditCard className="w-4 h-4 text-emerald-400 shrink-0" />
                <div className="flex-1 truncate">
                  <span className="text-slate-400 block">{t('childBots.cardInfo')}</span>
                  <span className="font-mono text-slate-300 text-[11px]">
                    {bot.card_number} ({bot.cardholder_name})
                  </span>
                </div>
              </div>
            </div>

            {/* Card Actions */}
            <div className="pt-2 flex items-center justify-between text-xs">
              <span className="text-slate-500">ID: {bot.id}</span>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => handleCopyLink(bot.bot_username, bot.id)}
                  className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs flex items-center gap-1.5 transition"
                >
                  {copiedId === bot.id ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copiedId === bot.id ? (lang === 'fa' ? 'کپی شد' : 'Copied') : (lang === 'fa' ? 'کپی لینک ربات' : 'Copy Link')}</span>
                </button>
                <a
                  href={`https://t.me/${bot.bot_username}`}
                  target="_blank"
                  rel="noreferrer"
                  className="p-1.5 rounded-lg bg-slate-800 hover:bg-indigo-600 hover:text-white text-slate-400 transition"
                  title="Open Telegram Bot"
                >
                  <ExternalLink className="w-4 h-4" />
                </a>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Add New Bot Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
          <form
            onSubmit={handleDeployBot}
            className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg p-6 shadow-2xl space-y-4"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="w-9 h-9 rounded-xl bg-indigo-500/20 text-indigo-400 flex items-center justify-center">
                  <Bot className="w-5 h-5" />
                </div>
                <h3 className="text-lg font-bold text-white">{t('childBots.newBot')}</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-3 bg-indigo-500/10 border border-indigo-500/20 rounded-xl text-xs text-indigo-300 flex items-start gap-2">
              <ShieldCheck className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{t('childBots.addNotice')}</span>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('childBots.token')} *
              </label>
              <input
                type="text"
                required
                value={botToken}
                onChange={(e) => setBotToken(e.target.value)}
                placeholder="123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm font-mono text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">
                  {lang === 'fa' ? 'شماره کارت بانکی واریز' : 'Payment Card Number'}
                </label>
                <input
                  type="text"
                  value={cardNumber}
                  onChange={(e) => setCardNumber(e.target.value)}
                  placeholder="6037-9975-xxxx-xxxx"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm font-mono text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1">
                  {lang === 'fa' ? 'نام دارنده کارت' : 'Cardholder Name'}
                </label>
                <input
                  type="text"
                  value={cardholderName}
                  onChange={(e) => setCardholderName(e.target.value)}
                  placeholder="e.g. Alex Reseller"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('childBots.lang')}
              </label>
              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => setDefaultLang('fa')}
                  className={cn(
                    'p-2.5 rounded-xl border text-xs font-semibold transition',
                    defaultLang === 'fa'
                      ? 'bg-indigo-600/20 border-indigo-500 text-indigo-300'
                      : 'bg-slate-950 border-slate-800 text-slate-400 hover:text-white'
                  )}
                >
                  فارسی (Persian)
                </button>
                <button
                  type="button"
                  onClick={() => setDefaultLang('en')}
                  className={cn(
                    'p-2.5 rounded-xl border text-xs font-semibold transition',
                    defaultLang === 'en'
                      ? 'bg-indigo-600/20 border-indigo-500 text-indigo-300'
                      : 'bg-slate-950 border-slate-800 text-slate-400 hover:text-white'
                  )}
                >
                  English
                </button>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                disabled={saving}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium transition"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={saving}
                className="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-md shadow-indigo-600/25 transition flex items-center gap-1.5"
              >
                {saving && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                <span>{lang === 'fa' ? 'راه‌اندازی و اتصال ربات' : 'Deploy Child Bot'}</span>
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
