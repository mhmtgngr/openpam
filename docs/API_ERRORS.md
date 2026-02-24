# OpenPAM API Error Codes Reference

This document provides a comprehensive reference for all error codes returned by the OpenPAM API.

## HTTP Status Codes

The OpenPAM API uses standard HTTP status codes to indicate the success or failure of API requests.

| Status Code | Description |
|-------------|-------------|
| 200 OK | Request succeeded |
| 201 Created | Resource created successfully |
| 204 No Content | Request succeeded with no return data |
| 400 Bad Request | Invalid request data |
| 401 Unauthorized | Authentication required or failed |
| 403 Forbidden | Insufficient permissions |
| 404 Not Found | Resource not found |
| 413 Payload Too Large | Request body exceeds size limit |
| 415 Unsupported Media Type | Invalid Content-Type |
| 422 Unprocessable Entity | Semantic errors in request |
| 429 Too Many Requests | Rate limit exceeded |
| 500 Internal Server Error | Server error |

## Error Response Format

All error responses follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "additional": "context-specific information"
    },
    "request_id": "uuid-request-id"
  }
}
```

## Standard Error Codes

### Authentication & Authorization Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Missing or invalid authentication token |
| `INVALID_CREDENTIALS` | 401 | Username or password incorrect |
| `TOKEN_EXPIRED` | 401 | JWT token has expired |
| `TOKEN_INVALID` | 401 | JWT token signature invalid or malformed |
| `MFA_REQUIRED` | 403 | Multi-factor authentication required |
| `MFA_INVALID_CODE` | 403 | MFA verification code incorrect |
| `MFA_EXPIRED` | 403 | MFA verification expired (valid for 5 minutes) |
| `FORBIDDEN` | 403 | Insufficient permissions for requested action |
| `ACCOUNT_LOCKED` | 403 | User account is locked |
| `ACCOUNT_SUSPENDED` | 403 | User account is suspended |
| `CREDENTIAL_EXPIRED` | 403 | User credentials have expired |

### Resource Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `NOT_FOUND` | 404 | Requested resource does not exist |
| `USER_NOT_FOUND` | 404 | User not found |
| `TARGET_NOT_FOUND` | 404 | Target system not found |
| `CREDENTIAL_NOT_FOUND` | 404 | Credential not found |
| `SESSION_NOT_FOUND` | 404 | Session not found |
| `ROLE_NOT_FOUND` | 404 | Role not found |
| `POLICY_NOT_FOUND` | 404 | Policy not found |
| `TENANT_NOT_FOUND` | 404 | Tenant not found |

### Validation Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | General validation error |
| `INVALID_EMAIL` | 400 | Email address format invalid |
| `INVALID_PASSWORD` | 400 | Password does not meet requirements |
| `INVALID_INPUT` | 400 | Input data invalid |
| `MISSING_REQUIRED_FIELD` | 400 | Required field missing |
| `INVALID_BODY` | 400 | Request body malformed or invalid JSON |
| `DUPLICATE_RESOURCE` | 409 | Resource with conflicting attributes already exists |
| `USER_ALREADY_EXISTS` | 409 | User email already registered |
| `TARGET_ALREADY_EXISTS` | 409 | Target with same name/host already exists |

### Rate Limiting Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests; retry after specified time |

#### Rate Limit Error Response (429)

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Please retry after the specified time.",
    "details": {
      "limit": 100,
      "used": 105,
      "resets_at": "2026-02-24T12:35:00Z"
    },
    "request_id": "req_1234567890"
  }
}
```

#### Rate Limit HTTP Headers

The API includes rate limit information in HTTP headers on all responses:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed in the time window |
| `X-RateLimit-Used` | Number of requests made in the current window |
| `X-RateLimit-Remaining` | Number of requests remaining in the current window |
| `X-RateLimit-Reset` | ISO 8601 timestamp when the current window resets |
| `Retry-After` | (Only on 429 responses) Seconds until the rate limit resets |

#### Rate Limiting by Endpoint

| Endpoint Pattern | Limit | Window |
|------------------|-------|--------|
| `/api/v1/auth/login` | 5 | 5 minutes |
| `/api/v1/auth/*` | 10 | 5 minutes |
| `/api/v1/credentials/*/checkout` | 10 | 1 minute |
| Other endpoints | 100 | 1 minute |

### Session Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `SESSION_EXPIRED` | 401 | Session has expired |
| `SESSION_TERMINATED` | 403 | Session was terminated |
| `SESSION_NOT_ACTIVE` | 400 | Session is not in active state |
| `MAX_SESSIONS_REACHED` | 429 | Maximum concurrent sessions reached |
| `SESSION_TIMEOUT` | 408 | Session timed out due to inactivity |

