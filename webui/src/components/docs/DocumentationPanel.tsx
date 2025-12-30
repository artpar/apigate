/**
 * Documentation Panel Component
 *
 * Right-side panel that displays contextual documentation
 * based on what's currently focused (module, field, or action).
 *
 * Tabs:
 * - Spec: Field type, constraints, validation rules
 * - Details: Description, help text
 * - API: REST endpoints, curl examples
 * - Examples: Sample data, common use cases
 */

import React, { useState, useMemo } from 'react';
import { useDocumentation } from '@/context/DocumentationContext';
import {
  formatFieldType,
  getFieldValidation,
  generateCurl,
  generateSampleData,
  generateAPIExamples,
} from '@/utils/docs';
import type { FieldSchema, ActionSchema, ModuleSchema } from '@/types/schema';

type TabId = 'spec' | 'details' | 'api' | 'examples';

interface Tab {
  id: TabId;
  label: string;
}

const tabs: Tab[] = [
  { id: 'spec', label: 'Spec' },
  { id: 'details', label: 'Details' },
  { id: 'api', label: 'API' },
  { id: 'examples', label: 'Examples' },
];

export function DocumentationPanel() {
  const { focus, isExpanded, toggleExpanded, cancelClearFocus } = useDocumentation();
  const [activeTab, setActiveTab] = useState<TabId>('spec');

  if (!isExpanded) {
    return (
      <button
        onClick={toggleExpanded}
        className="fixed right-0 top-1/2 -translate-y-1/2 bg-primary-600 text-white p-2 rounded-l-lg shadow-lg hover:bg-primary-700 transition-colors"
        title="Show Documentation"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      </button>
    );
  }

  return (
    <div
      className="w-80 bg-white border-l border-gray-200 flex flex-col h-full"
      onMouseEnter={cancelClearFocus}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-gray-50">
        <h2 className="text-sm font-semibold text-gray-700">Documentation</h2>
        <button
          onClick={toggleExpanded}
          className="text-gray-400 hover:text-gray-600 transition-colors"
          title="Hide Documentation"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex-1 px-3 py-2 text-xs font-medium transition-colors ${
              activeTab === tab.id
                ? 'text-primary-600 border-b-2 border-primary-600 bg-primary-50'
                : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {focus.type === 'none' && <DefaultContent />}
        {focus.type === 'module' && focus.module && (
          <ModuleContent module={focus.module} tab={activeTab} />
        )}
        {focus.type === 'field' && focus.field && focus.module && (
          <FieldContent field={focus.field} module={focus.module} tab={activeTab} />
        )}
        {focus.type === 'action' && focus.action && focus.module && (
          <ActionContent action={focus.action} module={focus.module} tab={activeTab} />
        )}
      </div>
    </div>
  );
}

function DefaultContent() {
  return (
    <div className="text-center text-gray-500 py-8">
      <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p className="text-sm">
        Hover over or focus on a field to see its documentation.
      </p>
    </div>
  );
}

function ModuleContent({ module, tab }: { module: ModuleSchema; tab: TabId }) {
  const examples = useMemo(() => generateAPIExamples(module), [module]);

  switch (tab) {
    case 'spec':
      return (
        <div className="space-y-4">
          <Section title="Module">
            <p className="text-sm font-medium text-gray-900">{module.module}</p>
            <p className="text-xs text-gray-500">Plural: {module.plural}</p>
          </Section>
          <Section title="Fields">
            <ul className="space-y-1">
              {module.fields.filter(f => !f.internal).map(f => (
                <li key={f.name} className="text-xs">
                  <code className="text-primary-600">{f.name}</code>
                  <span className="text-gray-500"> : {formatFieldType(f)}</span>
                  {f.required && <span className="text-red-500 ml-1">*</span>}
                </li>
              ))}
            </ul>
          </Section>
          <Section title="Lookups">
            <p className="text-xs text-gray-600">
              Records can be looked up by: {module.lookups.join(', ')}
            </p>
          </Section>
        </div>
      );

    case 'details':
      return (
        <div className="space-y-4">
          <Section title="Description">
            <p className="text-sm text-gray-700">{module.description || 'No description available.'}</p>
          </Section>
          <Section title="Actions">
            <ul className="space-y-2">
              {module.actions.map(a => (
                <li key={a.name} className="text-xs">
                  <span className="font-medium text-gray-900">{a.name}</span>
                  <p className="text-gray-500">{a.description}</p>
                </li>
              ))}
            </ul>
          </Section>
        </div>
      );

    case 'api':
      return (
        <div className="space-y-4">
          <Section title="Endpoints">
            {module.actions.map(a => (
              <div key={a.name} className="mb-3">
                <div className="flex items-center gap-2 mb-1">
                  <MethodBadge method={a.http.method} />
                  <code className="text-xs text-gray-700">{a.http.path.replace('{module}', module.plural)}</code>
                </div>
                <p className="text-xs text-gray-500">{a.description}</p>
              </div>
            ))}
          </Section>
        </div>
      );

    case 'examples':
      return (
        <div className="space-y-4">
          {examples.slice(0, 3).map((ex, i) => (
            <Section key={i} title={ex.title}>
              <CodeBlock code={ex.curl} />
            </Section>
          ))}
        </div>
      );

    default:
      return null;
  }
}

function FieldContent({ field, module: _module, tab }: { field: FieldSchema; module: ModuleSchema; tab: TabId }) {
  const validation = useMemo(() => getFieldValidation(field), [field]);
  void _module; // Used for type consistency with other content components

  switch (tab) {
    case 'spec':
      return (
        <div className="space-y-4">
          <Section title="Field">
            <p className="text-sm font-medium text-gray-900">{field.name}</p>
          </Section>
          <Section title="Type">
            <code className="text-sm text-primary-600">{formatFieldType(field)}</code>
          </Section>
          {field.values && field.values.length > 0 && (
            <Section title="Allowed Values">
              <ul className="space-y-1">
                {field.values.map(v => (
                  <li key={v} className="text-xs">
                    <code className="bg-gray-100 px-1 rounded">{v}</code>
                  </li>
                ))}
              </ul>
            </Section>
          )}
          <Section title="Attributes">
            <ul className="space-y-1 text-xs">
              <li>Required: <span className={field.required ? 'text-red-600' : 'text-gray-500'}>{field.required ? 'Yes' : 'No'}</span></li>
              {field.unique && <li>Unique: <span className="text-amber-600">Yes</span></li>}
              {field.immutable && <li>Immutable: <span className="text-amber-600">Yes</span></li>}
              {field.computed && <li>Computed: <span className="text-blue-600">Yes (auto-generated)</span></li>}
            </ul>
          </Section>
        </div>
      );

    case 'details':
      return (
        <div className="space-y-4">
          <Section title="Description">
            <p className="text-sm text-gray-700">{field.description || 'No description available.'}</p>
          </Section>
          {validation.length > 0 && (
            <Section title="Validation Rules">
              <ul className="space-y-1">
                {validation.map((rule, i) => (
                  <li key={i} className="text-xs text-gray-600 flex items-start gap-2">
                    <span className="text-primary-500 mt-0.5">*</span>
                    {rule}
                  </li>
                ))}
              </ul>
            </Section>
          )}
          {field.ref && (
            <Section title="Reference">
              <p className="text-xs text-gray-600">
                References a record in the <code className="text-primary-600">{field.ref}</code> module.
              </p>
            </Section>
          )}
        </div>
      );

    case 'api':
      return (
        <div className="space-y-4">
          <Section title="JSON Path">
            <code className="text-sm text-gray-700">$.data.{field.name}</code>
          </Section>
          <Section title="In Request Body">
            <CodeBlock code={JSON.stringify({ [field.name]: generateSampleValue(field) }, null, 2)} />
          </Section>
        </div>
      );

    case 'examples':
      return (
        <div className="space-y-4">
          <Section title="Example Values">
            <CodeBlock code={String(generateSampleValue(field))} />
          </Section>
          {field.type === 'enum' && field.values && (
            <Section title="Valid Options">
              <ul className="space-y-1">
                {field.values.map(v => (
                  <li key={v} className="text-xs">
                    <code className="bg-gray-100 px-2 py-0.5 rounded">{v}</code>
                  </li>
                ))}
              </ul>
            </Section>
          )}
        </div>
      );

    default:
      return null;
  }
}

function generateSampleValue(field: FieldSchema): unknown {
  if (field.default !== undefined) return field.default;
  switch (field.type) {
    case 'string': return `sample_${field.name}`;
    case 'int': return 100;
    case 'float': return 99.99;
    case 'bool': return true;
    case 'email': return 'user@example.com';
    case 'url': return 'https://example.com';
    case 'enum': return field.values?.[0] ?? 'option1';
    case 'ref': return `${field.ref}_id`;
    default: return null;
  }
}

function ActionContent({ action, module, tab }: { action: ActionSchema; module: ModuleSchema; tab: TabId }) {
  const sampleData = useMemo(
    () => action.type === 'create' || action.type === 'update'
      ? generateSampleData(module, action.type)
      : undefined,
    [action, module]
  );
  const curl = useMemo(() => generateCurl(action, module, sampleData), [action, module, sampleData]);

  switch (tab) {
    case 'spec':
      return (
        <div className="space-y-4">
          <Section title="Action">
            <p className="text-sm font-medium text-gray-900">{action.name}</p>
          </Section>
          <Section title="Type">
            <span className="inline-block px-2 py-0.5 bg-primary-100 text-primary-700 text-xs rounded">
              {action.type}
            </span>
          </Section>
          <Section title="HTTP">
            <div className="flex items-center gap-2">
              <MethodBadge method={action.http.method} />
              <code className="text-xs text-gray-700">{action.http.path.replace('{module}', module.plural)}</code>
            </div>
          </Section>
          {action.auth && action.auth !== 'none' && (
            <Section title="Authentication">
              <p className="text-xs text-gray-600">Requires: {action.auth}</p>
            </Section>
          )}
          {action.confirm && (
            <Section title="Confirmation">
              <p className="text-xs text-amber-600">This action requires confirmation</p>
            </Section>
          )}
        </div>
      );

    case 'details':
      return (
        <div className="space-y-4">
          <Section title="Description">
            <p className="text-sm text-gray-700">{action.description}</p>
          </Section>
          {action.input.length > 0 && (
            <Section title="Input Fields">
              <ul className="space-y-2">
                {action.input.map(input => (
                  <li key={input.name} className="text-xs">
                    <code className="text-primary-600">{input.name}</code>
                    <span className="text-gray-500"> : {input.type}</span>
                    {input.required && <span className="text-red-500 ml-1">*</span>}
                    {input.description && <p className="text-gray-500 mt-0.5">{input.description}</p>}
                  </li>
                ))}
              </ul>
            </Section>
          )}
        </div>
      );

    case 'api':
      return (
        <div className="space-y-4">
          <Section title="Curl Command">
            <CodeBlock code={curl} />
          </Section>
        </div>
      );

    case 'examples':
      return (
        <div className="space-y-4">
          {sampleData && (
            <Section title="Request Body">
              <CodeBlock code={JSON.stringify(sampleData, null, 2)} />
            </Section>
          )}
          <Section title="Response">
            <CodeBlock code={JSON.stringify({
              module: module.module,
              data: action.type === 'delete' ? null : generateSampleData(module),
            }, null, 2)} />
          </Section>
        </div>
      );

    default:
      return null;
  }
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">{title}</h3>
      {children}
    </div>
  );
}

function MethodBadge({ method }: { method: string }) {
  const colors: Record<string, string> = {
    GET: 'bg-green-100 text-green-700',
    POST: 'bg-blue-100 text-blue-700',
    PUT: 'bg-amber-100 text-amber-700',
    PATCH: 'bg-orange-100 text-orange-700',
    DELETE: 'bg-red-100 text-red-700',
  };

  return (
    <span className={`inline-block px-1.5 py-0.5 text-xs font-mono font-medium rounded ${colors[method] || 'bg-gray-100 text-gray-700'}`}>
      {method}
    </span>
  );
}

function CodeBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <pre className="bg-gray-900 text-gray-100 text-xs p-3 rounded-lg overflow-x-auto">
        <code>{code}</code>
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 p-1 bg-gray-700 rounded opacity-0 group-hover:opacity-100 transition-opacity"
        title="Copy to clipboard"
      >
        {copied ? (
          <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        ) : (
          <svg className="w-4 h-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        )}
      </button>
    </div>
  );
}
