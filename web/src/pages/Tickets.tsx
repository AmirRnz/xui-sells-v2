import React, { useEffect, useState } from 'react';
import {
  MessageSquare,
  Send,
  User,
  ShieldCheck,
  RefreshCw,
  Clock,
  CheckCircle,
  HelpCircle,
} from 'lucide-react';
import { api } from '../api/client';
import type { Ticket, TicketStatus } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { formatDate, cn } from '../lib/utils';

export const Tickets: React.FC = () => {
  const { t, lang } = useLanguage();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedTicketId, setSelectedTicketId] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [replyText, setReplyText] = useState<string>('');
  const [sending, setSending] = useState<boolean>(false);

  const fetchTickets = async () => {
    setLoading(true);
    try {
      const data = await api.getTickets();
      setTickets(data);
      if (data.length > 0 && !selectedTicketId) {
        setSelectedTicketId(data[0].id);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTickets();
  }, []);

  const selectedTicket = tickets.find((tkt) => tkt.id === selectedTicketId);

  const handleSendReply = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTicketId || !replyText.trim()) return;

    setSending(true);
    try {
      const newMsg = await api.replyTicket(selectedTicketId, replyText.trim());
      setTickets((prev) =>
        prev.map((tkt) => {
          if (tkt.id === selectedTicketId) {
            return {
              ...tkt,
              status: 'answered' as TicketStatus,
              messages: [...tkt.messages, newMsg],
            };
          }
          return tkt;
        })
      );
      setReplyText('');
    } catch {
      alert(t('common.error'));
    } finally {
      setSending(false);
    }
  };

  const getStatusBadge = (st: TicketStatus) => {
    switch (st) {
      case 'open':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Clock className="w-3 h-3" />
            {t('tickets.open')}
          </span>
        );
      case 'answered':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            <CheckCircle className="w-3 h-3" />
            {t('tickets.answered')}
          </span>
        );
      case 'closed':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-700/50 text-slate-300">
            {t('tickets.closed')}
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
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            <MessageSquare className="w-6 h-6 text-indigo-400" />
            <span>{t('tickets.title')}</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">{t('tickets.subtitle')}</p>
        </div>
        <button
          type="button"
          onClick={fetchTickets}
          disabled={loading}
          className="self-start sm:self-auto flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white text-sm transition"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
          <span>{loading ? t('common.loading') : (lang === 'fa' ? 'بروزرسانی' : 'Refresh')}</span>
        </button>
      </div>

      {/* Main 2-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 min-h-[600px]">
        {/* Ticket List (Left 4 cols) */}
        <div className="lg:col-span-4 bg-slate-900/80 border border-slate-800 rounded-2xl overflow-hidden flex flex-col">
          <div className="p-4 border-b border-slate-800 flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
              {lang === 'fa' ? 'لیست تیکت‌ها' : 'All Inquiries'} ({tickets.length})
            </span>
          </div>

          <div className="divide-y divide-slate-800/80 overflow-y-auto flex-1 max-h-[600px]">
            {tickets.length === 0 ? (
              <div className="p-8 text-center text-slate-500 text-sm">
                {lang === 'fa' ? 'هیچ تیکتی وجود ندارد' : 'No tickets found'}
              </div>
            ) : (
              tickets.map((tkt) => {
                const isSelected = tkt.id === selectedTicketId;
                const lastMsg = tkt.messages[tkt.messages.length - 1];

                return (
                  <button
                    key={tkt.id}
                    type="button"
                    onClick={() => setSelectedTicketId(tkt.id)}
                    className={cn(
                      'w-full text-start p-4 transition flex flex-col gap-1.5',
                      isSelected
                        ? 'bg-slate-800/80 border-s-4 border-indigo-500'
                        : 'hover:bg-slate-800/40'
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-mono text-indigo-300">TG: {tkt.user_tg_id}</span>
                      {getStatusBadge(tkt.status)}
                    </div>
                    <h4 className="text-sm font-semibold text-white truncate">{tkt.subject}</h4>
                    {lastMsg && (
                      <p className="text-xs text-slate-400 line-clamp-1">{lastMsg.message}</p>
                    )}
                    <span className="text-[10px] text-slate-500 self-end mt-1">
                      {formatDate(tkt.created_at, lang)}
                    </span>
                  </button>
                );
              })
            )}
          </div>
        </div>

        {/* Conversation Viewer & Reply (Right 8 cols) */}
        <div className="lg:col-span-8 bg-slate-900/80 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
          {selectedTicket ? (
            <>
              {/* Header */}
              <div className="p-4 border-b border-slate-800 bg-slate-950/40 flex items-center justify-between">
                <div>
                  <h3 className="font-bold text-white text-base">{selectedTicket.subject}</h3>
                  <div className="flex items-center gap-3 text-xs text-slate-400 mt-1">
                    <span>
                      {t('tickets.user')}: <span className="font-mono text-slate-200">{selectedTicket.user_tg_id}</span>
                    </span>
                    <span>&bull;</span>
                    <span>{formatDate(selectedTicket.created_at, lang)}</span>
                  </div>
                </div>
                <div>{getStatusBadge(selectedTicket.status)}</div>
              </div>

              {/* Message Feed */}
              <div className="flex-1 p-5 overflow-y-auto space-y-4 max-h-[460px]">
                {selectedTicket.messages.map((msg) => {
                  const isReseller = msg.sender_role === 'reseller' || msg.sender_role === 'support';

                  return (
                    <div
                      key={msg.id}
                      className={cn('flex flex-col', isReseller ? 'items-end' : 'items-start')}
                    >
                      <div className="flex items-center gap-1.5 mb-1 text-[11px] text-slate-400 px-1">
                        {isReseller ? (
                          <>
                            <ShieldCheck className="w-3.5 h-3.5 text-indigo-400" />
                            <span className="font-medium text-indigo-300">
                              {lang === 'fa' ? 'شما (پشتیبانی نمایندگی)' : 'You (Support)'}
                            </span>
                          </>
                        ) : (
                          <>
                            <User className="w-3.5 h-3.5 text-slate-400" />
                            <span className="font-mono text-slate-300">{selectedTicket.user_tg_id}</span>
                          </>
                        )}
                        <span>&bull;</span>
                        <span>{formatDate(msg.created_at, lang)}</span>
                      </div>

                      <div
                        className={cn(
                          'max-w-xl p-3.5 rounded-2xl text-sm leading-relaxed whitespace-pre-wrap break-words shadow-sm',
                          isReseller
                            ? 'bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-te-none'
                            : 'bg-slate-800 text-slate-200 border border-slate-700/60 rounded-ts-none'
                        )}
                      >
                        {msg.message}
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Reply Input Box */}
              <form onSubmit={handleSendReply} className="p-4 border-t border-slate-800 bg-slate-950/60">
                <div className="flex flex-col sm:flex-row items-end gap-3">
                  <textarea
                    rows={3}
                    required
                    value={replyText}
                    onChange={(e) => setReplyText(e.target.value)}
                    placeholder={t('tickets.typeReply')}
                    className="w-full bg-slate-900 border border-slate-800 rounded-xl p-3 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 resize-none transition"
                  />
                  <button
                    type="submit"
                    disabled={sending || !replyText.trim()}
                    className="w-full sm:w-auto px-5 py-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-semibold flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/25 shrink-0 transition"
                  >
                    {sending ? (
                      <RefreshCw className="w-4 h-4 animate-spin" />
                    ) : (
                      <Send className="w-4 h-4" />
                    )}
                    <span>{t('tickets.sendReply')}</span>
                  </button>
                </div>
              </form>
            </>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-slate-500">
              <HelpCircle className="w-12 h-12 mb-3 text-slate-600" />
              <p className="text-sm">{t('tickets.selectTicket')}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
