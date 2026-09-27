import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function formatBytes(bytes: number, decimals: number = 2): string {
  if (!bytes || bytes <= 0) return '0 B';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const safeI = Math.min(i, sizes.length - 1);

  return `${parseFloat((bytes / Math.pow(k, safeI)).toFixed(dm))} ${sizes[safeI]}`;
}

export function formatCurrency(amount: number, currency: string = 'IRT', locale: string = 'en'): string {
  if (amount === undefined || amount === null || isNaN(amount)) {
    return '0';
  }

  const formattedNumber = new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : 'en-US').format(amount);

  const upperCurr = currency ? currency.toUpperCase() : 'IRT';
  if (upperCurr === 'IRT' || upperCurr === 'TOMAN' || upperCurr === 'تومان') {
    return locale === 'fa' ? `${formattedNumber} تومان` : `${formattedNumber} Toman`;
  }
  if (upperCurr === 'USD' || upperCurr === '$') {
    return `$${formattedNumber}`;
  }
  if (upperCurr === 'IRR' || upperCurr === 'RIAL' || upperCurr === 'ریال') {
    return locale === 'fa' ? `${formattedNumber} ریال` : `${formattedNumber} IRR`;
  }

  return `${formattedNumber} ${currency}`;
}

export function formatDate(timestamp: number | string | null | undefined, locale: string = 'en'): string {
  if (!timestamp) return '-';
  const date = typeof timestamp === 'number' ? new Date(timestamp) : new Date(timestamp);
  if (isNaN(date.getTime())) return String(timestamp);

  return new Intl.DateTimeFormat(locale === 'fa' ? 'fa-IR' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}
