/**
 * Usage Dashboard Page
 *
 * Displays usage analytics and statistics.
 * Shows request counts, user activity, and plan distribution.
 */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  fetchUsage,
  formatBytes,
  formatNumber,
  type UsageResponse,
  type UserUsageSummary,
  type PlanUsageSummary,
} from '@/api/analytics';

type Period = 'day' | 'week' | 'month';

export function UsageDashboard() {
  const [period, setPeriod] = useState<Period>('month');

  const { data, isLoading, error } = useQuery({
    queryKey: ['usage', period],
    queryFn: () => fetchUsage({ period }),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="flex items-center gap-3 text-gray-500">
          <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          Loading analytics...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6">
        <h3 className="text-lg font-medium text-red-800">Failed to load analytics</h3>
        <p className="text-red-600 mt-1">{error instanceof Error ? error.message : 'Unknown error'}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Usage Analytics</h1>
          <p className="text-gray-500">Monitor API usage and activity</p>
        </div>
        <div className="flex items-center gap-2">
          <PeriodSelector value={period} onChange={setPeriod} />
        </div>
      </div>

      {/* Summary Cards */}
      {data && <SummaryCards data={data} />}

      {/* Charts/Details */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Usage by User */}
        {data?.by_user && data.by_user.length > 0 && (
          <UserUsageTable users={data.by_user} />
        )}

        {/* Usage by Plan */}
        {data?.by_plan && data.by_plan.length > 0 && (
          <PlanUsageTable plans={data.by_plan} />
        )}
      </div>

      {/* Empty state */}
      {data && (!data.by_user || data.by_user.length === 0) && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center">
          <svg className="w-12 h-12 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No usage data</h3>
          <p className="text-gray-500">
            Usage data will appear here once API requests are made.
          </p>
        </div>
      )}
    </div>
  );
}

// Period selector component
function PeriodSelector({ value, onChange }: { value: Period; onChange: (v: Period) => void }) {
  const periods: { value: Period; label: string }[] = [
    { value: 'day', label: 'Last 24 Hours' },
    { value: 'week', label: 'Last 7 Days' },
    { value: 'month', label: 'Last 30 Days' },
  ];

  return (
    <div className="flex rounded-lg border border-gray-200 bg-white p-1">
      {periods.map((p) => (
        <button
          key={p.value}
          onClick={() => onChange(p.value)}
          className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
            value === p.value
              ? 'bg-primary-100 text-primary-700'
              : 'text-gray-600 hover:text-gray-900'
          }`}
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}

// Summary cards component
function SummaryCards({ data }: { data: UsageResponse }) {
  const cards = [
    {
      label: 'Total Requests',
      value: formatNumber(data.summary.total_requests),
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
        </svg>
      ),
      color: 'text-primary-600 bg-primary-100',
    },
    {
      label: 'Active Users',
      value: formatNumber(data.summary.total_users),
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      ),
      color: 'text-green-600 bg-green-100',
    },
    {
      label: 'API Keys',
      value: formatNumber(data.summary.total_keys),
      icon: (
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
        </svg>
      ),
      color: 'text-orange-600 bg-orange-100',
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {cards.map((card) => (
        <div key={card.label} className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center gap-4">
            <div className={`p-3 rounded-lg ${card.color}`}>
              {card.icon}
            </div>
            <div>
              <p className="text-sm text-gray-500">{card.label}</p>
              <p className="text-2xl font-bold text-gray-900">{card.value}</p>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// User usage table component
function UserUsageTable({ users }: { users: UserUsageSummary[] }) {
  // Sort by requests descending
  const sortedUsers = [...users].sort((a, b) => b.requests - a.requests);
  const topUsers = sortedUsers.slice(0, 10);

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-semibold text-gray-900">Top Users by Requests</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Requests</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Data In</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Data Out</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {topUsers.map((user) => (
              <tr key={user.user_id} className="hover:bg-gray-50">
                <td className="px-6 py-4">
                  <Link to={`/users/${user.user_id}`} className="text-primary-600 hover:underline">
                    {user.email}
                  </Link>
                  <p className="text-xs text-gray-500">{user.plan_id}</p>
                </td>
                <td className="px-6 py-4 text-right font-medium">
                  {formatNumber(user.requests)}
                </td>
                <td className="px-6 py-4 text-right text-gray-600">
                  {formatBytes(user.bytes_in)}
                </td>
                <td className="px-6 py-4 text-right text-gray-600">
                  {formatBytes(user.bytes_out)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {users.length > 10 && (
        <div className="px-6 py-3 border-t border-gray-200 bg-gray-50 text-center">
          <span className="text-sm text-gray-500">
            Showing top 10 of {users.length} users
          </span>
        </div>
      )}
    </div>
  );
}

// Plan usage table component
function PlanUsageTable({ plans }: { plans: PlanUsageSummary[] }) {
  // Sort by requests descending
  const sortedPlans = [...plans].sort((a, b) => b.requests - a.requests);

  // Calculate total for percentages
  const totalRequests = sortedPlans.reduce((sum, p) => sum + p.requests, 0);

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-semibold text-gray-900">Usage by Plan</h3>
      </div>
      <div className="p-6 space-y-4">
        {sortedPlans.map((plan) => {
          const percentage = totalRequests > 0 ? (plan.requests / totalRequests) * 100 : 0;
          return (
            <div key={plan.plan_id}>
              <div className="flex items-center justify-between mb-1">
                <div>
                  <Link to={`/plans/${plan.plan_id}`} className="font-medium text-gray-900 hover:text-primary-600">
                    {plan.plan_name}
                  </Link>
                  <span className="text-sm text-gray-500 ml-2">
                    ({plan.user_count} users)
                  </span>
                </div>
                <span className="text-sm font-medium text-gray-600">
                  {formatNumber(plan.requests)} requests
                </span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-primary-600 h-2 rounded-full transition-all"
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
