/**
 * Dynamic Table Component
 *
 * Renders a data table based on module schema.
 * Supports sorting, pagination, and row actions.
 */

import React, { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import type { ModuleSchema, FieldSchema, Record } from '@/types/schema';
import { useDocumentation } from '@/context/DocumentationContext';

interface DynamicTableProps {
  module: ModuleSchema;
  records: Record[];
  isLoading?: boolean;
  onDelete?: (id: string) => void;
  onAction?: (action: string, id: string) => void;
}

export function DynamicTable({
  module,
  records,
  isLoading,
  onDelete,
  onAction,
}: DynamicTableProps) {
  const { focusModule, focusField } = useDocumentation();
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

  // Get visible columns (non-internal, non-secret)
  const columns = useMemo(() => {
    return module.fields.filter(
      (f) => !f.internal && f.type !== 'secret' && f.type !== 'bytes'
    );
  }, [module.fields]);

  // Get custom actions
  const customActions = useMemo(() => {
    return module.actions.filter((a) => a.type === 'custom');
  }, [module.actions]);

  // Sort records
  const sortedRecords = useMemo(() => {
    if (!sortField) return records;

    return [...records].sort((a, b) => {
      const aVal = a[sortField];
      const bVal = b[sortField];

      if (aVal === bVal) return 0;
      if (aVal === null || aVal === undefined) return 1;
      if (bVal === null || bVal === undefined) return -1;

      const comparison = String(aVal).localeCompare(String(bVal));
      return sortDir === 'asc' ? comparison : -comparison;
    });
  }, [records, sortField, sortDir]);

  // Handle column header click for sorting
  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortField(field);
      setSortDir('asc');
    }
  };

  // Focus module on mount
  React.useEffect(() => {
    focusModule(module);
  }, [module, focusModule]);

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8">
        <div className="flex items-center justify-center gap-3 text-gray-500">
          <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          Loading...
        </div>
      </div>
    );
  }

  if (records.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center">
        <svg className="w-12 h-12 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
        <h3 className="text-lg font-medium text-gray-900 mb-2">No {module.plural} found</h3>
        <p className="text-gray-500 mb-4">Get started by creating a new {module.module}.</p>
        <Link
          to={`/${module.plural}/new`}
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create {module.module}
        </Link>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.name}
                  onClick={() => handleSort(col.name)}
                  onMouseEnter={() => focusField(col, module)}
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    {formatLabel(col.name)}
                    {sortField === col.name && (
                      <svg className={`w-4 h-4 ${sortDir === 'desc' ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                      </svg>
                    )}
                  </div>
                </th>
              ))}
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {sortedRecords.map((record) => (
              <tr key={String(record.id)} className="hover:bg-gray-50 transition-colors">
                {columns.map((col) => (
                  <td key={col.name} className="px-4 py-3 text-sm text-gray-900 whitespace-nowrap">
                    {formatValue(record[col.name], col)}
                  </td>
                ))}
                <td className="px-4 py-3 text-right text-sm whitespace-nowrap">
                  <div className="flex items-center justify-end gap-2">
                    {/* View/Edit */}
                    <Link
                      to={`/${module.plural}/${record.id}`}
                      className="text-primary-600 hover:text-primary-800"
                      title="View"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    </Link>

                    {/* Custom actions */}
                    {customActions.map((action) => (
                      <button
                        key={action.name}
                        onClick={() => onAction?.(action.name, String(record.id))}
                        className="text-gray-400 hover:text-gray-600"
                        title={action.name}
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                        </svg>
                      </button>
                    ))}

                    {/* Delete */}
                    {onDelete && (
                      <button
                        onClick={() => onDelete(String(record.id))}
                        className="text-red-400 hover:text-red-600"
                        title="Delete"
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination placeholder */}
      <div className="px-4 py-3 border-t border-gray-200 bg-gray-50 flex items-center justify-between">
        <div className="text-sm text-gray-500">
          Showing {records.length} {module.plural}
        </div>
      </div>
    </div>
  );
}

// Helper functions

function formatLabel(name: string): string {
  return name
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function formatValue(value: unknown, field: FieldSchema): React.ReactNode {
  if (value === null || value === undefined) {
    return <span className="text-gray-400">-</span>;
  }

  switch (field.type) {
    case 'bool':
      return (
        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
          value ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
        }`}>
          {value ? 'Yes' : 'No'}
        </span>
      );

    case 'enum':
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800">
          {String(value)}
        </span>
      );

    case 'datetime':
      return new Date(String(value)).toLocaleString();

    case 'date':
      return new Date(String(value)).toLocaleDateString();

    case 'email':
      return (
        <a href={`mailto:${value}`} className="text-primary-600 hover:underline">
          {String(value)}
        </a>
      );

    case 'url':
      return (
        <a href={String(value)} target="_blank" rel="noopener noreferrer" className="text-primary-600 hover:underline">
          {truncate(String(value), 30)}
        </a>
      );

    case 'json':
      return (
        <code className="text-xs bg-gray-100 px-1 rounded">
          {truncate(JSON.stringify(value), 30)}
        </code>
      );

    case 'ref':
      return (
        <span className="text-gray-600">
          {truncate(String(value), 20)}
        </span>
      );

    default:
      return truncate(String(value), 40);
  }
}

function truncate(str: string, length: number): string {
  if (str.length <= length) return str;
  return str.slice(0, length) + '...';
}
