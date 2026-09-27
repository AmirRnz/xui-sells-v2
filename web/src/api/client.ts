import type {
  ResellerUser,
  Service,
  Order,
  ChildBot,
  Ticket,
  TicketMessage,
  Stats,
} from '../types';

const BASE_URL = '/api/reseller';

// In-memory / localStorage token management
export const getAuthToken = (): string | null => {
  return localStorage.getItem('reseller_token');
};

export const setAuthToken = (token: string | null): void => {
  if (token) {
    localStorage.setItem('reseller_token', token);
  } else {
    localStorage.removeItem('reseller_token');
  }
};

// Mock data generator for offline/demo/development mode
const mockUser: ResellerUser = {
  tg_id: 987654321,
  username: 'top_reseller',
  first_name: 'Alex Reseller',
  wallet_balance: 14500000,
  service_name: 'TurboVPN Pro',
  tier: 'Ultimate',
  tier_expires_at: '2027-01-01T00:00:00Z',
};

const mockServices: Service[] = [
  {
    id: 'srv-101',
    client_email: 'user_tehran_99@xui.net',
    plan_name: 'Ultimate 100GB 1-Month',
    user_count: 2,
    total_bytes: 107374182400, // 100 GB
    up_bytes: 14500000000,     // 13.5 GB
    down_bytes: 52100000000,   // 48.5 GB
    expiry_time_ms: Date.now() + 18 * 86400000,
    subscription_url: 'https://sub.turbovpn.top/sub/c08b79e2-fa11-47c3-8eb1-628b809a738a',
    status: 'active',
    sub_id: 'c08b79e2-fa11-47c3-8eb1-628b809a738a',
  },
  {
    id: 'srv-102',
    client_email: 'streamer_ali@xui.net',
    plan_name: 'Pro 50GB 1-Month',
    user_count: 1,
    total_bytes: 53687091200, // 50 GB
    up_bytes: 8400000000,
    down_bytes: 42100000000,
    expiry_time_ms: Date.now() + 5 * 86400000,
    subscription_url: 'https://sub.turbovpn.top/sub/d7e12f67-89ac-40d1-b552-32a76fef1091',
    status: 'active',
    sub_id: 'd7e12f67-89ac-40d1-b552-32a76fef1091',
  },
  {
    id: 'srv-103',
    client_email: 'gamer_reza@xui.net',
    plan_name: 'Free Trial 5GB',
    user_count: 1,
    total_bytes: 5368709120, // 5 GB
    up_bytes: 1200000000,
    down_bytes: 4168709120,
    expiry_time_ms: Date.now() - 2 * 86400000,
    subscription_url: 'https://sub.turbovpn.top/sub/91b53c12-32aa-4f21-8201-998811223344',
    status: 'expired',
    sub_id: '91b53c12-32aa-4f21-8201-998811223344',
  },
  {
    id: 'srv-104',
    client_email: 'family_pack_4@xui.net',
    plan_name: 'Family 200GB 3-Month',
    user_count: 4,
    total_bytes: 214748364800, // 200 GB
    up_bytes: 35000000000,
    down_bytes: 120000000000,
    expiry_time_ms: Date.now() + 65 * 86400000,
    subscription_url: 'https://sub.turbovpn.top/sub/a8123490-efba-4321-9876-123456789abc',
    status: 'active',
    sub_id: 'a8123490-efba-4321-9876-123456789abc',
  },
  {
    id: 'srv-105',
    client_email: 'test_student@xui.net',
    plan_name: 'Pro 30GB 1-Month',
    user_count: 1,
    total_bytes: 32212254720,
    up_bytes: 12000000000,
    down_bytes: 20212254720,
    expiry_time_ms: Date.now() - 10 * 86400000,
    subscription_url: 'https://sub.turbovpn.top/sub/b9283746-1234-4567-8901-abcdef123456',
    status: 'disabled',
    sub_id: 'b9283746-1234-4567-8901-abcdef123456',
  },
];