### Credential Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `CREDENTIAL_CHECKED_OUT` | 409 | Credential is currently checked out |
| `CREDENTIAL_ROTATION_FAILED` | 500 | Automatic credential rotation failed |
| `CREDENTIAL_EXPIRED` | 403 | Credential has expired |
| `CHECKOUT_NOT_ALLOWED` | 403 | User not permitted to checkout this credential |
| `APPROVAL_REQUIRED` | 403 | Credential checkout requires approval |
| `APPROVAL_DENIED` | 403 | Request for credential access was denied |

### Request/Approval Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `REQUEST_NOT_FOUND` | 404 | Access request not found |
| `REQUEST_ALREADY_APPROVED` | 409 | Request has already been approved |
| `REQUEST_ALREADY_DENIED` | 409 | Request has already been denied |
| `REQUEST_EXPIRED` | 410 | Access request has expired |
| `APPROVAL_WORKFLOW_ERROR` | 500 | Error in approval workflow processing |
| `APPROVER_NOT_FOUND` | 404 | Specified approver not found |

### Audit & Compliance Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `AUDIT_LOG_CORRUPTED` | 500 | Audit log integrity check failed |
| `COMPLIANCE_REPORT_FAILED` | 500 | Failed to generate compliance report |
| `AUDIT_QUERY_FAILED` | 500 | Audit log query execution failed |

### Policy Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `POLICY_EVALUATION_FAILED` | 500 | Failed to evaluate access policy |
| `POLICY_CONFLICT` | 409 | Multiple policies have conflicting rules |
| `POLICY_VALIDATION_FAILED` | 400 | Policy definition is invalid |

### Tenant Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `TENANT_LIMIT_REACHED` | 429 | Maximum number of tenants reached |
| `USER_LIMIT_REACHED` | 429 | Tenant user limit exceeded |
| `TARGET_LIMIT_REACHED` | 429 | Tenant target limit exceeded |
| `TENANT_SUSPENDED` | 403 | Tenant account is suspended |

### System Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `INTERNAL_ERROR` | 500 | Unexpected internal server error |
| `DATABASE_ERROR` | 500 | Database operation failed |
| `CACHE_ERROR` | 500 | Cache operation failed |
| `SERVICE_UNAVAILABLE` | 503 | Required service is unavailable |
| `REQUEST_TIMEOUT` | 504 | Request processing timed out |
| `NETWORK_ERROR` | 503 | Network connectivity issue |

### Media Type & Size Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `UNSUPPORTED_MEDIA_TYPE` | 415 | Content-Type must be application/json |
| `REQUEST_TOO_LARGE` | 413 | Request body exceeds maximum allowed size |

### Request ID Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `MISSING_REQUEST_ID` | 400 | X-Request-ID header is required |
| `INVALID_REQUEST_ID` | 400 | X-Request-ID must be a valid UUID |

## Deprecated Error Codes

The following error codes are deprecated but may still be returned by older API versions:

| Deprecated Code | Replacement |
|-----------------|-------------|
| `1308` | `RATE_LIMIT_EXCEEDED` (HTTP 429) |

## Error Handling Best Practices

1. **Always check the HTTP status code first** - The error code provides detail, but the status code indicates the general category.

2. **Handle 429 responses with exponential backoff** - When receiving `RATE_LIMIT_EXCEEDED`, use the `Retry-After` header value to schedule retry attempts.

3. **Include the request_id in bug reports** - The `request_id` field helps trace requests through the system for debugging.

4. **Handle rate limits proactively** - Monitor the `X-RateLimit-*` headers to avoid hitting limits.

5. **Handle MFA_REQUIRED gracefully** - Some operations require MFA verification; prompt users accordingly.

## Example Error Handling

```typescript
try {
  const response = await api.post('/credentials/abc123/checkout');
  return response.data;
} catch (error) {
  if (error.response?.status === 429) {
    const { retry_after, resets_at } = error.response.data.error.details;
    // Schedule retry using the Retry-After header value
    setTimeout(() => retryRequest(), retry_after * 1000);
  } else if (error.response?.status === 403) {
    const code = error.response.data.error.code;
    if (code === 'MFA_REQUIRED') {
      // Prompt user for MFA
    } else if (code === 'APPROVAL_REQUIRED') {
      // Show approval workflow
    }
  }
  throw error;
}
```

## Change Log

| Date | Change |
|------|--------|
| 2026-02-24 | Added `RATE_LIMIT_EXCEEDED` error code documentation; deprecated numeric code 1308; added `Retry-After` header specification |
