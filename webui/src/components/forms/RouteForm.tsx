/**
 * Route Form Component
 *
 * Specialized form for routes with grouped fields and better UX.
 */

import React, { useState, useCallback } from 'react';
import type { ModuleSchema, Record } from '@/types/schema';
import { FieldRenderer } from '@/components/fields/FieldRenderer';
import { MethodsField } from '@/components/fields/MethodsField';
import { HeadersField } from '@/components/fields/HeadersField';
import { TransformField } from '@/components/fields/TransformField';

interface RouteFormProps {
  module: ModuleSchema;
  initialData?: Record;
  mode: 'create' | 'edit';
  onSubmit: (data: Record) => Promise<void>;
  onCancel?: () => void;
  isLoading?: boolean;
}

interface FieldError {
  [key: string]: string;
}

// Field groupings for route form
const FIELD_GROUPS = {
  basic: {
    title: 'Basic Information',
    description: 'Route name and description',
    fields: ['name', 'description', 'enabled'],
    defaultOpen: true,
  },
  matching: {
    title: 'Request Matching',
    description: 'How incoming requests are matched to this route',
    fields: ['path_pattern', 'match_type', 'methods', 'headers'],
    defaultOpen: true,
  },
  target: {
    title: 'Target & Rewriting',
    description: 'Where to forward requests and how to transform paths',
    fields: ['upstream_id', 'path_rewrite', 'method_override', 'protocol'],
    defaultOpen: true,
  },
  transforms: {
    title: 'Request/Response Transforms',
    description: 'Transform headers, query params, or body',
    fields: ['request_transform', 'response_transform'],
    defaultOpen: false,
  },
  metering: {
    title: 'Metering & Priority',
    description: 'Usage tracking and route priority',
    fields: ['metering_expr', 'metering_mode', 'priority'],
    defaultOpen: false,
  },
};

export function RouteForm({
  module,
  initialData = {},
  mode,
  onSubmit,
  onCancel,
  isLoading,
}: RouteFormProps) {
  // Track which sections are open
  const [openSections, setOpenSections] = useState<Set<string>>(() => {
    const initial = new Set<string>();
    Object.entries(FIELD_GROUPS).forEach(([key, group]) => {
      if (group.defaultOpen) initial.add(key);
    });
    return initial;
  });

  // Initialize form data
  const [formData, setFormData] = useState<Record>(() => {
    if (mode === 'edit') return initialData;
    const defaults: Record = { ...initialData };
    for (const field of module.fields) {
      if (field.default !== undefined && defaults[field.name] === undefined) {
        defaults[field.name] = field.default;
      }
    }
    return defaults;
  });

  const [errors, setErrors] = useState<FieldError>({});
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Toggle section
  const toggleSection = (section: string) => {
    setOpenSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  };

  // Handle field change
  const handleChange = useCallback((name: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [name]: value }));
    if (errors[name]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[name];
        return next;
      });
    }
  }, [errors]);

  // Validate form
  const validate = useCallback((): boolean => {
    const newErrors: FieldError = {};
    const requiredFields = ['name', 'path_pattern', 'upstream_id'];

    for (const fieldName of requiredFields) {
      const value = formData[fieldName];
      if (value === undefined || value === null || value === '') {
        newErrors[fieldName] = `${formatLabel(fieldName)} is required`;
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [formData]);

  // Handle submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitError(null);

    if (!validate()) return;

    try {
      await onSubmit(formData);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'An error occurred');
    }
  };

  // Get field by name
  const getField = (name: string) => module.fields.find((f) => f.name === name);

  // Render a specialized field
  const renderField = (fieldName: string) => {
    const field = getField(fieldName);
    if (!field) return null;

    // Skip fields in edit mode that are immutable
    if (mode === 'edit' && field.immutable) return null;

    const value = formData[fieldName];
    const error = errors[fieldName];

    // Specialized renderers
    switch (fieldName) {
      case 'methods':
        return (
          <div key={fieldName} className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              HTTP Methods
            </label>
            <MethodsField
              value={value as string[] | null}
              onChange={(v) => handleChange(fieldName, v)}
              disabled={isLoading}
              error={error}
            />
            <p className="mt-1 text-xs text-gray-500">{field.description}</p>
          </div>
        );

      case 'headers':
        return (
          <div key={fieldName} className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Header Conditions
            </label>
            <HeadersField
              value={value as any}
              onChange={(v) => handleChange(fieldName, v)}
              disabled={isLoading}
              error={error}
            />
            <p className="mt-1 text-xs text-gray-500">{field.description}</p>
          </div>
        );

      case 'request_transform':
        return (
          <div key={fieldName} className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Request Transform
            </label>
            <TransformField
              value={value as any}
              onChange={(v) => handleChange(fieldName, v)}
              disabled={isLoading}
              error={error}
              type="request"
            />
            <p className="mt-1 text-xs text-gray-500">{field.description}</p>
          </div>
        );

      case 'response_transform':
        return (
          <div key={fieldName} className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Response Transform
            </label>
            <TransformField
              value={value as any}
              onChange={(v) => handleChange(fieldName, v)}
              disabled={isLoading}
              error={error}
              type="response"
            />
            <p className="mt-1 text-xs text-gray-500">{field.description}</p>
          </div>
        );

      default:
        return (
          <FieldRenderer
            key={fieldName}
            field={field}
            module={module}
            value={value}
            onChange={(v) => handleChange(fieldName, v)}
            error={error}
            disabled={isLoading}
          />
        );
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Grouped sections */}
      {Object.entries(FIELD_GROUPS).map(([key, group]) => {
        const isOpen = openSections.has(key);
        const hasErrors = group.fields.some((f) => errors[f]);

        return (
          <div
            key={key}
            className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden"
          >
            {/* Section header */}
            <button
              type="button"
              onClick={() => toggleSection(key)}
              className="w-full px-4 py-3 flex items-center justify-between bg-gray-50 hover:bg-gray-100 transition-colors"
            >
              <div className="flex items-center gap-3">
                <svg
                  className={`w-5 h-5 text-gray-500 transition-transform ${isOpen ? 'rotate-90' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
                <div className="text-left">
                  <h3 className="text-sm font-semibold text-gray-900">{group.title}</h3>
                  <p className="text-xs text-gray-500">{group.description}</p>
                </div>
              </div>
              {hasErrors && (
                <span className="px-2 py-0.5 text-xs font-medium bg-red-100 text-red-700 rounded-full">
                  Has errors
                </span>
              )}
            </button>

            {/* Section content */}
            {isOpen && (
              <div className="px-4 py-4 border-t border-gray-200">
                {group.fields.map((fieldName) => renderField(fieldName))}
              </div>
            )}
          </div>
        );
      })}

      {/* Error message */}
      {submitError && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p className="text-sm text-red-700">{submitError}</p>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isLoading}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
          >
            Cancel
          </button>
        )}
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50 flex items-center gap-2"
        >
          {isLoading && (
            <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          )}
          {mode === 'create' ? 'Create Route' : 'Save Changes'}
        </button>
      </div>
    </form>
  );
}

function formatLabel(name: string): string {
  return name
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}
