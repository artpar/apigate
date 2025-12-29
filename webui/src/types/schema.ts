/**
 * TypeScript types matching Go schema introspection API.
 * These types mirror core/schema/introspect.go
 */

/** Field type enum matching schema.FieldType */
export type FieldType =
  | 'string'
  | 'int'
  | 'float'
  | 'bool'
  | 'email'
  | 'url'
  | 'uuid'
  | 'datetime'
  | 'date'
  | 'time'
  | 'duration'
  | 'json'
  | 'text'
  | 'secret'
  | 'enum'
  | 'ref'
  | 'bytes';

/** Action type enum matching schema.ActionType */
export type ActionType =
  | 'list'
  | 'get'
  | 'create'
  | 'update'
  | 'delete'
  | 'custom';

/** Constraint type enum */
export type ConstraintType =
  | 'min'
  | 'max'
  | 'min_length'
  | 'max_length'
  | 'pattern'
  | 'ref_exists';

/** Constraint definition */
export interface Constraint {
  type: ConstraintType;
  value: string | number;
  message?: string;
}

/** HTTP method and path for an action */
export interface HTTPInfo {
  method: string;
  path: string;
}

/** Input field schema for actions */
export interface InputSchema {
  name: string;
  type: FieldType;
  required: boolean;
  default?: string;
  description?: string;
}

/** Field schema from module definition */
export interface FieldSchema {
  name: string;
  type: FieldType;
  required: boolean;
  unique?: boolean;
  values?: string[];      // enum options
  ref?: string;           // foreign key target module
  default?: unknown;
  internal?: boolean;     // hide from UI/API
  computed?: boolean;     // auto-generated
  immutable?: boolean;    // cannot update after create
  description?: string;
  constraints?: Constraint[];
}

/** Action schema from module definition */
export interface ActionSchema {
  name: string;
  type: ActionType;
  description: string;
  input: InputSchema[];
  auth: string;
  confirm: boolean;
  http: HTTPInfo;
}

/** Full module schema */
export interface ModuleSchema {
  module: string;
  plural: string;
  description: string;
  fields: FieldSchema[];
  actions: ActionSchema[];
  lookups: string[];
  constraints?: Constraint[];
}

/** Module list item (summary) */
export interface ModuleSummary {
  module: string;
  plural: string;
  description: string;
}

/** API list response wrapper */
export interface ListResponse<T> {
  module: string;
  count: number;
  data: T[];
}

/** API single record response wrapper */
export interface RecordResponse<T> {
  module: string;
  data: T;
}

/** API error response */
export interface ErrorResponse {
  error: string;
  field?: string;
  code?: string;
}

/** Generic record type */
export type Record = {
  [key: string]: unknown;
  id?: string;
};

/** Documentation focus target */
export interface DocFocus {
  type: 'module' | 'field' | 'action' | 'none';
  module?: ModuleSchema;
  field?: FieldSchema;
  action?: ActionSchema;
}

/** API example for documentation */
export interface APIExample {
  title: string;
  description: string;
  method: string;
  path: string;
  curl: string;
  requestBody?: string;
  responseBody?: string;
}
