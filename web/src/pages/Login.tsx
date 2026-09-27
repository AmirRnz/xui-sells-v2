import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ShieldCheck, Lock, User, AlertCircle, ArrowRight, ArrowLeft, Globe } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

export const Login: React.FC = () => {
  const { login } = useAuth();
  const { lang, setLang, isRTL } = useLanguage();
  const navigate = useNavigate();

  const [tgId, setTgId] = useState<string>('987654321');
  const [password, setPassword] = useState<string>('admin123');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const numericId = parseInt(tgId.trim(), 10);
    if (isNaN(numericId) || numericId <= 0) {
      setError(
        lang === 'fa'
          ? 'شناسه تلگرام باید یک مقدار عددی معتبر باشد'
          : 'Telegram ID must be a valid numeric ID'
      );
      return;
    }

    if (!password.trim()) {
      setError(lang === 'fa' ? 'لطفاً کلمه عبور را وارد کنید' : 'Please enter your password');
      return;
    }

    setLoading(true);
    try {
      await login(numericId, password);
      navigate('/');
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : lang === 'fa'
          ? 'ورود ناموفق بود. اطلاعات ورود را بررسی فرمایید.'
          : 'Failed to sign in. Please verify your credentials.'
      );
    } finally {
      setLoading(false);
    }
  };

  const toggleLanguage = () => {
    setLang(lang === 'fa' ? 'en' : 'fa');
  };

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col justify-center items-center px-4 relative overflow-hidden">
      {/* Background ambient lighting */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />

      {/* Language Switcher Top Corner */}
      <div className="absolute top-6 right-6 z-10">
        <button
          type="button"
          onClick={toggleLanguage}
          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-white text-xs font-medium transition"
        >
          <Globe className="w-3.5 h-3.5 text-indigo-400" />
          <span>{lang === 'fa' ? 'English' : 'فارسی'}</span>
        </button>
      </div>

      <div className="w-full max-w-md relative z-10">
        {/* Logo and Brand Title */}
        <div className="text-center mb-8">
          <div className="inline-flex w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-500 via-indigo-600 to-purple-600 items-center justify-center shadow-xl shadow-indigo-500/25 mb-4">
            <ShieldCheck className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-white">
            {lang === 'fa' ? 'ورود به پنل نمایندگی' : 'Reseller Web Portal'}
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            {lang === 'fa'
              ? 'مدیریت اشتراک‌ها، ربات‌های مشتریان و تایید پرداخت‌ها'
              : 'Sign in with your Telegram ID and bot password'}
          </p>
        </div>

        {/* Card */}
        <div className="bg-slate-900/90 border border-slate-800 rounded-2xl p-6 sm:p-8 shadow-2xl backdrop-blur-xl">
          {error && (
            <div className="mb-5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-sm flex items-start gap-2.5">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                {lang === 'fa' ? 'شناسه عددی تلگرام (Numeric TG ID)' : 'Telegram ID (Numeric)'}
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 start-0 flex items-center ps-3.5 pointer-events-none text-slate-400">
                  <User className="w-4 h-4" />
                </div>
                <input
                  type="number"
                  required
                  value={tgId}
                  onChange={(e) => setTgId(e.target.value)}
                  placeholder="e.g. 123456789"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 ps-10 text-white placeholder-slate-400 text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
                />
              </div>
              <p className="text-[11px] text-slate-400 mt-1">
                {lang === 'fa'
                  ? 'شناسه عددی اکانت تلگرام متصل به ربات نمایندگی'
                  : 'Your Telegram numerical account identifier'}
              </p>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                {lang === 'fa' ? 'رمز عبور پنل' : 'Web Panel Password'}
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 start-0 flex items-center ps-3.5 pointer-events-none text-slate-400">
                  <Lock className="w-4 h-4" />
                </div>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2.5 ps-10 text-white placeholder-slate-400 text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
                />
              </div>
              <p className="text-[11px] text-slate-400 mt-1">
                {lang === 'fa'
                  ? 'کلمه عبوری که از طریق دستور bot تنظیم نموده‌اید'
                  : 'The web password configured via Telegram bot'}
              </p>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full mt-2 py-3 px-4 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-500 hover:to-purple-500 text-white text-sm font-semibold shadow-lg shadow-indigo-600/30 flex items-center justify-center gap-2 transition disabled:opacity-50"
            >
              <span>
                {loading
                  ? (lang === 'fa' ? 'در حال بررسی...' : 'Authenticating...')
                  : (lang === 'fa' ? 'ورود به سیستم' : 'Sign In')}
              </span>
              {!loading && (
                isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />
              )}
            </button>
          </form>

          <div className="mt-6 pt-5 border-t border-slate-800 text-center">
            <span className="text-xs text-slate-400">
              {lang === 'fa'
                ? 'رمز عبور را فراموش کرده‌اید؟ در ربات تلگرام دستور /password را ارسال کنید.'
                : 'Forgot password? Use /password command in the Telegram reseller bot.'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
