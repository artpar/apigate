/**
 * Schema API client for fetching module definitions.
 * Consumes /mod/_schema endpoints.
 */

import type {
  ModuleSchema,
  ModuleSummary,
  ListResponse,
  RecordResponse,
  Record,
} from '@/types/schema';

const API_BASE = '/mod';

/** Cache for schema data */
const schemaCache = new Map<string, { data: ModuleSchema; timestamp: number }>();
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

/** Default fetch options with credentials for session cookies */
const fetchOptions: RequestInit = {
  credentials: 'include',
};

/**
 * Fetch all available modules (summary list).
 */
export async function fetchModules(): Promise<ModuleSummary[]> {
  const response = await fetch(`${API_BASE}/_schema`, fetchOptions);
  if (!response.ok) {
    throw new Error(`Failed to fetch modules: ${response.statusText}`);
  }
  const data = await response.json();
  // API returns {modules: [{name, plural}, ...], count}
  // Transform to match ModuleSummary type (module instead of name)
  return (data.modules || []).map((m: { name: string; plural: string; description?: string }) => ({
    module: m.name,
    plural: m.plural,
    description: m.description || '',
  }));
}

/**
 * Fetch full schema for a specific module.
 */
export async function fetchModuleSchema(module: string): Promise<ModuleSchema> {
  // Check cache
  const cached = schemaCache.get(module);
  if (cached && Date.now() - cached.timestamp < CACHE_TTL) {
    return cached.data;
  }

  const response = await fetch(`${API_BASE}/_schema/${module}`, fetchOptions);
  if (!response.ok) {
    throw new Error(`Failed to fetch schema for ${module}: ${response.statusText}`);
  }

  const data = await response.json();
  schemaCache.set(module, { data, timestamp: Date.now() });
  return data;
}

/**
 * Clear schema cache (useful after module updates).
 */
export function clearSchemaCache(module?: string): void {
  if (module) {
    schemaCache.delete(module);
  } else {
    schemaCache.clear();
  }
}

/**
 * Fetch list of records for a module.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function fetchRecords(
  modulePlural: string,
  params?: { limit?: number; offset?: number }
): Promise<ListResponse<Record>> {
  const searchParams = new URLSearchParams();
  if (params?.limit) searchParams.set('limit', String(params.limit));
  if (params?.offset) searchParams.set('offset', String(params.offset));

  // Use the plural form for the API path (e.g., /mod/users)
  const url = `${API_BASE}/${modulePlural}${searchParams.toString() ? `?${searchParams}` : ''}`;
  const response = await fetch(url, fetchOptions);
  if (!response.ok) {
    throw new Error(`Failed to fetch ${modulePlural}: ${response.statusText}`);
  }
  return response.json();
}

/**
 * Fetch a single record by ID or lookup.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function fetchRecord(
  modulePlural: string,
  idOrLookup: string
): Promise<RecordResponse<Record>> {
  const response = await fetch(`${API_BASE}/${modulePlural}/${encodeURIComponent(idOrLookup)}`, fetchOptions);
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error(`Record not found: ${idOrLookup}`);
    }
    throw new Error(`Failed to fetch record: ${response.statusText}`);
  }
  return response.json();
}

/**
 * Create a new record.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function createRecord(
  modulePlural: string,
  data: Record
): Promise<RecordResponse<Record>> {
  const response = await fetch(`${API_BASE}/${modulePlural}`, {
    ...fetchOptions,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `Failed to create record`);
  }
  return response.json();
}

/**
 * Update an existing record.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function updateRecord(
  modulePlural: string,
  idOrLookup: string,
  data: Record
): Promise<RecordResponse<Record>> {
  const response = await fetch(`${API_BASE}/${modulePlural}/${encodeURIComponent(idOrLookup)}`, {
    ...fetchOptions,
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `Failed to update record`);
  }
  return response.json();
}

/**
 * Delete a record.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function deleteRecord(
  modulePlural: string,
  idOrLookup: string
): Promise<void> {
  const response = await fetch(`${API_BASE}/${modulePlural}/${encodeURIComponent(idOrLookup)}`, {
    ...fetchOptions,
    method: 'DELETE',
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `Failed to delete record`);
  }
}

/**
 * Execute a custom action on a record.
 * @param modulePlural - The plural name of the module (e.g., "users", "plans")
 */
export async function executeAction(
  modulePlural: string,
  action: string,
  idOrLookup: string,
  data?: Record
): Promise<RecordResponse<Record>> {
  const response = await fetch(
    `${API_BASE}/${modulePlural}/${encodeURIComponent(idOrLookup)}/${action}`,
    {
      ...fetchOptions,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: data ? JSON.stringify(data) : undefined,
    }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || `Failed to execute ${action}`);
  }
  return response.json();
}
