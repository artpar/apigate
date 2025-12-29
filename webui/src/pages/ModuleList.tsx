/**
 * Module List Page
 *
 * Generic list view for any module.
 * Uses DynamicTable for rendering.
 */

import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchModuleSchema, fetchRecords, deleteRecord } from '@/api/schema';
import { DynamicTable } from '@/components/DynamicTable';

export function ModuleList() {
  const { module } = useParams<{ module: string }>();
  const queryClient = useQueryClient();
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  // Fetch schema
  const { data: schema, isLoading: schemaLoading } = useQuery({
    queryKey: ['schema', module],
    queryFn: () => fetchModuleSchema(module!),
    enabled: !!module,
  });

  // Fetch records (uses schema.plural for API path)
  const { data: recordsData, isLoading: recordsLoading } = useQuery({
    queryKey: ['records', module],
    queryFn: () => fetchRecords(schema!.plural, { limit: 100 }),
    enabled: !!module && !!schema,
  });

  // Delete mutation (uses schema.plural for API path)
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteRecord(schema!.plural, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['records', module] });
      setDeleteConfirm(null);
    },
  });

  const handleDelete = (id: string) => {
    setDeleteConfirm(id);
  };

  const confirmDelete = () => {
    if (deleteConfirm) {
      deleteMutation.mutate(deleteConfirm);
    }
  };

  if (schemaLoading) {
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 capitalize">{schema.plural}</h1>
          <p className="text-gray-500">{schema.description || `Manage ${schema.plural}`}</p>
        </div>
        <Link
          to={`/${module}/new`}
          className="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create {schema.module}
        </Link>
      </div>

      {/* Table */}
      <DynamicTable
        module={schema}
        records={recordsData?.data || []}
        isLoading={recordsLoading}
        onDelete={handleDelete}
      />

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Confirm Delete</h3>
            <p className="text-gray-600 mb-6">
              Are you sure you want to delete this {schema.module}? This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmDelete}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
