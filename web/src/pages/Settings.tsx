import React, { useState } from 'react';
import {
  Settings as SettingsIcon,
  Shield,
  KeyRound,
  Sparkles,
  Globe,
  User,
  CheckCircle2,
  AlertCircle,
  RefreshCw,
} from 'lucide-react';
import { api } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage, type Language } from '../context/LanguageContext';
import { formatCurrency, formatDate, cn } from '../lib/utils';

export const Settings: React.FC = () => {
  const { user } = useAuth();
  const { t, lang, setLang } = useLanguage();

  // Password state
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordSuccess(null);
    setPasswordError(null);

    if (newPassword.length < 6) {
      setPasswordError(
        lang === 'fa'
          ? 'کلمه عبور جدید باید حداقل ۶ کاراکتر باشد'
          : 'New password must be at least 6 characters'
      );
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordError(
        lang === 'fa'
          ? 'تکرار کلمه عبور با کلمه عبور جدید یکسان نیست'
          : 'Passwords do not match'
      );
      return;
    }

    setPasswordLoading(true);
    try {
      const res = await api.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      });
      setPasswordSuccess(
        lang === 'fa'
          ? 'کلمه عبور شما با موفقیت تغییر یافت.'
          : res.message || 'Password changed successfully.'
      );
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: unknown) {
      setPasswordError(
        err instanceof Error
          ? err.message
          : lang === 'fa'
          ? 'خطا در تغییر کلمه عبور'
          : 'Failed to change password'
      );
    } finally {
      setPasswordLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Title */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
          <SettingsIcon className="w-6 h-6 text-indigo-400" />
          <span>{t('settings.title')}</span>
        </h1>
        <p className="text-slate-400 text-sm mt-1">{t('settings.subtitle')}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Profile Card */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 flex items-center justify-center">
              <User className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">{t('settings.profile')}</h2>
              <span className="text-xs text-slate-400">Telegram Authentication Context</span>
            </div>
          </div>

          <div className="space-y-3 text-sm">
            <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
              <span className="text-slate-400">{t('settings.resellerId')}</span>
              <span className="font-mono text-indigo-300 font-semibold">{user?.tg_id}</span>
            </div>
            <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
              <span className="text-slate-400">{t('settings.username')}</span>
              <span className="font-medium text-slate-200">@{user?.username}</span>
            </div>
            <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
              <span className="text-slate-400">{t('settings.brandName')}</span>
              <span className="font-bold text-white">{user?.service_name}</span>
            </div>
            <div className="flex items-center justify-between p-3 rounded-xl bg-slate-950 border border-slate-800">
              <span className="text-slate-400">{t('header.wallet')}</span>
              <span className="font-bold text-emerald-400 tabular-nums">
                {formatCurrency(user?.wallet_balance ?? 0, 'IRT', lang)}
              </span>
            </div>
          </div>
        </div>

        {/* Tier Details Card */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-amber-500/10 text-amber-400 border border-amber-500/20 flex items-center justify-center">
              <Sparkles className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">{t('settings.membership')}</h2>
              <span className="text-xs text-slate-400">Membership Quotas & Permissions</span>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-gradient-to-r from-amber-500/10 via-purple-500/10 to-indigo-500/10 border border-amber-500/20 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs text-slate-300 font-medium">Active Membership</span>
              <span className="px-3 py-1 rounded-full text-xs font-bold bg-amber-500/20 text-amber-300 border border-amber-500/30">
                {user?.tier.toUpperCase()} TIER
              </span>
            </div>
            <div className="text-xs text-slate-400">
              {lang === 'fa' ? 'تاریخ انقضای اشتراک:' : 'Tier Expires At:'}{' '}
              <span className="text-slate-200 font-medium">
                {formatDate(user?.tier_expires_at, lang)}
              </span>
            </div>
          </div>

          {/* Tier description list */}
          <div className="space-y-2 text-xs text-slate-300">
            <div className="flex items-start gap-2 p-2.5 rounded-lg bg-slate-950 border border-slate-800">
              <Shield className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
              <span>{t('settings.tierUltimate')}</span>
            </div>
            <div className="flex items-start gap-2 p-2.5 rounded-lg bg-slate-950 border border-slate-800">
              <Shield className="w-4 h-4 text-blue-400 shrink-0 mt-0.5" />
              <span>{t('settings.tierPro')}</span>
            </div>
            <div className="flex items-start gap-2 p-2.5 rounded-lg bg-slate-950 border border-slate-800">
              <Shield className="w-4 h-4 text-slate-400 shrink-0 mt-0.5" />
              <span>{t('settings.tierFree')}</span>
            </div>
          </div>
        </div>

        {/* Change Password Form */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-purple-500/10 text-purple-400 border border-purple-500/20 flex items-center justify-center">
              <KeyRound className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">{t('settings.changePassword')}</h2>
              <span className="text-xs text-slate-400">Update Web Panel Credentials</span>
            </div>
          </div>

          {passwordSuccess && (
            <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{passwordSuccess}</span>
            </div>
          )}

          {passwordError && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{passwordError}</span>
            </div>
          )}

          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('settings.currentPass')}
              </label>
              <input
                type="password"
                required
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('settings.newPass')}
              </label>
              <input
                type="password"
                required
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('settings.confirmPass')}
              </label>
              <input
                type="password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <button
              type="submit"
              disabled={passwordLoading}
              className="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold transition disabled:opacity-50 flex items-center gap-2 shadow-md shadow-indigo-600/20"
            >
              {passwordLoading && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
              <span>{t('settings.save')}</span>
            </button>
          </form>
        </div>

        {/* Language & UI Preferences */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-6 shadow-sm space-y-5">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-sky-500/10 text-sky-400 border border-sky-500/20 flex items-center justify-center">
              <Globe className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">
                {lang === 'fa' ? 'زبان و چیدمان پنل' : 'Language & Display'}
              </h2>
              <span className="text-xs text-slate-400">Localization and RTL alignment</span>
            </div>
          </div>

          <div className="space-y-3">
            <label className="block text-xs font-medium text-slate-300">
              {lang === 'fa' ? 'انتخاب زبان رابط کاربری:' : 'Interface Language:'}
            </label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setLang('fa' as Language)}
                className={cn(
                  'p-4 rounded-xl border text-start flex flex-col gap-1 transition',
                  lang === 'fa'
                    ? 'bg-indigo-600/20 border-indigo-500 text-white'
                    : 'bg-slate-950 border-slate-800 text-slate-400 hover:text-white'
                )}
              >
                <span className="font-bold text-sm">فارسی (RTL)</span>
                <span className="text-xs text-slate-400">جهت راست به چپ همراه با تقویم و ارقام</span>
              </button>

              <button
                type="button"
                onClick={() => setLang('en' as Language)}
                className={cn(
                  'p-4 rounded-xl border text-start flex flex-col gap-1 transition',
                  lang === 'en'
                    ? 'bg-indigo-600/20 border-indigo-500 text-white'
                    : 'bg-slate-950 border-slate-800 text-slate-400 hover:text-white'
                )}
              >
                <span className="font-bold text-sm">English (LTR)</span>
                <span className="text-xs text-slate-400">Left-to-Right layout with standard formats</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
