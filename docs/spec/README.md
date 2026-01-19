# APIGate API Specification

> **This is the single source of truth for API behavior.**
>
> - If behavior is documented here, it MUST be implemented in code.
> - If behavior is implemented in code, it MUST be documented here.
> - Tests MUST verify behavior matches this specification.

## Specification Documents

| Document | Description | Implementation |
|----------|-------------|----------------|
| [JSON:API Format](json-api.md) | Response format specification | `pkg/jsonapi/` |
| [Error Codes](error-codes.md) | All error codes and HTTP statuses | `pkg/jsonapi/errors.go` |
| [Pagination](pagination.md) | Pagination behavior and parameters | `pkg/jsonapi/pagination.go` |
| [Resource Types](resource-types.md) | All API resource types and attributes | `adapters/http/admin/` |

## Compliance

### JSON:API Specification

APIGate implements [JSON:API v1.1](https://jsonapi.org/) with the following characteristics:

- **Content-Type**: `application/vnd.api+json`
- **Version**: 1.1
- **Top-level members**: `data`, `errors`, `meta`, `links`, `included`, `jsonapi`

### Validation Rules

1. Every API endpoint MUST return JSON:API compliant responses
2. Every error MUST use the JSON:API error format
3. Every collection MUST support pagination
4. Every resource MUST have `type` and `id` as top-level members
5. Resource attributes MUST NOT include `id` or `type`

## Adding New Features

When adding a new API feature:

1. **Document first**: Add the specification to the appropriate file
2. **Implement**: Write code that matches the specification
3. **Test**: Write tests that verify spec compliance
4. **Review**: Ensure spec and implementation match exactly

## Spec-Code Alignment

Run alignment checks:

```bash
# Verify all error codes are documented
go test ./pkg/jsonapi/... -run TestErrorCodesDocumented

# Verify all resource types are documented
go test ./adapters/http/... -run TestResourceTypesDocumented
```

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-01-19 | Initial JSON:API alignment |
