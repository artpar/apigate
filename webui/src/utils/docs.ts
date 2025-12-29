/**
 * Documentation generation utilities.
 * Generates API examples, curl commands, and documentation
 * from module schemas.
 */

import type {
  ModuleSchema,
  FieldSchema,
  ActionSchema,
  APIExample,
  Constraint,
} from '@/types/schema';

/**
 * Generate curl command for an action.
 */
export function generateCurl(
  action: ActionSchema,
  module: ModuleSchema,
  sampleData?: Record<string, unknown>
): string {
  const baseUrl = 'http://localhost:8080';
  let path = action.http.path
    .replace('{module}', module.plural)
    .replace('{id}', '<id>');

  let curl = `curl -X ${action.http.method} "${baseUrl}${path}"`;

  // Add headers
  curl += ` \\\n  -H "Content-Type: application/json"`;
  if (action.auth && action.auth !== 'none') {
    curl += ` \\\n  -H "X-API-Key: <your-api-key>"`;
  }

  // Add body for POST/PUT
  if (['POST', 'PUT', 'PATCH'].includes(action.http.method) && sampleData) {
    curl += ` \\\n  -d '${JSON.stringify(sampleData, null, 2)}'`;
  }

  return curl;
}

/**
 * Generate sample data for a module based on field types.
 */
export function generateSampleData(
  module: ModuleSchema,
  forAction: 'create' | 'update' = 'create'
): Record<string, unknown> {
  const sample: Record<string, unknown> = {};

  for (const field of module.fields) {
    // Skip internal, computed fields
    if (field.internal || field.computed) continue;
    // Skip immutable fields for update
    if (forAction === 'update' && field.immutable) continue;

    sample[field.name] = generateSampleValue(field);
  }

  return sample;
}

/**
 * Generate a sample value for a field based on its type.
 */
export function generateSampleValue(field: FieldSchema): unknown {
  if (field.default !== undefined) {
    return field.default;
  }

  switch (field.type) {
    case 'string':
    case 'text':
      return `sample_${field.name}`;
    case 'int':
      return 100;
    case 'float':
      return 99.99;
    case 'bool':
      return true;
    case 'email':
      return 'user@example.com';
    case 'url':
      return 'https://example.com';
    case 'uuid':
      return '550e8400-e29b-41d4-a716-446655440000';
    case 'datetime':
      return new Date().toISOString();
    case 'date':
      return new Date().toISOString().split('T')[0];
    case 'time':
      return '12:00:00';
    case 'duration':
      return '1h30m';
    case 'json':
      return { key: 'value' };
    case 'secret':
      return '********';
    case 'enum':
      return field.values?.[0] ?? 'option1';
    case 'ref':
      return `${field.ref}_id_123`;
    case 'bytes':
      return '<binary data>';
    default:
      return null;
  }
}

/**
 * Generate API examples for a module.
 */
export function generateAPIExamples(module: ModuleSchema): APIExample[] {
  const examples: APIExample[] = [];

  for (const action of module.actions) {
    const example = generateActionExample(action, module);
    if (example) {
      examples.push(example);
    }
  }

  return examples;
}

/**
 * Generate example for a specific action.
 */
export function generateActionExample(
  action: ActionSchema,
  module: ModuleSchema
): APIExample | null {
  const sampleData = action.type === 'create' || action.type === 'update'
    ? generateSampleData(module, action.type)
    : undefined;

  const path = action.http.path
    .replace('{module}', module.plural)
    .replace('{id}', 'user_123');

  return {
    title: `${action.type.charAt(0).toUpperCase() + action.type.slice(1)} ${module.module}`,
    description: action.description,
    method: action.http.method,
    path,
    curl: generateCurl(action, module, sampleData),
    requestBody: sampleData ? JSON.stringify(sampleData, null, 2) : undefined,
    responseBody: generateResponseExample(action, module),
  };
}

/**
 * Generate sample response for an action.
 */
function generateResponseExample(
  action: ActionSchema,
  module: ModuleSchema
): string {
  switch (action.type) {
    case 'list':
      return JSON.stringify({
        module: module.module,
        count: 2,
        data: [generateSampleData(module), generateSampleData(module)],
      }, null, 2);
    case 'get':
    case 'create':
    case 'update':
      return JSON.stringify({
        module: module.module,
        data: generateSampleData(module),
      }, null, 2);
    case 'delete':
      return JSON.stringify({ success: true }, null, 2);
    default:
      return JSON.stringify({
        module: module.module,
        data: generateSampleData(module),
      }, null, 2);
  }
}

/**
 * Format field type for display.
 */
export function formatFieldType(field: FieldSchema): string {
  if (field.type === 'enum' && field.values?.length) {
    return `enum(${field.values.join(' | ')})`;
  } else if (field.type === 'ref' && field.ref) {
    return `ref(${field.ref})`;
  }

  return field.type;
}

/**
 * Format constraint for display.
 */
export function formatConstraint(constraint: Constraint): string {
  switch (constraint.type) {
    case 'min':
      return `Minimum value: ${constraint.value}`;
    case 'max':
      return `Maximum value: ${constraint.value}`;
    case 'min_length':
      return `Minimum length: ${constraint.value} characters`;
    case 'max_length':
      return `Maximum length: ${constraint.value} characters`;
    case 'pattern':
      return `Must match pattern: ${constraint.value}`;
    case 'ref_exists':
      return `Referenced record must exist`;
    default:
      return `${constraint.type}: ${constraint.value}`;
  }
}

/**
 * Get validation rules description for a field.
 */
export function getFieldValidation(field: FieldSchema): string[] {
  const rules: string[] = [];

  if (field.required) {
    rules.push('Required field');
  }
  if (field.unique) {
    rules.push('Must be unique');
  }
  if (field.immutable) {
    rules.push('Cannot be changed after creation');
  }

  if (field.constraints) {
    for (const c of field.constraints) {
      rules.push(formatConstraint(c));
    }
  }

  // Type-specific validation
  switch (field.type) {
    case 'email':
      rules.push('Must be a valid email address');
      break;
    case 'url':
      rules.push('Must be a valid URL');
      break;
    case 'uuid':
      rules.push('Must be a valid UUID');
      break;
  }

  return rules;
}