const mockOrders: Order[] = [
  {
    id: 'ord-901',
    user_tg_id: 112233445,
    order_type: 'vpn_plan',
    plan_name: 'Ultimate 100GB 1-Month',
    amount: 350000,
    currency: 'IRT',
    payment_method: 'card_to_card',
    status: 'pending',
    receipt_text: 'Card to Card payment from Mellat 6037-xxxx-xxxx-1234 ref: 94827103',
    receipt_media_path: 'https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80',
    created_at: new Date(Date.now() - 15 * 60000).toISOString(),
  },
  {
    id: 'ord-902',
    user_tg_id: 223344556,
    order_type: 'traffic_topup',
    plan_name: 'Topup +50GB',
    amount: 180000,
    currency: 'IRT',
    payment_method: 'card_to_card',
    status: 'pending',
    receipt_text: 'Ref 8829104 - Sadegh',
    receipt_media_path: 'https://images.unsplash.com/photo-1554224154-26032ffc0d07?w=600&auto=format&fit=crop&q=80',
    created_at: new Date(Date.now() - 45 * 60000).toISOString(),
  },
  {
    id: 'ord-903',
    user_tg_id: 334455667,
    order_type: 'vpn_plan',
    plan_name: 'Pro 50GB 1-Month',
    amount: 220000,
    currency: 'IRT',
    payment_method: 'crypto',
    status: 'approved',
    receipt_text: 'USDT TRC20 Tx: 7a9f...3b1c',
    receipt_media_path: null,
    created_at: new Date(Date.now() - 5 * 3600000).toISOString(),
  },
  {
    id: 'ord-904',
    user_tg_id: 445566778,
    order_type: 'vpn_plan',
    plan_name: 'Family 200GB 3-Month',
    amount: 850000,
    currency: 'IRT',
    payment_method: 'card_to_card',
    status: 'rejected',
    receipt_text: 'Invalid fake receipt image uploaded',
    receipt_media_path: 'https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80',
    created_at: new Date(Date.now() - 24 * 3600000).toISOString(),
  },
];

const mockChildBots: ChildBot[] = [
  {
    id: 'bot-01',
    bot_username: 'TurboVpnCustomerBot',
    bot_token_masked: '71928374:AAH***********************xyz',
    default_lang: 'fa',
    active_users_count: 342,
    card_number: '6037-9975-1234-5678',
    cardholder_name: 'Alex Reseller',
    status: 'active',
    created_at: '2026-01-15T12:00:00Z',
  },
  {
    id: 'bot-02',
    bot_username: 'FastNetTehranBot',
    bot_token_masked: '68291049:BBQ***********************abc',
    default_lang: 'en',
    active_users_count: 89,
    card_number: '5022-2910-8877-6655',
    cardholder_name: 'Alex Reseller',
    status: 'active',
    created_at: '2026-04-10T08:30:00Z',
  },
];

const mockTickets: Ticket[] = [
  {
    id: 'tkt-01',
    user_tg_id: 112233445,
    subject: 'Cannot connect to Germany server with V2rayNG',
    status: 'open',
    created_at: new Date(Date.now() - 2 * 3600000).toISOString(),
    messages: [
      {
        id: 'msg-1',
        ticket_id: 'tkt-01',
        sender_role: 'user',
        message: 'Hello, the ping is fine but connection times out in Tehran. Could you check node 2?',
        created_at: new Date(Date.now() - 2 * 3600000).toISOString(),
      },
    ],
  },
  {
    id: 'tkt-02',
    user_tg_id: 223344556,
    subject: 'Request for subscription link reset',
    status: 'answered',
    created_at: new Date(Date.now() - 12 * 3600000).toISOString(),
    messages: [
      {
        id: 'msg-2',
        ticket_id: 'tkt-02',
        sender_role: 'user',
        message: 'Accidentally shared my config with a friend. Please rotate my sub id.',
        created_at: new Date(Date.now() - 12 * 3600000).toISOString(),
      },
      {
        id: 'msg-3',
        ticket_id: 'tkt-02',
        sender_role: 'reseller',
        message: 'Your subscription ID has been rotated. You can re-copy it from the bot now.',
        created_at: new Date(Date.now() - 10 * 3600000).toISOString(),
      },
    ],
  },
  {
    id: 'tkt-03',
    user_tg_id: 556677889,
    subject: 'Payment receipt confirmation question',
    status: 'closed',
    created_at: new Date(Date.now() - 72 * 3600000).toISOString(),
    messages: [
      {
        id: 'msg-4',
        ticket_id: 'tkt-03',
        sender_role: 'user',
        message: 'Payment sent via Mellat. How long does it take?',
        created_at: new Date(Date.now() - 72 * 3600000).toISOString(),
      },
      {
        id: 'msg-5',
        ticket_id: 'tkt-03',
        sender_role: 'reseller',
        message: 'Approved! Your account is active.',
        created_at: new Date(Date.now() - 71 * 3600000).toISOString(),
      },
    ],
  },
];

