import React, { useEffect, useState } from 'react';
import {
  Search,
  RotateCcw,
  RefreshCw,
  QrCode,
  Copy,
  Check,
  X,
  ExternalLink,
  ShieldAlert,
} from 'lucide-react';
import { api } from '../api/client';
import type { Service, ServiceStatus } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { formatBytes, formatDate, cn } from '../lib/utils';

export const Services: React.FC = () => {
  const { t, lang } = useLanguage();
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [search, setSearch] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  // Modal states
  const [qrModalService, setQrModalService] = useState<Service | null>(null);
  const [copied, setCopied] = useState<boolean>(false);
  const [actionPrompt, setActionPrompt] = useState<{
    type: 'reset' | 'rotate';
    service: Service;
  } | null>(null);
  const [actionLoading, setActionLoading] = useState<boolean>(false);
  const [bannerNotice, setBannerNotice] = useState<string | null>(null);

  const fetchServices = async () => {
    setLoading(true);
    try {
      const data = await api.getServices({ search, status: statusFilter });
      setServices(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServices();
  }, [statusFilter]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    fetchServices();
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const executeResetTraffic = async () => {
    if (!actionPrompt) return;
    setActionLoading(true);
    try {
      await api.resetTraffic(actionPrompt.service.id);
      setServices((prev) =>
        prev.map((s) =>
          s.id === actionPrompt.service.id ? { ...s, up_bytes: 0, down_bytes: 0 } : s
        )
      );
      setBannerNotice(
        lang === 'fa'
          ? `ترافیک مصرفی سرویس ${actionPrompt.service.client_email} با موفقیت صفر شد.`
          : `Traffic usage for ${actionPrompt.service.client_email} has been reset to 0.`
      );
      setActionPrompt(null);
    } catch {
      alert(t('common.error'));
    } finally {
      setActionLoading(false);
    }
  };

  const executeRotateSubId = async () => {
    if (!actionPrompt) return;
    setActionLoading(true);
    try {
      const res = await api.rotateSubId(actionPrompt.service.id);
      setServices((prev) =>
        prev.map((s) =>
          s.id === actionPrompt.service.id
            ? { ...s, sub_id: res.sub_id, subscription_url: res.subscription_url }
            : s
        )
      );
      setBannerNotice(
        lang === 'fa'
          ? `شناسه ساب با موفقیت تغییر کرد. لینک اشتراک جدید اعمال شد.`
          : `SubId successfully rotated. New subscription URL assigned.`
      );
      setActionPrompt(null);
    } catch {
      alert(t('common.error'));
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusBadge = (status: ServiceStatus) => {
    switch (status) {
      case 'active':
        return (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            {t('status.active')}
          </span>
        );
      case 'expired':
        return (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">
            {t('status.expired')}
          </span>
        );
      case 'disabled':
        return (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20">
            {t('status.disabled')}
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-700/50 text-slate-300">
            {status}
          </span>
        );
    }
  };

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">{t('services.title')}</h1>
          <p className="text-slate-400 text-sm mt-1">{t('services.subtitle')}</p>
        </div>
        <button
          type="button"
          onClick={fetchServices}
          disabled={loading}
          className="self-start sm:self-auto flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white text-sm transition"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
          <span>{loading ? t('common.loading') : (lang === 'fa' ? 'بروزرسانی' : 'Refresh')}</span>
        </button>
      </div>

      {/* Banner notice if any */}
      {bannerNotice && (
        <div className="p-4 rounded-xl bg-indigo-500/10 border border-indigo-500/30 text-indigo-300 text-sm flex items-center justify-between">
          <span>{bannerNotice}</span>
          <button
            type="button"
            onClick={() => setBannerNotice(null)}
            className="text-slate-400 hover:text-white"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Filter and Search Bar */}
      <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-4 flex flex-col md:flex-row gap-4 justify-between items-center">
        {/* Search */}
        <form onSubmit={handleSearchSubmit} className="relative w-full md:w-96">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('services.search')}
            className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-2 text-sm text-white placeholder-slate-400 focus:outline-none focus:border-indigo-500"
          />
          <button
            type="submit"
            className="absolute end-2 top-1/2 -translate-y-1/2 p-1.5 text-slate-400 hover:text-white"
          >
            <Search className="w-4 h-4" />
          </button>
        </form>

        {/* Status Filters */}
        <div className="flex items-center gap-1.5 overflow-x-auto w-full md:w-auto pb-1 md:pb-0">
          {['all', 'active', 'expired', 'disabled'].map((st) => (
            <button
              key={st}
              type="button"
              onClick={() => setStatusFilter(st)}
              className={cn(
                'px-3.5 py-1.5 rounded-lg text-xs font-medium transition shrink-0',
                statusFilter === st
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              )}
            >
              {st === 'all'
                ? t('status.all')
                : st === 'active'
                ? t('status.active')
                : st === 'expired'
                ? t('status.expired')
                : t('status.disabled')}
            </button>
          ))}
        </div>
      </div>

      {/* Data Table */}
      <div className="bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-start text-sm">
            <thead className="bg-slate-950/60 border-b border-slate-800 text-slate-400 text-xs font-semibold">
              <tr>
                <th className="py-3.5 px-4 text-start">{t('services.email')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.plan')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.ipLimit')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.traffic')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.expiry')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.status')}</th>
                <th className="py-3.5 px-4 text-end">{t('services.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {services.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-slate-500">
                    {lang === 'fa' ? 'هیچ سرویسی با این مشخصات یافت نشد.' : 'No services found.'}
                  </td>
                </tr>
              ) : (
                services.map((srv) => {
                  const usedBytes = srv.up_bytes + srv.down_bytes;
                  const percent = srv.total_bytes > 0
                    ? Math.min(100, Math.round((usedBytes / srv.total_bytes) * 100))
                    : 0;

                  return (
                    <tr key={srv.id} className="hover:bg-slate-800/40 transition">
                      <td className="py-3 px-4 font-mono text-xs text-indigo-300">
                        {srv.client_email}
                      </td>
                      <td className="py-3 px-4 font-medium text-slate-200">
                        {srv.plan_name}
                      </td>
                      <td className="py-3 px-4 text-slate-300">
                        <span className="px-2 py-0.5 rounded bg-slate-800 text-xs font-mono">
                          {srv.user_count} IP
                        </span>
                      </td>
                      <td className="py-3 px-4 min-w-[180px]">
                        <div className="space-y-1">
                          <div className="flex justify-between text-xs text-slate-400 font-mono">
                            <span>{formatBytes(usedBytes)}</span>
                            <span>{formatBytes(srv.total_bytes)}</span>
                          </div>
                          <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                            <div
                              className={cn(
                                'h-full transition-all duration-300 rounded-full',
                                percent > 90
                                  ? 'bg-rose-500'
                                  : percent > 75
                                  ? 'bg-amber-500'
                                  : 'bg-indigo-500'
                              )}
                              style={{ width: `${percent}%` }}
                            />
                          </div>
                        </div>
                      </td>
                      <td className="py-3 px-4 text-xs text-slate-400">
                        {formatDate(srv.expiry_time_ms, lang)}
                      </td>
                      <td className="py-3 px-4">
                        {getStatusBadge(srv.status)}
                      </td>
                      <td className="py-3 px-4 text-end">
                        <div className="flex items-center justify-end gap-1.5">
                          {/* QR Code & Link Modal Trigger */}
                          <button
                            type="button"
                            onClick={() => setQrModalService(srv)}
                            className="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition"
                            title={t('services.showQr')}
                          >
                            <QrCode className="w-4 h-4" />
                          </button>

                          {/* Reset Traffic Trigger */}
                          <button
                            type="button"
                            onClick={() => setActionPrompt({ type: 'reset', service: srv })}
                            className="p-1.5 rounded-lg bg-slate-800 hover:bg-amber-500/20 text-slate-300 hover:text-amber-400 transition"
                            title={t('services.resetTraffic')}
                          >
                            <RotateCcw className="w-4 h-4" />
                          </button>

                          {/* Rotate SubId Trigger */}
                          <button
                            type="button"
                            onClick={() => setActionPrompt({ type: 'rotate', service: srv })}
                            className="p-1.5 rounded-lg bg-slate-800 hover:bg-rose-500/20 text-slate-300 hover:text-rose-400 transition"
                            title={t('services.rotateSub')}
                          >
                            <RefreshCw className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* QR Code / Link Modal */}
      {qrModalService && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md p-6 relative shadow-2xl">
            <button
              type="button"
              onClick={() => setQrModalService(null)}
              className="absolute top-4 end-4 p-1.5 text-slate-400 hover:text-white rounded-lg"
            >
              <X className="w-5 h-5" />
            </button>

            <h3 className="text-lg font-bold text-white mb-1">
              {t('services.showQr')}
            </h3>
            <p className="text-xs text-slate-400 mb-4 font-mono">
              {qrModalService.client_email}
            </p>

            {/* QR Code Container */}
            <div className="bg-white p-4 rounded-xl flex items-center justify-center mx-auto my-3 w-52 h-52 shadow-inner">
              <img
                src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(
                  qrModalService.subscription_url
                )}`}
                alt="Subscription QR"
                className="w-44 h-44 object-contain"
                onError={(e) => {
                  // Fallback if offline
                  (e.target as HTMLElement).style.display = 'none';
                }}
              />
            </div>

            {/* URL Display with Copy Button */}
            <div className="mt-4 space-y-2">
              <label className="text-xs text-slate-400 font-medium">
                {t('services.copyLink')}
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  readOnly
                  value={qrModalService.subscription_url}
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-slate-300 select-all"
                />
                <button
                  type="button"
                  onClick={() => handleCopy(qrModalService.subscription_url)}
                  className="px-3 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold flex items-center gap-1.5 shrink-0 transition"
                >
                  {copied ? <Check className="w-4 h-4 text-white" /> : <Copy className="w-4 h-4" />}
                  <span>{copied ? t('services.copied') : t('common.save')}</span>
                </button>
              </div>
            </div>

            <div className="mt-4 pt-3 border-t border-slate-800 flex justify-between items-center text-xs text-slate-400">
              <span>Sub ID: <span className="font-mono text-slate-300">{qrModalService.sub_id}</span></span>
              <a
                href={qrModalService.subscription_url}
                target="_blank"
                rel="noreferrer"
                className="text-indigo-400 hover:underline flex items-center gap-1"
              >
                <span>Direct Link</span>
                <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          </div>
        </div>
      )}

      {/* Confirmation Prompt Dialog for Reset / Rotate */}
      {actionPrompt && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-sm p-6 shadow-2xl">
            <div className="flex items-center gap-3 mb-4">
              <div
                className={cn(
                  'w-10 h-10 rounded-xl flex items-center justify-center',
                  actionPrompt.type === 'reset'
                    ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                    : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                )}
              >
                {actionPrompt.type === 'reset' ? (
                  <RotateCcw className="w-5 h-5" />
                ) : (
                  <ShieldAlert className="w-5 h-5" />
                )}
              </div>
              <h3 className="text-base font-bold text-white">
                {actionPrompt.type === 'reset'
                  ? t('services.resetTraffic')
                  : t('services.rotateSub')}
              </h3>
            </div>

            <p className="text-sm text-slate-300 mb-5 leading-relaxed">
              {actionPrompt.type === 'reset'
                ? t('services.confirmReset')
                : t('services.confirmRotate')}
            </p>

            <div className="text-xs font-mono bg-slate-950 p-2.5 rounded-lg border border-slate-800 text-slate-400 mb-5">
              Email: {actionPrompt.service.client_email}
            </div>

            <div className="flex items-center justify-end gap-2.5">
              <button
                type="button"
                onClick={() => setActionPrompt(null)}
                disabled={actionLoading}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium transition"
              >
                {t('common.cancel')}
              </button>

              <button
                type="button"
                onClick={actionPrompt.type === 'reset' ? executeResetTraffic : executeRotateSubId}
                disabled={actionLoading}
                className={cn(
                  'px-4 py-2 rounded-xl text-white text-xs font-semibold shadow-md transition disabled:opacity-50 flex items-center gap-1.5',
                  actionPrompt.type === 'reset'
                    ? 'bg-amber-600 hover:bg-amber-500 shadow-amber-600/20'
                    : 'bg-rose-600 hover:bg-rose-500 shadow-rose-600/20'
                )}
              >
                {actionLoading && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                <span>{t('common.confirm')}</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
