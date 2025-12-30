/**
 * Field Renderer Component
 *
 * Dynamically renders the appropriate input widget based on field type.
 * Integrates with DocumentationContext for contextual help.
 */

import React, { forwardRef } from 'react';
import type { FieldSchema, ModuleSchema } from '@/types/schema';
import { useFieldDocumentation } from '@/context/DocumentationContext';
import { RefField } from './RefField';

interface FieldRendererProps {
  field: FieldSchema;
  module: ModuleSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  error?: string;
  disabled?: boolean;
}

export const FieldRenderer = forwardRef<HTMLInputElement, FieldRendererProps>(
  function FieldRenderer({ field, module, value, onChange, error, disabled }, ref) {
    const docHandlers = useFieldDocumentation(field, module);

    const baseClasses = `
      w-full px-3 py-2 border rounded-lg text-sm
      focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
      transition-colors
      ${error ? 'border-red-300 bg-red-50' : 'border-gray-300 bg-white'}
      ${disabled ? 'bg-gray-100 text-gray-500 cursor-not-allowed' : ''}
    `;

    const label = (
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {formatLabel(field.name)}
        {field.required && <span className="text-red-500 ml-1">*</span>}
        {field.computed && <span className="text-gray-400 ml-1 text-xs">(auto)</span>}
      </label>
    );

    const errorMessage = error && (
      <p className="mt-1 text-xs text-red-500">{error}</p>
    );

    const helpText = field.description && (
      <p className="mt-1 text-xs text-gray-500">{field.description}</p>
    );

    // Render based on field type
    switch (field.type) {
      case 'bool':
        return (
          <div className="mb-4" {...docHandlers}>
            <div className="flex items-center gap-2">
              <input
                ref={ref as React.Ref<HTMLInputElement>}
                type="checkbox"
                checked={Boolean(value)}
                onChange={(e) => onChange(e.target.checked)}
                disabled={disabled || field.computed}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
              <label className="text-sm font-medium text-gray-700">
                {formatLabel(field.name)}
                {field.required && <span className="text-red-500 ml-1">*</span>}
              </label>
            </div>
            {helpText}
            {errorMessage}
          </div>
        );

      case 'enum':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <select
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
            >
              <option value="">Select {formatLabel(field.name)}...</option>
              {field.values?.map((opt) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
            {helpText}
            {errorMessage}
          </div>
        );

      case 'text':
      case 'json':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <textarea
              value={field.type === 'json' ? formatJSON(value) : String(value ?? '')}
              onChange={(e) => onChange(field.type === 'json' ? parseJSON(e.target.value) : e.target.value)}
              disabled={disabled || field.computed}
              rows={4}
              className={baseClasses}
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'secret':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="password"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
              autoComplete="new-password"
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'email':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="email"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
              placeholder="user@example.com"
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'url':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="url"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
              placeholder="https://example.com"
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'int':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="number"
              value={value !== null && value !== undefined ? String(value) : ''}
              onChange={(e) => onChange(e.target.value ? parseInt(e.target.value, 10) : null)}
              disabled={disabled || field.computed}
              className={baseClasses}
              step="1"
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'float':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="number"
              value={value !== null && value !== undefined ? String(value) : ''}
              onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
              disabled={disabled || field.computed}
              className={baseClasses}
              step="any"
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'datetime':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="datetime-local"
              value={formatDatetime(value)}
              onChange={(e) => onChange(e.target.value ? new Date(e.target.value).toISOString() : null)}
              disabled={disabled || field.computed}
              className={baseClasses}
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'date':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="date"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'time':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="time"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'ref':
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <RefField
              targetModule={field.ref || ''}
              value={value as string | undefined}
              onChange={(val) => onChange(val)}
              disabled={disabled || field.computed}
              hasError={!!error}
              placeholder={`Select ${formatLabel(field.ref || 'reference')}...`}
            />
            {helpText}
            {errorMessage}
          </div>
        );

      case 'string':
      default:
        return (
          <div className="mb-4" {...docHandlers}>
            {label}
            <input
              ref={ref}
              type="text"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              disabled={disabled || field.computed}
              className={baseClasses}
            />
            {helpText}
            {errorMessage}
          </div>
        );
    }
  }
);

// Helper functions

function formatLabel(name: string): string {
  return name
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function formatJSON(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  return JSON.stringify(value, null, 2);
}

function parseJSON(value: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

function formatDatetime(value: unknown): string {
  if (!value) return '';
  try {
    const date = new Date(String(value));
    return date.toISOString().slice(0, 16);
  } catch {
    return '';
  }
}
