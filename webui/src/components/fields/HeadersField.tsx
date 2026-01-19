/**
 * Headers Field Component
 *
 * Renders header matching conditions as a key-value editor.
 */


interface HeaderCondition {
  name: string;
  value: string;
  isRegex?: boolean;
  required?: boolean;
}

interface HeadersFieldProps {
  value: HeaderCondition[] | null | undefined;
  onChange: (value: HeaderCondition[]) => void;
  disabled?: boolean;
  error?: string;
}

export function HeadersField({ value, onChange, disabled, error }: HeadersFieldProps) {
  const headers = Array.isArray(value) ? value : [];

  const addHeader = () => {
    onChange([...headers, { name: '', value: '', isRegex: false, required: false }]);
  };

  const updateHeader = (index: number, updates: Partial<HeaderCondition>) => {
    const newHeaders = [...headers];
    newHeaders[index] = { ...newHeaders[index], ...updates };
    onChange(newHeaders);
  };

  const removeHeader = (index: number) => {
    onChange(headers.filter((_, i) => i !== index));
  };

  return (
    <div className="space-y-3">
      {headers.length === 0 ? (
        <p className="text-sm text-gray-500 italic">No header conditions (route matches all requests)</p>
      ) : (
        <div className="space-y-2">
          {headers.map((header, index) => (
            <div key={index} className="flex items-start gap-2 p-3 bg-gray-50 rounded-lg border border-gray-200">
              <div className="flex-1 grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">Header Name</label>
                  <input
                    type="text"
                    value={header.name}
                    onChange={(e) => updateHeader(index, { name: e.target.value })}
                    disabled={disabled}
                    placeholder="X-Custom-Header"
                    className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">Value</label>
                  <input
                    type="text"
                    value={header.value}
                    onChange={(e) => updateHeader(index, { value: e.target.value })}
                    disabled={disabled}
                    placeholder="expected-value"
                    className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>
              </div>
              <div className="flex flex-col gap-1 pt-5">
                <label className="inline-flex items-center gap-1 text-xs text-gray-600">
                  <input
                    type="checkbox"
                    checked={header.isRegex || false}
                    onChange={(e) => updateHeader(index, { isRegex: e.target.checked })}
                    disabled={disabled}
                    className="w-3 h-3"
                  />
                  Regex
                </label>
                <label className="inline-flex items-center gap-1 text-xs text-gray-600">
                  <input
                    type="checkbox"
                    checked={header.required || false}
                    onChange={(e) => updateHeader(index, { required: e.target.checked })}
                    disabled={disabled}
                    className="w-3 h-3"
                  />
                  Required
                </label>
              </div>
              <button
                type="button"
                onClick={() => removeHeader(index)}
                disabled={disabled}
                className="p-1 text-gray-400 hover:text-red-500 disabled:opacity-50"
                title="Remove header"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        onClick={addHeader}
        disabled={disabled}
        className="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-primary-600 hover:text-primary-700 hover:bg-primary-50 rounded-md disabled:opacity-50"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add Header Condition
      </button>
      {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
    </div>
  );
}
