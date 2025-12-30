/**
 * Module View Page
 *
 * View and edit a single record.
 * Also handles create mode for new records.
 */

import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  fetchModuleSchema,
  fetchRecord,
  createRecord,
  updateRecord,
} from '@/api/schema';
import { DynamicForm } from '@/components/DynamicForm';
import { PasswordModal } from '@/components/PasswordModal';
import type { Record, RecordResponse } from '@/types/schema';

export function ModuleView() {
  const { module, id } = useParams<{ module: string; id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isCreate = id === 'new';

  // State for showing raw key after API key creation
  const [createdKey, setCreatedKey] = useState<{rawKey: string; recordId: string} | null>(null);

  // State for password modal (user module only)
  const [showPasswordModal, setShowPasswordModal] = useState(false);

  // Fetch schema
  const { data: schema, isLoading: schemaLoading } = useQuery({
    queryKey: ['schema', module],
    queryFn: () => fetchModuleSchema(module!),
    enabled: !!module,
  });

  // Fetch record (only in edit mode, uses schema.plural for API path)
  const { data: recordData, isLoading: recordLoading } = useQuery({
    queryKey: ['record', module, id],
    queryFn: () => fetchRecord(schema!.plural, id!),
    enabled: !!module && !!id && !isCreate && !!schema,
  });

  // Create mutation (uses schema.plural for API path)
  const createMutation = useMutation({
    mutationFn: (data: Record) => createRecord(schema!.plural, data),
    onSuccess: (result: RecordResponse<Record>) => {
      queryClient.invalidateQueries({ queryKey: ['records', module] });
      // Check if we have a raw key (API key creation)
      if (result.meta?.raw_key) {
        setCreatedKey({
          rawKey: result.meta.raw_key,
          recordId: result.data.id as string,
        });
        // Don't navigate yet - let user copy the key first
      } else {
        navigate(`/${module}/${result.data.id}`);
      }
    },
  });

  // Update mutation (uses schema.plural for API path)
  const updateMutation = useMutation({
    mutationFn: (data: Record) => updateRecord(schema!.plural, id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['records', module] });
      queryClient.invalidateQueries({ queryKey: ['record', module, id] });
    },
  });

  const handleSubmit = async (data: Record) => {
    if (isCreate) {
      await createMutation.mutateAsync(data);
    } else {
      await updateMutation.mutateAsync(data);
    }
  };

  const handleCancel = () => {
    navigate(`/${module}`);
  };

  if (schemaLoading || (!isCreate && recordLoading)) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="flex items-center gap-3 text-gray-500">
          <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          Loading...
        </div>
      </div>
    );
  }

  if (!schema) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <h3 className="text-lg font-medium text-red-800">Module not found</h3>
        <p className="text-red-600">The module "{module}" does not exist.</p>
      </div>
    );
  }

  // Check if this is a user record (for password management)
  const isUserModule = module === 'user';
  const userEmail = recordData?.data?.email as string | undefined;

  return (
    <div className="space-y-6">
      {/* Header with breadcrumb and actions */}
      <div className="flex items-center justify-between">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-2 text-sm">
          <Link to={`/${module}`} className="text-gray-500 hover:text-gray-700 capitalize">
            {schema.plural}
          </Link>
          <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          <span className="text-gray-900 font-medium">
            {isCreate ? 'Create New' : recordData?.data?.id || id}
          </span>
        </nav>

        {/* Actions */}
        {isUserModule && !isCreate && (
          <button
            type="button"
            onClick={() => setShowPasswordModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg text-sm font-medium transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
            Set Password
          </button>
        )}
      </div>

      {/* Form */}
      <DynamicForm
        module={schema}
        initialData={isCreate ? {} : recordData?.data || {}}
        mode={isCreate ? 'create' : 'edit'}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />

      {/* Success message */}
      {updateMutation.isSuccess && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            <p className="text-sm text-green-700">Changes saved successfully!</p>
          </div>
        </div>
      )}

      {/* API Key Creation Modal */}
      {createdKey && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="flex-shrink-0 w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
                <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">API Key Created</h3>
                <p className="text-sm text-gray-500">Copy this key now. You won't be able to see it again!</p>
              </div>
            </div>

            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">Your API Key</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-white border border-gray-300 rounded px-3 py-2 text-sm font-mono break-all select-all">
                  {createdKey.rawKey}
                </code>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(createdKey.rawKey);
                  }}
                  className="flex-shrink-0 px-3 py-2 bg-gray-100 hover:bg-gray-200 rounded text-sm font-medium text-gray-700 transition-colors"
                >
                  Copy
                </button>
              </div>
            </div>

            <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-4">
              <div className="flex items-start gap-2">
                <svg className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <p className="text-sm text-amber-800">
                  <strong>Important:</strong> This is the only time you'll see this key. Store it securely.
                </p>
              </div>
            </div>

            <div className="flex justify-end">
              <button
                type="button"
                onClick={() => {
                  setCreatedKey(null);
                  navigate(`/${module}/${createdKey.recordId}`);
                }}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
              >
                I've saved my key
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Password Modal (for user module) */}
      {showPasswordModal && isUserModule && id && (
        <PasswordModal
          userId={id}
          userEmail={userEmail || 'Unknown user'}
          onClose={() => setShowPasswordModal(false)}
          onSuccess={() => {
            // Could show a success message here
          }}
        />
      )}
    </div>
  );
}
