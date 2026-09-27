import React, { useEffect, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import {
  Server,
  Users,
  DollarSign,
  AlertTriangle,
  ArrowUpRight,
  RefreshCw,
  TrendingUp,
  Activity,
  Layers,
} from 'lucide-react';
import { api } from '../api/client';
import type { Stats, Service } from '../types';
import { useLanguage } from '../context/LanguageContext';
import { formatCurrency } from '../lib/utils';
import { Link } from 'react-router-dom';

export const Dashboard: React.FC = () => {
  const { t, lang } = useLanguage();
  const [stats, setStats] = useState<Stats | null>(null);
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [sData, srvData] = await Promise.all([
        api.getStats(),
        api.getServices(),
      ]);
      setStats(sData);
      setServices(srvData);
    } catch (err) {
      console.error('Failed to load dashboard data', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  // 1. Traffic Line Chart Options
  const getTrafficChartOption = () => {
    if (!stats) return {};
    const dates = stats.traffic_chart_data.map((d) => d.date);
    const uploads = stats.traffic_chart_data.map((d) => d.upload_gb);
    const downloads = stats.traffic_chart_data.map((d) => d.download_gb);
    const totals = stats.traffic_chart_data.map((d) => d.total_gb);

    return {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#0f172a',
        borderColor: '#334155',
        textStyle: { color: '#f8fafc' },
      },
      legend: {
        data: [
          lang === 'fa' ? 'آپلود (GB)' : 'Upload (GB)',
          lang === 'fa' ? 'دانلود (GB)' : 'Download (GB)',
          lang === 'fa' ? 'مجموع (GB)' : 'Total (GB)',
        ],
        textStyle: { color: '#94a3b8' },
        top: 0,
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: dates,
        axisLine: { lineStyle: { color: '#334155' } },
        axisLabel: { color: '#94a3b8' },
      },
      yAxis: {
        type: 'value',
        splitLine: { lineStyle: { color: '#1e293b' } },
        axisLabel: { color: '#94a3b8' },
      },
      series: [
        {
          name: lang === 'fa' ? 'آپلود (GB)' : 'Upload (GB)',
          type: 'line',
          smooth: true,
          data: uploads,
          color: '#38bdf8',
          areaStyle: {
            opacity: 0.15,
            color: '#38bdf8',
          },
        },
        {
          name: lang === 'fa' ? 'دانلود (GB)' : 'Download (GB)',
          type: 'line',
          smooth: true,
          data: downloads,
          color: '#a855f7',
          areaStyle: {
            opacity: 0.15,
            color: '#a855f7',
          },
        },
        {
          name: lang === 'fa' ? 'مجموع (GB)' : 'Total (GB)',
          type: 'line',
          smooth: true,
          data: totals,
          color: '#6366f1',
          areaStyle: {
            opacity: 0.2,
            color: '#6366f1',
          },
        },
      ],
    };
  };

  // 2. Revenue Bar Chart Options
  const getRevenueChartOption = () => {
    if (!stats) return {};
    const dates = stats.revenue_chart_data.map((d) => d.date);
    const revenues = stats.revenue_chart_data.map((d) => d.revenue);

    return {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#0f172a',
        borderColor: '#334155',
        textStyle: { color: '#f8fafc' },
        formatter: (params: { name: string; value: number }[]) => {
          if (!params || !params[0]) return '';
          const p = params[0];
          return `${p.name}<br/>${formatCurrency(p.value, 'IRT', lang)}`;
        },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: dates,
        axisLine: { lineStyle: { color: '#334155' } },
        axisLabel: { color: '#94a3b8' },
      },
      yAxis: {
        type: 'value',
        splitLine: { lineStyle: { color: '#1e293b' } },
        axisLabel: {
          color: '#94a3b8',
          formatter: (val: number) => {
            if (val >= 1000000) return `${(val / 1000000).toFixed(0)}M`;
            return `${val}`;
          },
        },
      },
      series: [
        {
          type: 'bar',
          data: revenues,
          barWidth: '40%',
          itemStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: '#10b981' },
                { offset: 1, color: '#047857' },
              ],
            },
            borderRadius: [6, 6, 0, 0],
          },
        },
      ],
    };
  };

  // 3. Status Doughnut Chart Options
  const getDoughnutChartOption = () => {
    const activeCount = services.filter((s) => s.status === 'active').length || (stats?.active_services ?? 0);
    const expiredCount = services.filter((s) => s.status === 'expired').length || (stats?.expired_services ?? 0);
    const disabledCount = services.filter((s) => s.status === 'disabled').length;

    return {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        backgroundColor: '#0f172a',
        borderColor: '#334155',
        textStyle: { color: '#f8fafc' },
      },
      legend: {
        bottom: 0,
        textStyle: { color: '#94a3b8' },
      },
      series: [
        {
          name: lang === 'fa' ? 'وضعیت سرویس' : 'Service Status',
          type: 'pie',
          radius: ['45%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 8,
            borderColor: '#0b0f19',
            borderWidth: 2,
          },
          label: {
            show: false,
            position: 'center',
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold',
              color: '#f8fafc',
            },
          },
          data: [
            {
              value: activeCount,
              name: t('status.active'),
              itemStyle: { color: '#10b981' },
            },
            {
              value: expiredCount,
              name: t('status.expired'),
              itemStyle: { color: '#f59e0b' },
            },
            {
              value: disabledCount,
              name: t('status.disabled'),
              itemStyle: { color: '#ef4444' },
            },
          ],
        },
      ],
    };
  };

  return (
    <div className="space-y-6">
      {/* Title & Refresh header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            <span>{t('dashboard.title')}</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">{t('dashboard.subtitle')}</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={fetchData}
            disabled={loading}
            className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 hover:text-white hover:bg-slate-800 text-sm transition"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
            <span>{loading ? t('common.loading') : (lang === 'fa' ? 'بروزرسانی' : 'Refresh')}</span>
          </button>
        </div>
      </div>

      {/* 4 Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Card 1: Active Services */}
        <div className="bg-slate-900/80 border border-slate-800/90 rounded-2xl p-5 relative overflow-hidden group hover:border-indigo-500/40 transition">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">{t('dashboard.activeServices')}</span>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
              <Server className="w-5 h-5" />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-2xl sm:text-3xl font-bold text-white tracking-tight">
              {stats?.active_services ?? 0}
            </span>
            <span className="text-xs text-emerald-400 flex items-center font-medium">
              <TrendingUp className="w-3.5 h-3.5 me-1" />
              +12%
            </span>
          </div>
        </div>

        {/* Card 2: Expired Services */}
        <div className="bg-slate-900/80 border border-slate-800/90 rounded-2xl p-5 relative overflow-hidden group hover:border-amber-500/40 transition">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">{t('dashboard.expiredServices')}</span>
            <div className="w-10 h-10 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400">
              <AlertTriangle className="w-5 h-5" />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-2xl sm:text-3xl font-bold text-white tracking-tight">
              {stats?.expired_services ?? 0}
            </span>
            <span className="text-xs text-amber-400 font-medium">
              {lang === 'fa' ? 'نیاز به تمدید' : 'Needs renewal'}
            </span>
          </div>
        </div>

        {/* Card 3: Total Customers */}
        <div className="bg-slate-900/80 border border-slate-800/90 rounded-2xl p-5 relative overflow-hidden group hover:border-purple-500/40 transition">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">{t('dashboard.totalCustomers')}</span>
            <div className="w-10 h-10 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400">
              <Users className="w-5 h-5" />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-2xl sm:text-3xl font-bold text-white tracking-tight">
              {stats?.total_customers ?? 0}
            </span>
            <span className="text-xs text-indigo-400 font-medium">
              {lang === 'fa' ? 'کاربران ربات' : 'Bot users'}
            </span>
          </div>
        </div>

        {/* Card 4: Monthly Revenue */}
        <div className="bg-slate-900/80 border border-slate-800/90 rounded-2xl p-5 relative overflow-hidden group hover:border-emerald-500/40 transition">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">{t('dashboard.monthlyRevenue')}</span>
            <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-400">
              <DollarSign className="w-5 h-5" />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-xl sm:text-2xl font-bold text-emerald-400 tracking-tight">
              {formatCurrency(stats?.monthly_revenue ?? 0, 'IRT', lang)}
            </span>
            <span className="text-xs text-emerald-400 flex items-center font-medium">
              <TrendingUp className="w-3.5 h-3.5 me-1" />
              +24%
            </span>
          </div>
        </div>
      </div>

      {/* Charts Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Traffic Line Chart (spans 2 cols) */}
        <div className="lg:col-span-2 bg-slate-900/80 border border-slate-800 rounded-2xl p-5 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Activity className="w-4 h-4 text-indigo-400" />
              <h2 className="text-sm font-semibold text-white">{t('dashboard.trafficTrend')}</h2>
            </div>
            <span className="text-xs text-slate-400">7 {lang === 'fa' ? 'روز اخیر' : 'Days'}</span>
          </div>
          <div className="h-72 w-full">
            <ReactECharts
              option={getTrafficChartOption()}
              style={{ height: '100%', width: '100%' }}
              notMerge={true}
              lazyUpdate={true}
            />
          </div>
        </div>

        {/* Doughnut Chart: Service Status */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Layers className="w-4 h-4 text-purple-400" />
              <h2 className="text-sm font-semibold text-white">{t('dashboard.statusDistribution')}</h2>
            </div>
          </div>
          <div className="h-72 w-full">
            <ReactECharts
              option={getDoughnutChartOption()}
              style={{ height: '100%', width: '100%' }}
              notMerge={true}
              lazyUpdate={true}
            />
          </div>
        </div>
      </div>

      {/* Second Row: Revenue Chart + Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Revenue Bar Chart (spans 2 cols) */}
        <div className="lg:col-span-2 bg-slate-900/80 border border-slate-800 rounded-2xl p-5 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <DollarSign className="w-4 h-4 text-emerald-400" />
              <h2 className="text-sm font-semibold text-white">{t('dashboard.revenueTrend')}</h2>
            </div>
            <span className="text-xs text-slate-400">{lang === 'fa' ? 'ماه جاری' : 'Current Month'}</span>
          </div>
          <div className="h-64 w-full">
            <ReactECharts
              option={getRevenueChartOption()}
              style={{ height: '100%', width: '100%' }}
              notMerge={true}
              lazyUpdate={true}
            />
          </div>
        </div>

        {/* Quick Actions Card */}
        <div className="bg-slate-900/80 border border-slate-800 rounded-2xl p-5 shadow-sm flex flex-col justify-between">
          <div>
            <h2 className="text-sm font-semibold text-white mb-3 flex items-center gap-2">
              <ArrowUpRight className="w-4 h-4 text-indigo-400" />
              <span>{t('dashboard.quickActions')}</span>
            </h2>
            <div className="space-y-2.5">
              <Link
                to="/services"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/60 hover:bg-slate-800 border border-slate-800/80 text-sm text-slate-200 transition group"
              >
                <div className="flex items-center gap-2.5">
                  <Server className="w-4 h-4 text-indigo-400" />
                  <span>{lang === 'fa' ? 'مدیریت و فیلتر سرویس‌ها' : 'Manage & Filter Services'}</span>
                </div>
                <ArrowUpRight className="w-4 h-4 text-slate-500 group-hover:text-indigo-400 transition" />
              </Link>

              <Link
                to="/orders"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/60 hover:bg-slate-800 border border-slate-800/80 text-sm text-slate-200 transition group"
              >
                <div className="flex items-center gap-2.5">
                  <TrendingUp className="w-4 h-4 text-emerald-400" />
                  <span>{lang === 'fa' ? 'بررسی فیش‌های در انتظار' : 'Pending Payment Approvals'}</span>
                </div>
                <ArrowUpRight className="w-4 h-4 text-slate-500 group-hover:text-emerald-400 transition" />
              </Link>

              <Link
                to="/child-bots"
                className="flex items-center justify-between p-3 rounded-xl bg-slate-950/60 hover:bg-slate-800 border border-slate-800/80 text-sm text-slate-200 transition group"
              >
                <div className="flex items-center gap-2.5">
                  <Users className="w-4 h-4 text-purple-400" />
                  <span>{lang === 'fa' ? 'پیکربندی ربات‌های فرزند' : 'Child Customer Bots'}</span>
                </div>
                <ArrowUpRight className="w-4 h-4 text-slate-500 group-hover:text-purple-400 transition" />
              </Link>
            </div>
          </div>

          <div className="mt-4 p-3 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-xs text-indigo-300">
            {lang === 'fa'
              ? 'تمامی سرویس‌های مشتریان به صورت مستقیم با نودهای گروه سرور شما در 3x-ui همگام‌سازی می‌گردند.'
              : 'All customer instances sync directly with your dedicated 3x-ui server node group.'}
          </div>
        </div>
      </div>
    </div>
  );
};