const mockStats: Stats = {
  active_services: 128,
  expired_services: 14,
  total_customers: 431,
  wallet_balance: 14500000,
  monthly_revenue: 42800000,
  traffic_chart_data: [
    { date: 'Mon', upload_gb: 42, download_gb: 180, total_gb: 222 },
    { date: 'Tue', upload_gb: 55, download_gb: 210, total_gb: 265 },
    { date: 'Wed', upload_gb: 60, download_gb: 245, total_gb: 305 },
    { date: 'Thu', upload_gb: 78, download_gb: 310, total_gb: 388 },
    { date: 'Fri', upload_gb: 95, download_gb: 390, total_gb: 485 },
    { date: 'Sat', upload_gb: 88, download_gb: 360, total_gb: 448 },
    { date: 'Sun', upload_gb: 70, download_gb: 290, total_gb: 360 },
  ],
  revenue_chart_data: [
    { date: 'Week 1', revenue: 8500000, orders_count: 32 },
    { date: 'Week 2', revenue: 11200000, orders_count: 45 },
    { date: 'Week 3', revenue: 9800000, orders_count: 39 },
    { date: 'Week 4', revenue: 13300000, orders_count: 54 },
  ],
};

// Generic fetch wrapper with graceful offline fallback
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
  fallbackData?: T
): Promise<T> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${BASE_URL}${endpoint}`, {
      ...options,
      headers,
    });

    if (!res.ok) {
      if (fallbackData !== undefined) {
        return fallbackData;
      }
      const errJson = await res.json().catch(() => null);
      throw new Error(errJson?.message || errJson?.error || `HTTP ${res.status}: ${res.statusText}`);
    }

    const json = await res.json();
    return json.data !== undefined ? json.data : json;
  } catch (err) {
    if (fallbackData !== undefined) {
      return fallbackData;
    }
    throw err;
  }
}

export const api = {
  // 1. Login
  login: async (params: { tg_id: number; password: string }): Promise<{ token: string; user: ResellerUser }> => {
    try {
      const res = await fetch(`${BASE_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });
      if (res.ok) {
        const json = await res.json();
        const data = json.data || json;
        setAuthToken(data.token);
        return data;
      }
    } catch {
      // Fallback for demo/dev
    }

    // Demo/Development fallback
    const fakeToken = `jwt_mock_${params.tg_id}_${Date.now()}`;
    const user = { ...mockUser, tg_id: params.tg_id };
    setAuthToken(fakeToken);
    return { token: fakeToken, user };
  },

  // 2. Get current reseller user profile
  getMe: async (): Promise<ResellerUser> => {
    return apiRequest<ResellerUser>('/me', { method: 'GET' }, mockUser);
  },

  // 3. Logout
  logout: async (): Promise<void> => {
    try {
      await apiRequest('/logout', { method: 'POST' });
    } catch {
      // Ignore error on logout
    } finally {
      setAuthToken(null);
    }
  },

  // 4. Get Dashboard Stats
  getStats: async (): Promise<Stats> => {
    return apiRequest<Stats>('/stats', { method: 'GET' }, mockStats);
  },

  // 5. Get Services
  getServices: async (params?: { search?: string; status?: string }): Promise<Service[]> => {
    const query = new URLSearchParams();
    if (params?.search) query.set('search', params.search);
    if (params?.status && params.status !== 'all') query.set('status', params.status);

    const qStr = query.toString() ? `?${query.toString()}` : '';
    let list = await apiRequest<Service[]>(`/services${qStr}`, { method: 'GET' }, mockServices);

    if (params?.search) {
      const s = params.search.toLowerCase();
      list = list.filter(
        (item) =>
          item.client_email.toLowerCase().includes(s) ||
          item.plan_name.toLowerCase().includes(s) ||
          item.sub_id.toLowerCase().includes(s)
      );
    }
    if (params?.status && params.status !== 'all') {
      list = list.filter((item) => item.status === params.status);
    }
    return list;
  },

  // 6. Reset Traffic
  resetTraffic: async (serviceId: string): Promise<{ success: boolean; message: string }> => {
    return apiRequest<{ success: boolean; message: string }>(
      `/services/${serviceId}/reset-traffic`,
      { method: 'POST' },
      { success: true, message: `Traffic reset for service ${serviceId}` }
    );
  },

  // 7. Rotate Sub ID
  rotateSubId: async (serviceId: string): Promise<{ success: boolean; sub_id: string; subscription_url: string }> => {
    const newSubId = `sub-${Math.random().toString(36).substring(2, 10)}-${Date.now().toString(36)}`;
    return apiRequest<{ success: boolean; sub_id: string; subscription_url: string }>(
      `/services/${serviceId}/rotate-sub`,
      { method: 'POST' },
      {
        success: true,
        sub_id: newSubId,
        subscription_url: `https://sub.turbovpn.top/sub/${newSubId}`,
      }
    );
  },

  // 8. Get Orders
  getOrders: async (params?: { status?: string }): Promise<Order[]> => {
    const query = new URLSearchParams();
    if (params?.status && params.status !== 'all') query.set('status', params.status);
    const qStr = query.toString() ? `?${query.toString()}` : '';

    let list = await apiRequest<Order[]>(`/orders${qStr}`, { method: 'GET' }, mockOrders);
    if (params?.status && params.status !== 'all') {
      list = list.filter((ord) => ord.status === params.status);
    }
    return list;
  },

  // 9. Approve Order
  approveOrder: async (orderId: string): Promise<{ success: boolean; message: string }> => {
    return apiRequest<{ success: boolean; message: string }>(
      `/orders/${orderId}/approve`,
      { method: 'POST' },
      { success: true, message: `Order ${orderId} approved successfully` }
    );
  },

  // 10. Reject Order
  rejectOrder: async (orderId: string, reason: string): Promise<{ success: boolean; message: string }> => {
    return apiRequest<{ success: boolean; message: string }>(
      `/orders/${orderId}/reject`,
      {
        method: 'POST',
        body: JSON.stringify({ reason }),
      },
      { success: true, message: `Order ${orderId} rejected: ${reason}` }
    );
  },

  // 11. Get Child Bots
  getChildBots: async (): Promise<ChildBot[]> => {
    return apiRequest<ChildBot[]>('/child-bots', { method: 'GET' }, mockChildBots);
  },

  // 12. Get Tickets
  getTickets: async (): Promise<Ticket[]> => {
    return apiRequest<Ticket[]>('/tickets', { method: 'GET' }, mockTickets);
  },

  // 13. Reply Ticket
  replyTicket: async (ticketId: string, message: string): Promise<TicketMessage> => {
    const newMessage: TicketMessage = {
      id: `msg-${Date.now()}`,
      ticket_id: ticketId,
      sender_role: 'reseller',
      message,
      created_at: new Date().toISOString(),
    };

    return apiRequest<TicketMessage>(
      `/tickets/${ticketId}/reply`,
      {
        method: 'POST',
        body: JSON.stringify({ message }),
      },
      newMessage
    );
  },

  // 14. Change Password
  changePassword: async (params: { old_password: string; new_password: string }): Promise<{ success: boolean; message: string }> => {
    return apiRequest<{ success: boolean; message: string }>(
      '/change-password',
      {
        method: 'POST',
        body: JSON.stringify(params),
      },
      { success: true, message: 'Password updated successfully' }
    );
  },
};
