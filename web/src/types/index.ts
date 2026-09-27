export type TierType = 'free' | 'pro' | 'ultimate' | 'Free' | 'Pro' | 'Ultimate';

export interface ResellerUser {
  tg_id: number;
  username: string;
  first_name: string;
  wallet_balance: number;
  service_name: string;
  tier: TierType;
  tier_expires_at: string | null;
}

export type ServiceStatus = 'active' | 'expired' | 'disabled' | 'pending';

export interface Service {
  id: string;
  client_email: string;
  plan_name: string;
  user_count: number;
  total_bytes: number;
  up_bytes: number;
  down_bytes: number;
  expiry_time_ms: number;
  subscription_url: string;
  status: ServiceStatus;
  sub_id: string;
}

export type OrderType = 'vpn_plan' | 'wallet_topup' | 'tier_membership' | 'traffic_topup';
export type OrderStatus = 'pending' | 'approved' | 'rejected' | 'completed' | 'cancelled';

export interface Order {
  id: string;
  user_tg_id: number;
  order_type: OrderType;
  plan_name: string;
  amount: number;
  currency: string;
  payment_method: string;
  status: OrderStatus;
  receipt_text: string | null;
  receipt_media_path: string | null;
  created_at: string;
}

export type BotStatus = 'active' | 'stopped' | 'suspended';

export interface ChildBot {
  id: string;
  bot_username: string;
  bot_token_masked: string;
  default_lang: 'fa' | 'en';
  active_users_count: number;
  card_number: string;
  cardholder_name: string;
  status: BotStatus;
  created_at: string;
}

export interface TicketMessage {
  id: string;
  ticket_id: string;
  sender_role: 'user' | 'reseller' | 'support' | 'system';
  message: string;
  created_at: string;
}

export type TicketStatus = 'open' | 'answered' | 'closed';

export interface Ticket {
  id: string;
  user_tg_id: number;
  subject: string;
  status: TicketStatus;
  created_at: string;
  messages: TicketMessage[];
}

export interface TrafficChartPoint {
  date: string;
  upload_gb: number;
  download_gb: number;
  total_gb: number;
}

export interface RevenueChartPoint {
  date: string;
  revenue: number;
  orders_count: number;
}

export interface Stats {
  active_services: number;
  expired_services: number;
  total_customers: number;
  wallet_balance: number;
  monthly_revenue: number;
  traffic_chart_data: TrafficChartPoint[];
  revenue_chart_data: RevenueChartPoint[];
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}
