/**
 * Analytics API client for fetching usage statistics.
 * Consumes /admin/usage and related endpoints.
 */

export interface UsageSummary {
  total_requests: number;
  total_users: number;
  total_keys: number;
  requests_today: number;
  requests_week: number;
  requests_month: number;
}

export interface UserUsageSummary {
  user_id: string;
  email: string;
  plan_id: string;
  requests: number;
  bytes_in: number;
  bytes_out: number;
  last_request_at?: string;
}

export interface PlanUsageSummary {
  plan_id: string;
  plan_name: string;
  user_count: number;
  requests: number;
}

export interface UsageResponse {
  period: string;
  start_date: string;
  end_date: string;
  summary: UsageSummary;
  by_user?: UserUsageSummary[];
  by_plan?: PlanUsageSummary[];
}

export interface AnalyticsEvent {
  id: string;
  timestamp: string;
  channel: string;
  module: string;
  action: string;
  record_id?: string;
  user_id?: string;
  api_key_id?: string;
  remote_ip?: string;
  duration_ns: number;
  memory_bytes: number;
  request_bytes: number;
  response_bytes: number;
  success: boolean;
  status_code?: number;
  error?: string;
}

export interface AnalyticsSummary {
  channel?: string;
  module?: string;
  action?: string;
  period: string;
  start: string;
  end: string;
  total_requests: number;
  success_requests: number;
  error_requests: number;
  avg_duration_ns: number;
  min_duration_ns: number;
  max_duration_ns: number;
  p50_duration_ns: number;
  p95_duration_ns: number;
  p99_duration_ns: number;
  total_memory_bytes: number;
  total_request_bytes: number;
  total_response_bytes: number;
  cost_units: number;
}

/**
 * Fetch usage statistics.
 */
export async function fetchUsage(params?: {
  period?: 'day' | 'week' | 'month';
  user_id?: string;
  start_date?: string;
  end_date?: string;
}): Promise<UsageResponse> {
  const searchParams = new URLSearchParams();
  if (params?.period) searchParams.set('period', params.period);
  if (params?.user_id) searchParams.set('user_id', params.user_id);
  if (params?.start_date) searchParams.set('start_date', params.start_date);
  if (params?.end_date) searchParams.set('end_date', params.end_date);

  const url = `/admin/usage${searchParams.toString() ? `?${searchParams}` : ''}`;
  const response = await fetch(url, { credentials: 'include' });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'Failed to fetch usage');
  }

  return response.json();
}

/**
 * Fetch recent analytics events.
 */
export async function fetchRecentEvents(params?: {
  limit?: number;
  module?: string;
  action?: string;
  user_id?: string;
}): Promise<{ data: AnalyticsEvent[]; count: number }> {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set('limit', String(params.limit));
  if (params?.module) searchParams.set('module', params.module);
  if (params?.action) searchParams.set('action', params.action);
  if (params?.user_id) searchParams.set('user_id', params.user_id);

  const url = `/admin/analytics/events${searchParams.toString() ? `?${searchParams}` : ''}`;
  const response = await fetch(url, { credentials: 'include' });

  if (!response.ok) {
    // Fall back to empty if endpoint doesn't exist
    if (response.status === 404) {
      return { data: [], count: 0 };
    }
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'Failed to fetch events');
  }

  return response.json();
}

/**
 * Fetch aggregated analytics.
 */
export async function fetchAnalyticsSummary(params?: {
  period?: 'minute' | 'hour' | 'day';
  group_by?: string[];
  start?: string;
  end?: string;
}): Promise<AnalyticsSummary[]> {
  const searchParams = new URLSearchParams();
  if (params?.period) searchParams.set('period', params.period);
  if (params?.group_by) searchParams.set('group_by', params.group_by.join(','));
  if (params?.start) searchParams.set('start', params.start);
  if (params?.end) searchParams.set('end', params.end);

  const url = `/admin/analytics/summary${searchParams.toString() ? `?${searchParams}` : ''}`;
  const response = await fetch(url, { credentials: 'include' });

  if (!response.ok) {
    // Fall back to empty if endpoint doesn't exist
    if (response.status === 404) {
      return [];
    }
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'Failed to fetch summary');
  }

  return response.json();
}

/**
 * Format bytes to human-readable string.
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

/**
 * Format nanoseconds to human-readable duration.
 */
export function formatDuration(ns: number): string {
  if (ns < 1000) return `${ns}ns`;
  if (ns < 1000000) return `${(ns / 1000).toFixed(2)}μs`;
  if (ns < 1000000000) return `${(ns / 1000000).toFixed(2)}ms`;
  return `${(ns / 1000000000).toFixed(2)}s`;
}

/**
 * Format a number with thousands separators.
 */
export function formatNumber(num: number): string {
  return num.toLocaleString();
}
