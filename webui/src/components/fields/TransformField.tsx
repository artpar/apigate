/**
 * Transform Field Component
 *
 * Renders request/response transforms with a more intuitive UI.
 */

import { useState } from 'react';

interface Transform {
  set_headers?: Record<string, string>;
  delete_headers?: string[];
  body_expr?: string;
  set_query?: Record<string, string>;
  delete_query?: string[];
}

interface TransformFieldProps {
  value: Transform | null | undefined;
  onChange: (value: Transform) => void;
  disabled?: boolean;
  error?: string;
  type: 'request' | 'response';
}

type TransformSection = 'headers' | 'query' | 'body';

export function TransformField({ value, onChange, disabled, error, type }: TransformFieldProps) {
  const [activeSection, setActiveSection] = useState<TransformSection | null>(null);
  const transform: Transform = value || {};

  const updateTransform = (updates: Partial<Transform>) => {
    onChange({ ...transform, ...updates });
  };

  // Headers
  const setHeaders = transform.set_headers || {};
  const deleteHeaders = transform.delete_headers || [];

  const addSetHeader = () => {
    updateTransform({ set_headers: { ...setHeaders, '': '' } });
  };

  const updateSetHeader = (oldKey: string, newKey: string, newValue: string) => {
    const newHeaders = { ...setHeaders };
    if (oldKey !== newKey) delete newHeaders[oldKey];
    newHeaders[newKey] = newValue;
    updateTransform({ set_headers: newHeaders });
  };

  const removeSetHeader = (key: string) => {
    const newHeaders = { ...setHeaders };
    delete newHeaders[key];
    updateTransform({ set_headers: newHeaders });
  };

  const addDeleteHeader = () => {
    updateTransform({ delete_headers: [...deleteHeaders, ''] });
  };

  const updateDeleteHeader = (index: number, value: string) => {
    const newHeaders = [...deleteHeaders];
    newHeaders[index] = value;
    updateTransform({ delete_headers: newHeaders });
  };

  const removeDeleteHeader = (index: number) => {
    updateTransform({ delete_headers: deleteHeaders.filter((_, i) => i !== index) });
  };

  // Query params (request only)
  const setQuery = transform.set_query || {};
  const deleteQuery = transform.delete_query || [];

  const hasContent =
    Object.keys(setHeaders).length > 0 ||
    deleteHeaders.length > 0 ||
    Object.keys(setQuery).length > 0 ||
    deleteQuery.length > 0 ||
    !!transform.body_expr;

  const sections: { id: TransformSection; label: string; available: boolean }[] = [
    { id: 'headers', label: 'Headers', available: true },
    { id: 'query', label: 'Query Params', available: type === 'request' },
    { id: 'body', label: 'Body', available: true },
  ];

  return (
    <div className="space-y-3">
      {!hasContent && !activeSection && (
        <p className="text-sm text-gray-500 italic">No transforms configured</p>
      )}

      {/* Section tabs */}
      <div className="flex flex-wrap gap-2">
        {sections.filter(s => s.available).map((section) => (
          <button
            key={section.id}
            type="button"
            onClick={() => setActiveSection(activeSection === section.id ? null : section.id)}
            disabled={disabled}
            className={`
              px-3 py-1.5 text-sm rounded-md border transition-colors
              ${activeSection === section.id
                ? 'bg-primary-50 border-primary-300 text-primary-700'
                : 'bg-white border-gray-200 text-gray-600 hover:border-gray-300'}
              disabled:opacity-50
            `}
          >
            {section.label}
            {section.id === 'headers' && (Object.keys(setHeaders).length > 0 || deleteHeaders.length > 0) && (
              <span className="ml-1 text-xs bg-primary-100 text-primary-600 px-1.5 rounded-full">
                {Object.keys(setHeaders).length + deleteHeaders.length}
              </span>
            )}
            {section.id === 'query' && (Object.keys(setQuery).length > 0 || deleteQuery.length > 0) && (
              <span className="ml-1 text-xs bg-primary-100 text-primary-600 px-1.5 rounded-full">
                {Object.keys(setQuery).length + deleteQuery.length}
              </span>
            )}
            {section.id === 'body' && transform.body_expr && (
              <span className="ml-1 text-xs bg-primary-100 text-primary-600 px-1.5 rounded-full">1</span>
            )}
          </button>
        ))}
      </div>

      {/* Headers section */}
      {activeSection === 'headers' && (
        <div className="p-4 bg-gray-50 rounded-lg border border-gray-200 space-y-4">
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">Set Headers</h4>
            {Object.entries(setHeaders).map(([key, val], index) => (
              <div key={index} className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={key}
                  onChange={(e) => updateSetHeader(key, e.target.value, val)}
                  placeholder="Header-Name"
                  disabled={disabled}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded"
                />
                <span className="text-gray-400">=</span>
                <input
                  type="text"
                  value={val}
                  onChange={(e) => updateSetHeader(key, key, e.target.value)}
                  placeholder="value or {{expr}}"
                  disabled={disabled}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded"
                />
                <button type="button" onClick={() => removeSetHeader(key)} className="text-red-500">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={addSetHeader}
              disabled={disabled}
              className="text-sm text-primary-600 hover:text-primary-700"
            >
              + Add header to set
            </button>
          </div>

          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">Delete Headers</h4>
            {deleteHeaders.map((header, index) => (
              <div key={index} className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={header}
                  onChange={(e) => updateDeleteHeader(index, e.target.value)}
                  placeholder="Header-Name"
                  disabled={disabled}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded"
                />
                <button type="button" onClick={() => removeDeleteHeader(index)} className="text-red-500">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={addDeleteHeader}
              disabled={disabled}
              className="text-sm text-primary-600 hover:text-primary-700"
            >
              + Add header to delete
            </button>
          </div>
        </div>
      )}

      {/* Query section (request only) */}
      {activeSection === 'query' && type === 'request' && (
        <div className="p-4 bg-gray-50 rounded-lg border border-gray-200 space-y-4">
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">Set Query Parameters</h4>
            {Object.entries(setQuery).map(([key, val], index) => (
              <div key={index} className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={key}
                  onChange={(e) => {
                    const newQuery = { ...setQuery };
                    delete newQuery[key];
                    newQuery[e.target.value] = val;
                    updateTransform({ set_query: newQuery });
                  }}
                  placeholder="param"
                  disabled={disabled}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded"
                />
                <span className="text-gray-400">=</span>
                <input
                  type="text"
                  value={val}
                  onChange={(e) => {
                    updateTransform({ set_query: { ...setQuery, [key]: e.target.value } });
                  }}
                  placeholder="value"
                  disabled={disabled}
                  className="flex-1 px-2 py-1.5 text-sm border border-gray-300 rounded"
                />
                <button
                  type="button"
                  onClick={() => {
                    const newQuery = { ...setQuery };
                    delete newQuery[key];
                    updateTransform({ set_query: newQuery });
                  }}
                  className="text-red-500"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => updateTransform({ set_query: { ...setQuery, '': '' } })}
              disabled={disabled}
              className="text-sm text-primary-600 hover:text-primary-700"
            >
              + Add query parameter
            </button>
          </div>
        </div>
      )}

      {/* Body section */}
      {activeSection === 'body' && (
        <div className="p-4 bg-gray-50 rounded-lg border border-gray-200">
          <h4 className="text-sm font-medium text-gray-700 mb-2">Body Expression</h4>
          <textarea
            value={transform.body_expr || ''}
            onChange={(e) => updateTransform({ body_expr: e.target.value })}
            placeholder="Expression to transform body, e.g.: { ...body, extra: 'value' }"
            disabled={disabled}
            rows={3}
            className="w-full px-3 py-2 text-sm border border-gray-300 rounded font-mono"
          />
          <p className="text-xs text-gray-500 mt-1">
            Use expressions to transform the {type} body. Available: body, headers, path, query
          </p>
        </div>
      )}

      {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
    </div>
  );
}
