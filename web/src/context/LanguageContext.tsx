import React, { createContext, useContext, useEffect, useState } from 'react';

export type Language = 'en' | 'fa';

interface LanguageContextType {
  lang: Language;
  setLang: (lang: Language) => void;
  t: (key: string) => string;
  isRTL: boolean;
}

const translations: Record<Language, Record<string, string>> = {
  en: {
    // Navigation
    'nav.dashboard': 'Dashboard',
    'nav.services': 'Services',
    'nav.orders': 'Orders & Approvals',
    'nav.childBots': 'Child Bots',
    'nav.tickets': 'Support Tickets',
    'nav.settings': 'Settings',
    'nav.logout': 'Logout',

    // Header & User
    'header.wallet': 'Wallet Balance',
    'header.service': 'Service',
    'header.tier': 'Tier',
    'header.switchLang': 'فارسی',

    // Statuses
    'status.active': 'Active',
    'status.expired': 'Expired',
    'status.disabled': 'Disabled',
    'status.pending': 'Pending',
    'status.approved': 'Approved',
    'status.rejected': 'Rejected',
    'status.completed': 'Completed',
    'status.cancelled': 'Cancelled',
    'status.all': 'All Statuses',

    // Dashboard
    'dashboard.title': 'Reseller Dashboard',
    'dashboard.subtitle': 'Overview of your VPN reseller operations, sales, and infrastructure',
    'dashboard.activeServices': 'Active Services',
    'dashboard.expiredServices': 'Expired Services',
    'dashboard.totalCustomers': 'Total Customers',
    'dashboard.monthlyRevenue': 'Monthly Revenue',
    'dashboard.trafficTrend': 'Traffic Consumption Trend (GB)',
    'dashboard.revenueTrend': 'Weekly Revenue & Sales',
    'dashboard.statusDistribution': 'Service Status Distribution',
    'dashboard.quickActions': 'Quick Actions',

    // Services
    'services.title': 'Customer Services',
    'services.subtitle': 'Monitor and manage all VPN subscriptions under your brand',
    'services.search': 'Search client email, plan, or sub ID...',
    'services.email': 'Client Email / Identifier',
    'services.plan': 'Plan Name',
    'services.ipLimit': 'IP Limit',
    'services.traffic': 'Traffic Usage',
    'services.expiry': 'Expiry Date',
    'services.status': 'Status',
    'services.actions': 'Actions',
    'services.showQr': 'QR & Link',
    'services.resetTraffic': 'Reset Traffic',
    'services.rotateSub': 'Rotate SubId',
    'services.copyLink': 'Copy Subscription URL',
    'services.copied': 'Copied to clipboard!',
    'services.confirmReset': 'Are you sure you want to reset traffic usage for this service to zero?',
    'services.confirmRotate': 'Rotating SubId will invalidate the previous subscription link. Proceed?',

    // Orders
    'orders.title': 'Orders & Payment Approvals',
    'orders.subtitle': 'Review customer payment receipts, approve activations, or reject invalid proofs',
    'orders.id': 'Order ID',
    'orders.user': 'Customer TG ID',
    'orders.type': 'Type',
    'orders.amount': 'Amount',
    'orders.paymentMethod': 'Method',
    'orders.receipt': 'Receipt Proof',
    'orders.viewReceipt': 'View Receipt',
    'orders.approve': 'Approve',
    'orders.reject': 'Reject',
    'orders.rejectReason': 'Rejection Reason',
    'orders.rejectReasonPlaceholder': 'e.g. Invalid tracking code, amount mismatch...',
    'orders.confirmApprove': 'Approve this order and deliver the service?',
    'orders.noReceipt': 'No receipt text provided',

    // Child Bots
    'childBots.title': 'Child Customer Bots',
    'childBots.subtitle': 'Autonomous Telegram bots operated by you for your end customers',
    'childBots.newBot': 'Deploy New Child Bot',
    'childBots.botUsername': 'Bot Username',
    'childBots.token': 'API Token',
    'childBots.users': 'Active Users',
    'childBots.cardInfo': 'Payment Card Info',
    'childBots.lang': 'Default Language',
    'childBots.configured': 'Configured & Running',
    'childBots.addNotice': 'To deploy a new child bot, provide the Telegram Bot Father token. It will inherit your VPN node group settings and pricing policies.',

    // Tickets
    'tickets.title': 'Support Tickets',
    'tickets.subtitle': 'Resolve customer inquiries and technical support questions',
    'tickets.user': 'Customer ID',
    'tickets.subject': 'Subject',
    'tickets.lastUpdate': 'Last Update',
    'tickets.selectTicket': 'Select a ticket to view the conversation',
    'tickets.typeReply': 'Type your reply to the customer...',
    'tickets.sendReply': 'Send Reply',
    'tickets.open': 'Open',
    'tickets.answered': 'Answered',
    'tickets.closed': 'Closed',

    // Settings
    'settings.title': 'Reseller Account Settings',
    'settings.subtitle': 'Manage your brand details, security credentials, and view membership benefits',
    'settings.profile': 'Profile Information',
    'settings.resellerId': 'Telegram ID',
    'settings.username': 'Username',
    'settings.brandName': 'Service Brand Name',
    'settings.membership': 'Membership Tier & Quotas',
    'settings.changePassword': 'Change Password',
    'settings.currentPass': 'Current Password',
    'settings.newPass': 'New Password',
    'settings.confirmPass': 'Confirm New Password',
    'settings.save': 'Save Changes',
    'settings.tierUltimate': 'Ultimate Tier - Unlimited child bots, custom trial variants, priority routing',
    'settings.tierPro': 'Pro Tier - Up to 3 child bots, 25 trials/day',
    'settings.tierFree': 'Free Tier - Standard reseller capabilities, 10 trials/day',

    // Common
    'common.cancel': 'Cancel',
    'common.confirm': 'Confirm',
    'common.loading': 'Loading...',
    'common.success': 'Operation completed successfully',
    'common.error': 'An error occurred. Please try again.',
    'common.close': 'Close',
    'common.save': 'Save',
  },
  fa: {
    // Navigation
    'nav.dashboard': 'داشبورد',
    'nav.services': 'سرویس‌های کاربران',
    'nav.orders': 'سفارش‌ها و تایید پرداخت',
    'nav.childBots': 'ربات‌های فرزند',
    'nav.tickets': 'تیکت‌های پشتیبانی',
    'nav.settings': 'تنظیمات حساب',
    'nav.logout': 'خروج',

    // Header & User
    'header.wallet': 'موجودی کیف پول',
    'header.service': 'نام برند سرویس',
    'header.tier': 'سطح اشتراک',
    'header.switchLang': 'English',

    // Statuses
    'status.active': 'فعال',
    'status.expired': 'منقضی شده',
    'status.disabled': 'غیرفعال',
    'status.pending': 'در انتظار تایید',
    'status.approved': 'تایید شده',
    'status.rejected': 'رد شده',
    'status.completed': 'تکمیل شده',
    'status.cancelled': 'لغو شده',
    'status.all': 'همه وضعیت‌ها',

    // Dashboard
    'dashboard.title': 'داشبورد مدیریت نمایندگی',
    'dashboard.subtitle': 'بررسی اجمالی عملیات فروش، مصرف ترافیک و زیرساخت سرویس‌ها',
    'dashboard.activeServices': 'سرویس‌های فعال',
    'dashboard.expiredServices': 'سرویس‌های منقضی',
    'dashboard.totalCustomers': 'کل مشتریان',
    'dashboard.monthlyRevenue': 'درآمد ماهانه',
    'dashboard.trafficTrend': 'روند مصرف ترافیک (گیگابایت)',
    'dashboard.revenueTrend': 'فروش و درآمد هفتگی',
    'dashboard.statusDistribution': 'توزیع وضعیت سرویس‌ها',
    'dashboard.quickActions': 'عملیات سریع',

    // Services
    'services.title': 'مدیریت سرویس‌های کاربران',
    'services.subtitle': 'نظارت و مدیریت اشتراک‌های متصل به گروه و برند اختصاصی شما',
    'services.search': 'جستجوی ایمیل، نام پلن یا شناسه ساب...',
    'services.email': 'شناسه / ایمیل کاربر',
    'services.plan': 'نام پلن',
    'services.ipLimit': 'محدودیت کاربر (IP)',
    'services.traffic': 'مصرف ترافیک',
    'services.expiry': 'تاریخ انقضا',
    'services.status': 'وضعیت',
    'services.actions': 'عملیات',
    'services.showQr': 'QR و لینک ساب',
    'services.resetTraffic': 'ریست ترافیک',
    'services.rotateSub': 'تغییر شناسه ساب',
    'services.copyLink': 'کپی لینک اتصال ساب‌اسکریپشن',
    'services.copied': 'لینک با موفقیت کپی شد!',
    'services.confirmReset': 'آیا از صفر کردن ترافیک مصرفی این سرویس اطمینان دارید؟',
    'services.confirmRotate': 'تغییر شناسه ساب باعث ابطال لینک قبلی مشتری می‌شود. آیا ادامه می‌دهید؟',

    // Orders
    'orders.title': 'سفارش‌ها و تایید فیش واریزی',
    'orders.subtitle': 'بررسی فیش‌های کارت‌به‌کارت و پرداخت‌های کاربران جهت تایید یا رد',
    'orders.id': 'شماره سفارش',
    'orders.user': 'شناسه تلگرام کاربر',
    'orders.type': 'نوع سفارش',
    'orders.amount': 'مبلغ',
    'orders.paymentMethod': 'روش پرداخت',
    'orders.receipt': 'رسید پرداخت',
    'orders.viewReceipt': 'مشاهده رسید',
    'orders.approve': 'تایید سفارش',
    'orders.reject': 'رد درخواست',
    'orders.rejectReason': 'دلیل رد فیش',
    'orders.rejectReasonPlaceholder': 'مثلاً: کد پیگیری نامعتبر، عدم تطابق مبلغ واریزی...',
    'orders.confirmApprove': 'آیا سفارش تایید شده و سرویس تحویل گردد؟',
    'orders.noReceipt': 'توضیحات متنی ثبت نشده است',

    // Child Bots
    'childBots.title': 'ربات‌های فرزند (مشتریان)',
    'childBots.subtitle': 'ربات‌های اختصاصی تلگرام شما جهت فروش مستقیم به مشتریان',
    'childBots.newBot': 'ثبت ربات جدید',
    'childBots.botUsername': 'نام کاربری ربات',
    'childBots.token': 'توکن تلگرام',
    'childBots.users': 'کاربران فعال',
    'childBots.cardInfo': 'شماره کارت بانکی',
    'childBots.lang': 'زبان پیش‌فرض',
    'childBots.configured': 'پیکربندی شده و فعال',
    'childBots.addNotice': 'برای راه‌اندازی ربات فرزند، توکن دریافتی از BotFather را وارد نمایید. تنظیمات و سرورها از پنل شما به ارث می‌رسد.',

    // Tickets
    'tickets.title': 'تیکت‌ها و پشتیبانی کاربران',
    'tickets.subtitle': 'پاسخگویی به سوالات فنی و درخواست‌های مشتریان ربات‌های شما',
    'tickets.user': 'شناسه کاربر',
    'tickets.subject': 'موضوع',
    'tickets.lastUpdate': 'آخرین بروزرسانی',
    'tickets.selectTicket': 'یک تیکت را برای مشاهده مکالمه انتخاب کنید',
    'tickets.typeReply': 'پاسخ خود را برای مشتری تایپ کنید...',
    'tickets.sendReply': 'ارسال پاسخ',
    'tickets.open': 'باز',
    'tickets.answered': 'پاسخ داده شده',
    'tickets.closed': 'بسته شده',

    // Settings
    'settings.title': 'تنظیمات حساب نمایندگی',
    'settings.subtitle': 'مدیریت اطلاعات برند، تغییر رمز عبور و سهمیه‌های سطح اشتراک',
    'settings.profile': 'اطلاعات کاربری',
    'settings.resellerId': 'شناسه عددی تلگرام',
    'settings.username': 'نام کاربری',
    'settings.brandName': 'نام برند اختصاصی (سرویس)',
    'settings.membership': 'سطح اشتراک و مزایا',
    'settings.changePassword': 'تغییر کلمه عبور پنل',
    'settings.currentPass': 'کلمه عبور فعلی',
    'settings.newPass': 'کلمه عبور جدید',
    'settings.confirmPass': 'تکرار کلمه عبور جدید',
    'settings.save': 'ذخیره تغییرات',
    'settings.tierUltimate': 'سطح آلتیمیت (Ultimate) - ربات‌های فرزند نامحدود، سهمیه تست اختصاصی، اولویت بالا',
    'settings.tierPro': 'سطح حرفه‌ای (Pro) - تا ۳ ربات فرزند، ۲۵ اکانت تست در روز',
    'settings.tierFree': 'سطح پایه (Free) - امکانات عمومی نمایندگی، ۱۰ اکانت تست در روز',

    // Common
    'common.cancel': 'انصراف',
    'common.confirm': 'تایید',
    'common.loading': 'در حال بارگذاری...',
    'common.success': 'عملیات با موفقیت انجام شد',
    'common.error': 'خطایی رخ داد. لطفا مجددا تلاش کنید.',
    'common.close': 'بستن',
    'common.save': 'ذخیره',
  },
};

const LanguageContext = createContext<LanguageContextType>({
  lang: 'fa',
  setLang: () => {},
  t: (key: string) => key,
  isRTL: true,
});

export const LanguageProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [lang, setLangState] = useState<Language>(() => {
    const saved = localStorage.getItem('reseller_lang');
    return saved === 'en' || saved === 'fa' ? saved : 'fa';
  });

  useEffect(() => {
    localStorage.setItem('reseller_lang', lang);
    const dir = lang === 'fa' ? 'rtl' : 'ltr';
    document.documentElement.dir = dir;
    document.documentElement.lang = lang;
  }, [lang]);

  const setLang = (newLang: Language) => {
    setLangState(newLang);
  };

  const t = (key: string): string => {
    return translations[lang]?.[key] || translations['en']?.[key] || key;
  };

  const isRTL = lang === 'fa';

  return (
    <LanguageContext.Provider value={{ lang, setLang, t, isRTL }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = (): LanguageContextType => useContext(LanguageContext);
