/**
 * Methods Field Component
 *
 * Renders HTTP methods as checkboxes instead of raw JSON.
 */


const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const;

interface MethodsFieldProps {
  value: string[] | null | undefined;
  onChange: (value: string[]) => void;
  disabled?: boolean;
  error?: string;
}

export function MethodsField({ value, onChange, disabled, error }: MethodsFieldProps) {
  const selectedMethods = Array.isArray(value) ? value : [];

  const toggleMethod = (method: string) => {
    if (selectedMethods.includes(method)) {
      onChange(selectedMethods.filter((m) => m !== method));
    } else {
      onChange([...selectedMethods, method]);
    }
  };

  const selectAll = () => onChange([...HTTP_METHODS]);
  const selectNone = () => onChange([]);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 mb-2">
        <button
          type="button"
          onClick={selectAll}
          disabled={disabled}
          className="text-xs text-primary-600 hover:text-primary-700 disabled:opacity-50"
        >
          Select all
        </button>
        <span className="text-gray-300">|</span>
        <button
          type="button"
          onClick={selectNone}
          disabled={disabled}
          className="text-xs text-primary-600 hover:text-primary-700 disabled:opacity-50"
        >
          Clear
        </button>
        {selectedMethods.length === 0 && (
          <span className="text-xs text-gray-500 ml-2">(empty = all methods)</span>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        {HTTP_METHODS.map((method) => {
          const isSelected = selectedMethods.includes(method);
          const methodColors: Record<string, string> = {
            GET: 'bg-green-100 border-green-300 text-green-700',
            POST: 'bg-blue-100 border-blue-300 text-blue-700',
            PUT: 'bg-yellow-100 border-yellow-300 text-yellow-700',
            PATCH: 'bg-orange-100 border-orange-300 text-orange-700',
            DELETE: 'bg-red-100 border-red-300 text-red-700',
            HEAD: 'bg-purple-100 border-purple-300 text-purple-700',
            OPTIONS: 'bg-gray-100 border-gray-300 text-gray-700',
          };

          return (
            <label
              key={method}
              className={`
                inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md border cursor-pointer
                transition-all text-sm font-medium
                ${disabled ? 'opacity-50 cursor-not-allowed' : 'hover:shadow-sm'}
                ${isSelected ? methodColors[method] : 'bg-gray-50 border-gray-200 text-gray-400'}
              `}
            >
              <input
                type="checkbox"
                checked={isSelected}
                onChange={() => toggleMethod(method)}
                disabled={disabled}
                className="sr-only"
              />
              {method}
            </label>
          );
        })}
      </div>
      {error && <p className="text-xs text-red-500 mt-1">{error}</p>}
    </div>
  );
}
