import React, { useEffect, useState } from 'react';
import {
  CheckCircle,
  XCircle,
  Eye,
  RefreshCw,
  Clock,
  Check,
  X,
  CreditCard,
  Image as ImageIcon,
  AlertTriangle,
} from 'lucide-react';
import { api } from '../api/client';
import type { Order, OrderStatus } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { formatCurrency, formatDate, cn } from '../lib/utils';

export const Orders: React.FC = () => {
  const { t, lang } = useLanguage();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [statusFilter, setStatusFilter] = useState<string>('pending');

  // Modals
  const [viewingReceiptOrder, setViewingReceiptOrder] = useState<Order | null>(null);
  const [rejectDialogOrder, setRejectDialogOrder] = useState<Order | null>(null);
  const [rejectReason, setRejectReason] = useState<string>('');
  const [actionLoading, setActionLoading] = useState<boolean>(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);

  const fetchOrders = async () => {
    setLoading(true);
    try {
      const data = await api.getOrders({ status: statusFilter });
      setOrders(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrders();
  }, [statusFilter]);

  const handleApprove = async (orderId: string) => {
    if (!window.confirm(t('orders.confirmApprove'))) return;
    setActionLoading(true);
    try {
      await api.approveOrder(orderId);
      setOrders((prev) =>
        prev.map((o) => (o.id === orderId ? { ...o, status: 'approved' as OrderStatus } : o))
      );
      if (viewingReceiptOrder?.id === orderId) {
        setViewingReceiptOrder((prev) => (prev ? { ...prev, status: 'approved' } : null));
      }
      setActionNotice(
        lang === 'fa'
          ? `سفارش ${orderId} با موفقیت تایید شد و سرویس فعال گردید.`
          : `Order ${orderId} approved and service provisioned.`
      );
    } catch {
      alert(t('common.error'));
    } finally {
      setActionLoading(false);
    }
  };

  const handleRejectSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rejectDialogOrder) return;
    if (!rejectReason.trim()) {
      alert(t('orders.rejectReasonPlaceholder'));
      return;
    }

    setActionLoading(true);
    try {
      await api.rejectOrder(rejectDialogOrder.id, rejectReason);
      setOrders((prev) =>
        prev.map((o) =>
          o.id === rejectDialogOrder.id ? { ...o, status: 'rejected' as OrderStatus } : o
        )
      );
      if (viewingReceiptOrder?.id === rejectDialogOrder.id) {
        setViewingReceiptOrder((prev) => (prev ? { ...prev, status: 'rejected' } : null));
      }
      setActionNotice(
        lang === 'fa'
          ? `سفارش ${rejectDialogOrder.id} رد شد.`
          : `Order ${rejectDialogOrder.id} was rejected.`
      );
      setRejectDialogOrder(null);
      setRejectReason('');
    } catch {
      alert(t('common.error'));
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusBadge = (st: OrderStatus) => {
    switch (st) {
      case 'pending':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Clock className="w-3 h-3" />
            {t('status.pending')}
          </span>
        );
      case 'approved':
      case 'completed':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <CheckCircle className="w-3 h-3" />
            {t('status.approved')}
          </span>
        );
      case 'rejected':
      case 'cancelled':
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20">
            <XCircle className="w-3 h-3" />
            {t('status.rejected')}
          </span>
        );
      default:
        return <span className="text-xs text-slate-400">{st}</span>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">{t('orders.title')}</h1>
          <p className="text-slate-400 text-sm mt-1">{t('orders.subtitle')}</p>
        </div>
        <button
          type="button"
          onClick={fetchOrders}
          disabled={loading}
          className="self-start sm:self-auto flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white text-sm transition"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
          <span>{loading ? t('common.loading') : (lang === 'fa' ? 'بروزرسانی' : 'Refresh')}</span>
        </button>
      </div>

      {actionNotice && (
        <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-sm flex items-center justify-between">
          <span>{actionNotice}</span>
          <button
            type="button"
            onClick={() => setActionNotice(null)}
            className="text-slate-400 hover:text-white"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Tabs / Filters */}
      <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
        {[
          { key: 'pending', label: t('status.pending') },
          { key: 'approved', label: t('status.approved') },
          { key: 'rejected', label: t('status.rejected') },
          { key: 'all', label: t('status.all') },
        ].map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setStatusFilter(tab.key)}
            className={cn(
              'px-4 py-2 rounded-xl text-xs sm:text-sm font-medium transition',
              statusFilter === tab.key
                ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Orders Table */}
      <div className="bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-start text-sm">
            <thead className="bg-slate-950/60 border-b border-slate-800 text-slate-400 text-xs font-semibold">
              <tr>
                <th className="py-3.5 px-4 text-start">{t('orders.id')}</th>
                <th className="py-3.5 px-4 text-start">{t('orders.user')}</th>
                <th className="py-3.5 px-4 text-start">{t('orders.type')}</th>
                <th className="py-3.5 px-4 text-start">{t('orders.amount')}</th>
                <th className="py-3.5 px-4 text-start">{t('orders.paymentMethod')}</th>
                <th className="py-3.5 px-4 text-start">{t('orders.receipt')}</th>
                <th className="py-3.5 px-4 text-start">{t('services.status')}</th>
                <th className="py-3.5 px-4 text-end">{t('services.actions')}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {orders.length === 0 ? (
                <tr>
                  <td colSpan={8} className="py-12 text-center text-slate-500">
                    {lang === 'fa' ? 'هیچ سفارشی یافت نشد.' : 'No orders found.'}
                  </td>
                </tr>
              ) : (
                orders.map((ord) => (
                  <tr key={ord.id} className="hover:bg-slate-800/40 transition">
                    <td className="py-3 px-4 font-mono text-xs text-indigo-300">
                      {ord.id}
                      <div className="text-[10px] text-slate-500">{formatDate(ord.created_at, lang)}</div>
                    </td>
                    <td className="py-3 px-4 font-mono text-xs text-slate-300">
                      {ord.user_tg_id}
                    </td>
                    <td className="py-3 px-4 text-slate-200">
                      <span className="font-medium">{ord.plan_name}</span>
                    </td>
                    <td className="py-3 px-4 font-semibold text-emerald-400 tabular-nums">
                      {formatCurrency(ord.amount, ord.currency, lang)}
                    </td>
                    <td className="py-3 px-4 text-xs text-slate-400 capitalize">
                      {ord.payment_method.replace('_', ' ')}
                    </td>
                    <td className="py-3 px-4">
                      {ord.receipt_text || ord.receipt_media_path ? (
                        <button
                          type="button"
                          onClick={() => setViewingReceiptOrder(ord)}
                          className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-indigo-300 transition"
                        >
                          <Eye className="w-3.5 h-3.5" />
                          <span>{t('orders.viewReceipt')}</span>
                          {ord.receipt_media_path && <ImageIcon className="w-3 h-3 text-slate-400" />}
                        </button>
                      ) : (
                        <span className="text-xs text-slate-500">-</span>
                      )}
                    </td>
                    <td className="py-3 px-4">
                      {getStatusBadge(ord.status)}
                    </td>
                    <td className="py-3 px-4 text-end">
                      {ord.status === 'pending' ? (
                        <div className="flex items-center justify-end gap-2">
                          <button
                            type="button"
                            onClick={() => handleApprove(ord.id)}
                            disabled={actionLoading}
                            className="px-2.5 py-1 rounded-lg bg-emerald-600/20 hover:bg-emerald-600 text-emerald-400 hover:text-white border border-emerald-500/30 text-xs font-semibold flex items-center gap-1 transition"
                            title={t('orders.approve')}
                          >
                            <Check className="w-3.5 h-3.5" />
                            <span>{t('orders.approve')}</span>
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setRejectDialogOrder(ord);
                              setRejectReason('');
                            }}
                            disabled={actionLoading}
                            className="px-2.5 py-1 rounded-lg bg-rose-600/20 hover:bg-rose-600 text-rose-400 hover:text-white border border-rose-500/30 text-xs font-semibold flex items-center gap-1 transition"
                            title={t('orders.reject')}
                          >
                            <X className="w-3.5 h-3.5" />
                            <span>{t('orders.reject')}</span>
                          </button>
                        </div>
                      ) : (
                        <span className="text-xs text-slate-500">-</span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Receipt Viewer Modal */}
      {viewingReceiptOrder && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-lg p-6 relative shadow-2xl max-h-[90vh] flex flex-col">
            <button
              type="button"
              onClick={() => setViewingReceiptOrder(null)}
              className="absolute top-4 end-4 p-1.5 text-slate-400 hover:text-white rounded-lg"
            >
              <X className="w-5 h-5" />
            </button>

            <h3 className="text-lg font-bold text-white mb-2 flex items-center gap-2">
              <CreditCard className="w-5 h-5 text-indigo-400" />
              <span>{t('orders.viewReceipt')}</span>
            </h3>

            <div className="overflow-y-auto space-y-4 my-3 pe-1">
              <div className="grid grid-cols-2 gap-3 text-xs bg-slate-950 p-3 rounded-xl border border-slate-800">
                <div>
                  <span className="text-slate-400 block">{t('orders.id')}</span>
                  <span className="font-mono text-slate-200">{viewingReceiptOrder.id}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">{t('orders.user')}</span>
                  <span className="font-mono text-slate-200">{viewingReceiptOrder.user_tg_id}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">{t('orders.amount')}</span>
                  <span className="font-semibold text-emerald-400">
                    {formatCurrency(viewingReceiptOrder.amount, viewingReceiptOrder.currency, lang)}
                  </span>
                </div>
                <div>
                  <span className="text-slate-400 block">{t('services.status')}</span>
                  <span>{getStatusBadge(viewingReceiptOrder.status)}</span>
                </div>
              </div>

              {/* Receipt Text */}
              <div>
                <label className="text-xs text-slate-400 block mb-1 font-medium">
                  {lang === 'fa' ? 'توضیحات واریز / شماره پیگیری:' : 'Payment Reference / Text:'}
                </label>
                <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 text-sm text-slate-200 font-mono">
                  {viewingReceiptOrder.receipt_text || t('orders.noReceipt')}
                </div>
              </div>

              {/* Receipt Image if available */}
              {viewingReceiptOrder.receipt_media_path && (
                <div>
                  <label className="text-xs text-slate-400 block mb-1 font-medium">
                    {lang === 'fa' ? 'تصویر فیش واریزی:' : 'Receipt Image Proof:'}
                  </label>
                  <div className="rounded-xl overflow-hidden border border-slate-800 bg-slate-950 max-h-72 flex items-center justify-center">
                    <img
                      src={viewingReceiptOrder.receipt_media_path}
                      alt="Receipt Attachment"
                      className="max-h-72 w-full object-contain"
                    />
                  </div>
                </div>
              )}
            </div>

            {/* Modal Footer with quick actions if pending */}
            {viewingReceiptOrder.status === 'pending' && (
              <div className="pt-3 border-t border-slate-800 flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => {
                    const ord = viewingReceiptOrder;
                    setRejectDialogOrder(ord);
                    setRejectReason('');
                  }}
                  className="px-4 py-2 rounded-xl bg-rose-600/20 hover:bg-rose-600 text-rose-400 hover:text-white border border-rose-500/30 text-xs font-semibold transition"
                >
                  {t('orders.reject')}
                </button>
                <button
                  type="button"
                  onClick={() => handleApprove(viewingReceiptOrder.id)}
                  className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold shadow-md shadow-emerald-600/20 transition"
                >
                  {t('orders.approve')}
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Reject with Reason Dialog */}
      {rejectDialogOrder && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
          <form
            onSubmit={handleRejectSubmit}
            className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-md p-6 shadow-2xl space-y-4"
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-rose-500/10 text-rose-400 border border-rose-500/20 flex items-center justify-center">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <h3 className="text-base font-bold text-white">{t('orders.reject')}</h3>
            </div>

            <p className="text-xs text-slate-400 leading-relaxed">
              {lang === 'fa'
                ? `دلیل رد سفارش شماره ${rejectDialogOrder.id} را وارد نمایید. این پیام به کاربر تلگرام ارسال می‌شود.`
                : `Enter the reason for rejecting order ${rejectDialogOrder.id}. This message will be sent to the customer.`}
            </p>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1">
                {t('orders.rejectReason')}
              </label>
              <textarea
                required
                rows={3}
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder={t('orders.rejectReasonPlaceholder')}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-sm text-white placeholder-slate-400 focus:outline-none focus:border-rose-500 transition"
              />
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-2">
              <button
                type="button"
                onClick={() => setRejectDialogOrder(null)}
                disabled={actionLoading}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium transition"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={actionLoading}
                className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold shadow-md shadow-rose-600/20 transition flex items-center gap-1.5"
              >
                {actionLoading && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                <span>{t('orders.reject')}</span>
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
