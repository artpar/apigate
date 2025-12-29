/**
 * Module View Page
 *
 * View and edit a single record.
 * Also handles create mode for new records.
 */

import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  fetchModuleSchema,
  fetchRecord,
  createRecord,
  updateRecord,
} from '@/api/schema';
import { DynamicForm } from '@/components/DynamicForm';
import type { Record } from '@/types/schema';

export function ModuleView() {
  const { module, id } = useParams<{ module: string; id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isCreate = id === 'new';

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
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['records', module] });
      navigate(`/${module}/${result.data.id}`);
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

  return (
    <div className="space-y-6">
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
    </div>
  );
}
