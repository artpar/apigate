/**
 * Reference Field Component
 *
 * Renders a select dropdown for foreign key references.
 * Fetches options from the referenced module.
 */

import { useQuery } from '@tanstack/react-query';
import { fetchRecords, fetchModules } from '@/api/schema';
import type { Record as RecordType } from '@/types/schema';

interface RefFieldProps {
  /** Target module name (singular, e.g., "plan") */
  targetModule: string;
  /** Current value (ID of referenced record) */
  value: string | undefined | null;
  /** Called when selection changes */
  onChange: (value: string | null) => void;
  /** Field is disabled */
  disabled?: boolean;
  /** CSS classes for the select element */
  className?: string;
  /** Placeholder text */
  placeholder?: string;
  /** Error state */
  hasError?: boolean;
}

/**
 * Determines the display label for a record.
 * Tries common fields: name, title, email, label, then falls back to ID.
 */
function getDisplayLabel(record: RecordType): string {
  const displayFields = ['name', 'title', 'email', 'label', 'slug'];
  for (const field of displayFields) {
    if (record[field] && typeof record[field] === 'string') {
      return record[field] as string;
    }
  }
  return String(record.id || 'Unknown');
}

export function RefField({
  targetModule,
  value,
  onChange,
  disabled,
  className = '',
  placeholder,
  hasError,
}: RefFieldProps) {
  // Get modules to find the plural form
  const { data: modules } = useQuery({
    queryKey: ['modules'],
    queryFn: fetchModules,
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
  });

  // Find plural form for API call
  const modulePlural = modules?.find((m) => m.module === targetModule)?.plural || `${targetModule}s`;

  // Fetch records from target module
  const {
    data: recordsResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['records', modulePlural],
    queryFn: () => fetchRecords(modulePlural, { limit: 100 }),
    enabled: !!modulePlural,
    staleTime: 30 * 1000, // Cache for 30 seconds
  });

  const records = recordsResponse?.data || [];

  // Build select classes
  const selectClasses = `
    w-full px-3 py-2 border rounded-lg text-sm
    focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
    transition-colors
    ${hasError ? 'border-red-300 bg-red-50' : 'border-gray-300 bg-white'}
    ${disabled ? 'bg-gray-100 text-gray-500 cursor-not-allowed' : ''}
    ${className}
  `.trim();

  if (isLoading) {
    return (
      <div className="relative">
        <select disabled className={selectClasses}>
          <option>Loading {targetModule}s...</option>
        </select>
        <div className="absolute right-8 top-1/2 -translate-y-1/2">
          <svg className="animate-spin w-4 h-4 text-gray-400" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500">
        Failed to load {targetModule}s: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    );
  }

  if (records.length === 0) {
    return (
      <div className="text-sm text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2">
        No {targetModule}s available. Please create a {targetModule} first.
      </div>
    );
  }

  return (
    <select
      value={value || ''}
      onChange={(e) => onChange(e.target.value || null)}
      disabled={disabled}
      className={selectClasses}
    >
      <option value="">{placeholder || `Select ${targetModule}...`}</option>
      {records.map((record) => (
        <option key={String(record.id)} value={String(record.id)}>
          {getDisplayLabel(record)}
          {record.id !== getDisplayLabel(record) && ` (${record.id})`}
        </option>
      ))}
    </select>
  );
}
