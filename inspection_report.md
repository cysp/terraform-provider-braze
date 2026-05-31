# Terraform Provider Braze - Code Inspection Report

## Executive Summary

This report provides a comprehensive inspection of the terraform-provider-braze codebase, including:
- Resource and list-resource implementations
- Provider implementation
- OpenAPI schema definitions
- Code-generated client/server (via ogen)
- Hand-written client code

## Architecture Overview

The provider follows a clean architecture with:
- **Provider Layer**: `internal/provider/` - Terraform-specific logic
- **Client Layer**: `internal/braze-client-go/` - Braze API client (mix of generated and hand-written)
- **OpenAPI Specs**: `internal/braze-client-go/openapi/` - API definitions

## 1. Resource Implementation: braze_content_block

### File: `internal/provider/braze_content_block_resource.go`

**Key Components:**
- Implements Terraform Framework's resource interfaces:
  - `resource.Resource`
  - `resource.ResourceWithConfigure`
  - `resource.ResourceWithIdentity`
  - `resource.ResourceWithImportState`

**CRUD Operations:**

#### Create (Lines 50-107)
- Converts Terraform plan to `CreateContentBlockRequest`
- Calls `client.CreateContentBlock()`
- Retrieves created resource via `GetContentBlockInfo()` for state consistency
- Sets identity and state with response data
- **Error Handling**: Checks for nil response and HTTP 404 after creation

#### Read (Lines 109-148)
- Fetches resource via `GetContentBlockInfo()`
- Handles 404 by removing resource from state (graceful degradation)
- Updates state with fetched data

#### Update (Lines 150-208)
- Converts plan to `UpdateContentBlockRequest`
- Calls `client.UpdateContentBlock()`
- Re-fetches via `GetContentBlockInfo()` for consistency
- Updates identity and state

#### Delete (Lines 210-219)
- **Important**: Braze API has no delete endpoint
- Issues warning: "Braze does not provide a delete API for content blocks; resource removed from Terraform state only."

**Identity Management:**
- Uses `IDIdentityModel` with `id` field
- Import state passes through identity: `resource.ImportStatePassthroughWithIdentity`

### File: `internal/provider/braze_content_block_resource_schema.go`

**Schema Definition:**

```go
Attributes:
- id: StringAttribute (optional, computed, requires replace)
- name: StringAttribute (required)
- description: StringAttribute (optional)
- content: StringAttribute (required)
- tags: ListAttribute with custom TypedList type (optional)
```

**Identity Schema:**
```go
id: StringAttribute (required for import)
```

### Model Files

#### `braze_content_block_model.go`
```go
type brazeContentBlockModel struct {
    IDIdentityModel
    Name        types.String
    Description types.String
    Content     types.String
    Tags        TypedList[types.String]
}
```

#### `braze_content_block_model_create_request.go`
- Converts Terraform model to `CreateContentBlockRequest`
- Handles nullable fields properly
- Tags: converts TypedList to slice, handles null

#### `braze_content_block_model_update_request.go`
- Converts to `UpdateContentBlockRequest`
- All fields except `content_block_id` are optional (OptString, OptNilString)

#### `braze_content_block_model_response.go`
- Converts `GetContentBlockInfoResponse` to Terraform model
- Handles nullable description and tags properly

## 2. List-Resource Implementation: braze_content_block

### File: `internal/provider/braze_content_block_list_resource.go`

**Key Components:**
- Implements:
  - `list.ListResource`
  - `list.ListResourceWithConfigure`

**Configuration Schema:**
```go
modified_after: RFC3339 timestamp (optional)
modified_before: RFC3339 timestamp (optional)
```

**List Operation (Lines 60-153):**
- Builds `ListContentBlocksParams` with limit, filters
- Calls `client.ListContentBlocks()`
- Iterates over results, yielding each as `ListResult`
- If `IncludeResource` is true, fetches full resource via `GetContentBlockInfo()`
- Sets identity and display name for each result

**Stream-based Results:**
- Uses Go 1.23+ iterator pattern with `yield` function
- Allows consumer to control flow (early termination)

## 3. Provider Implementation

### File: `internal/provider/braze_provider.go`

**Provider Type:**
```go
type brazeProvider struct {
    version    string
    baseURL    string
    apiKey     string
    httpClient *http.Client
}
```

**Configuration Schema:**
```go
base_url: StringAttribute (optional)
api_key: StringAttribute (optional, sensitive)
```

**Configure Method (Lines 81-143):**
1. Reads config from Terraform
2. Falls back to `BRAZE_API_KEY` environment variable
3. Creates retryable HTTP client with linear jitter backoff
4. Initializes Braze client with:
   - Base URL
   - API key security source
   - Custom user agent: `terraform-provider-braze/{version}`
5. Stores client in provider data

**Resources & List Resources:**
```go
Resources: [NewBrazeContentBlockResource]
ListResources: [NewBrazeContentBlockListResource]
DataSources: [] (empty)
```

**Provider Options (for testing):**
- `WithBaseURL(url)`
- `WithHTTPClient(client)`
- `WithAPIKey(key)`

### Supporting Files

#### `braze_api_key_security_source.go`
- Implements `brazeclient.SecuritySource`
- Returns `BrazeApiKey` with token for authentication

#### `http_client_user_agent.go`
- Wrapper that injects User-Agent header
- Format: `terraform-provider-braze/{version}`

#### `provider_data.go`
- Generic helper to extract provider data from configure requests
- Supports DataSource, List, Resource contexts

## 4. Custom Types: TypedList

### Files: `typed_list.go`, `typed_list_type.go`, `typed_list_string.go`

**Purpose:** Type-safe list handling with proper null/unknown semantics

**TypedList[T attr.Value]:**
- Generic type parameterized on element type
- States: Known, Null, Unknown
- Implements `attr.Value` and `basetypes.ListValuable`
- Immutable: `Elements()` returns a clone

**TypedListType[T attr.Value]:**
- Implements `attr.Type`, `attr.TypeWithElementType`, `basetypes.ListTypable`
- Converts between Terraform values and Go types
- Handles nested type information

**String Helpers:**
- `NewTypedListFromStringSlice`: []string → TypedList[types.String]
- `TypedListToStringSlice`: TypedList[types.String] → []string
- Filters out null/unknown elements during conversion

## 5. OpenAPI Schema

### File: `internal/braze-client-go/openapi/openapi.yml`

**API Endpoints:**

1. **GET /content_blocks/list**
   - Parameters: modified_after, modified_before, limit, offset
   - Returns: ListContentBlocksResponse

2. **GET /content_blocks/info**
   - Parameters: content_block_id, include_inclusion_data
   - Returns: GetContentBlockInfoResponse

3. **POST /content_blocks/create**
   - Body: CreateContentBlockRequest
   - Returns: CreateContentBlockResponse (201)

4. **POST /content_blocks/update**
   - Body: UpdateContentBlockRequest
   - Returns: UpdateContentBlockResponse (200)

**Security:**
```yaml
securitySchemes:
  brazeApiKey:
    type: http
    scheme: bearer
    bearerFormat: Braze
```

### Schema Files

#### `schemas/content_blocks/create/request.yml`
```yaml
required: [name, content]
properties:
  name: string (< 100 chars)
  description: string, nullable (< 250 chars)
  content: string (HTML/text)
  state: enum [active, draft] (default: active)
  tags: array of strings, nullable
```

#### `schemas/content_blocks/info/response.yml`
```yaml
required: [content_block_id, name, content]
properties:
  content_block_id: string
  name: string
  content: string
  description: string, nullable
  tags: array of strings, nullable
```

#### `schemas/content_blocks/list/response.yml`
```yaml
required: [count, content_blocks]
properties:
  count: integer
  content_blocks: array of ContentBlock
    - content_block_id: string (required)
    - name: string (required)
    - tags: array, nullable
```

#### `schemas/content_blocks/update/request.yml`
```yaml
required: [content_block_id]
properties:
  content_block_id: string
  name: string (optional)
  description: string, nullable (optional)
  content: string (optional)
  state: enum [active, draft] (optional)
  tags: array, nullable (optional)
```

#### `responses/error.yml`
```yaml
required: [message]
properties:
  message: string
  errors: array of strings (optional)
```

## 6. Code-Generated Client (ogen)

### Configuration: `internal/braze-client-go/ogen.yml`

```yaml
parser:
  allow_remote: true
  infer_types: true

generator:
  features:
    enable:
      - paths/client
      - paths/server
      - client/security/reentrant
      - client/request/validation
      - server/response/validation
  convenient_errors: "on"
```

### Generated Files (oas_*_gen.go)

#### `oas_client_gen.go`
- **Client struct**: holds serverURL, security source, baseClient
- **Invoker interface**: defines all API operations
- **Operations**: CreateContentBlock, GetContentBlockInfo, ListContentBlocks, UpdateContentBlock
- **URL handling**: context-based server URL override
- **Security**: applies BrazeApiKey to requests

#### `oas_schemas_gen.go`
- **Type definitions** for all OpenAPI schemas
- **Getters/Setters** for all fields
- **Custom types**:
  - `OptString`: optional string (Set flag)
  - `OptNilString`: optional nullable string (Set, Null flags)
  - `OptNilStringArray`: optional nullable string array
  - State enums with MarshalText/UnmarshalText

**Key Types:**
```go
CreateContentBlockRequest {
    Name        string
    Description OptNilString
    Content     string
    State       OptCreateContentBlockRequestState
    Tags        OptNilStringArray
}

UpdateContentBlockRequest {
    ContentBlockID string
    Name           OptString
    Description    OptNilString
    Content        OptString
    State          OptUpdateContentBlockRequestState
    Tags           OptNilStringArray
}

GetContentBlockInfoResponse {
    ContentBlockID string
    Name           string
    Content        string
    Description    OptNilString
    Tags           OptNilStringArray
}

ErrorResponseStatusCode {
    StatusCode int
    Response   ErrorResponse
}
```

#### `oas_server_gen.go`
- **Server struct**: HTTP handler for API
- **Handler interface**: must be implemented by server
- **Routing**: maps HTTP requests to handler methods

#### `oas_json_gen.go`
- JSON encoding/decoding for all types
- Uses go-faster/jx for performance

#### Other Generated Files
- `oas_request_encoders_gen.go`: encodes request bodies
- `oas_request_decoders_gen.go`: decodes request bodies
- `oas_response_encoders_gen.go`: encodes responses
- `oas_response_decoders_gen.go`: decodes responses
- `oas_parameters_gen.go`: query/path parameter handling
- `oas_validators_gen.go`: request/response validation
- `oas_security_gen.go`: security handling
- `oas_router_gen.go`: HTTP routing
- `oas_cfg_gen.go`: client configuration

## 7. Hand-Written Client Code

### File: `internal/braze-client-go/braze_client.go`
```go
const DefaultUserAgent = "braze-client-go/0.1"
```

### File: `internal/braze-client-go/optnilstring.go`
**Helper functions:**
```go
NewOptNilPointerString(v *string) OptNilString
  - nil → SetToNull()
  - non-nil → SetTo(*v)

GetPointer() *string
  - Returns nil if not set or null
  - Returns &Value otherwise
```

### File: `internal/braze-client-go/optstring.go`
**Helper functions:**
```go
NewOptPointerString(v *string) OptString
GetPointer() *string
```

### File: `internal/braze-client-go/ogen.go`
```go
//go:generate go run github.com/ogen-go/ogen/cmd/ogen -target . -package brazeclient -clean openapi/openapi.yml
```
- Defines code generation command
- Regenerates client from OpenAPI spec

## 8. Test Server Implementation

### File: `internal/braze-client-go/testing/server.go`

**Server struct:**
```go
type Server struct {
    server  *brazeclient.Server  // Generated server
    handler *Handler             // Custom handler
}
```

**noOpSecurityHandler:**
- Allows all requests without authentication
- For testing only

### File: `internal/braze-client-go/testing/handler.go`

**Handler struct:**
```go
type Handler struct {
    mu            sync.Mutex
    contentBlocks map[string]*brazeclient.GetContentBlockInfoResponse
}
```

**Purpose:** In-memory mock Braze API

### File: `internal/braze-client-go/testing/handler_content_blocks.go`

**Operations:**

1. **ListContentBlocks** (Lines 13-30)
   - Returns all content blocks from in-memory map
   - Converts to ListContentBlocksResponseContentBlock format

2. **GetContentBlockInfo** (Lines 32-42)
   - Looks up by ID
   - Returns 404 if not found

3. **CreateContentBlock** (Lines 44-78)
   - Validates name is not empty (422 if empty)
   - Generates UUID for new block
   - Stores in map
   - Returns CreateContentBlockResponse with ID

4. **UpdateContentBlock** (Lines 80-122)
   - Looks up existing block
   - Validates name if provided (422 if empty)
   - Updates fields if provided (Name, Content, Description, Tags)
   - Preserves unmodified fields

5. **setContentBlock** (Lines 124-145)
   - Helper to pre-populate content blocks for testing

**Error Handling:**
```go
type statusCodeError struct {
    StatusCode int
}

var errNotFound = newStatusCodeError(http.StatusNotFound)
```

### File: `internal/braze-client-go/testing/error.go`
- Custom error type with status codes
- Used by handler to return appropriate HTTP errors

## 9. Testing Infrastructure

### File: `internal/provider/braze_provider_testing_test.go`

**Test Helpers:**

1. **BrazeProviderMockedResourceTest**
   - Always uses mock server
   - Creates httptest.Server with provided handler
   - Injects test server URL and client into provider

2. **BrazeProviderMockableResourceTest**
   - Uses mock server if `TF_ACC_MOCKED` is set
   - Otherwise uses real API (for acceptance tests)

3. **BrazeProviderOptionsWithHTTPTestServer**
   - Creates provider options for test server:
     - Base URL: test server URL
     - HTTP Client: test server client
     - API Key: "12345" (mock)

### Test Cases

#### `braze_content_block_resource_test.go`

**TestAccBrazeContentBlock:**
- Creates content block without tags
- Imports resource
- Verifies no-op plan
- Updates with tags
- Updates tags to empty list
- Removes tags attribute
- Destroys resource

**TestAccBrazeContentBlockCreateNameEmpty:**
- Attempts to create with empty name
- Expects error: "Failed to create Content Block"

**TestAccBrazeContentBlockUpdateNameEmpty:**
- Creates with valid name
- Attempts to update to empty name
- Expects error: "Failed to update Content Block"

#### `braze_content_block_list_resource_test.go`

**TestAccBrazeContentBlockList:**
- Pre-populates test server with content block
- Queries list with date filters
- Includes full resource data
- Verifies all attributes

## 10. Key Design Patterns

### 1. Optional/Nullable Fields
- **OptString**: optional field (may be unset)
- **OptNilString**: optional nullable field (unset, null, or value)
- Three states: not set, null, explicitly set

### 2. Identity Separation
- Resources have separate identity schema (for import)
- Identity maintained independently from state
- Enables proper import/export behavior

### 3. State Consistency
- After Create/Update, always re-fetch via GET
- Ensures state matches server reality
- Handles server-side transformations

### 4. Error Propagation
- Client errors wrapped with context
- Status codes preserved in ErrorResponseStatusCode
- 404 handled specially (remove from state vs hard error)

### 5. Testing Strategy
- Mock server for unit/integration tests
- Optional real API for acceptance tests
- DRY with helper functions
- Test both success and error paths

### 6. Code Generation
- OpenAPI → ogen → Go types and client/server
- Hand-written helpers for common patterns
- Clear separation: generated vs manual code

## 11. Observations & Findings

### Strengths
1. **Clean Architecture**: Clear separation of concerns
2. **Type Safety**: Generic TypedList, proper null handling
3. **OpenAPI-First**: Schema-driven development
4. **Testability**: Comprehensive mock infrastructure
5. **Error Handling**: Proper 404 handling, status code preservation
6. **State Management**: Consistent re-fetch after mutations
7. **Documentation**: Well-commented schemas and code

### Areas of Note
1. **No Delete API**: Braze doesn't support deleting content blocks
   - Provider warns users
   - Resource only removed from Terraform state
2. **State Field**: Not exposed in Terraform resource
   - OpenAPI schema has state (active/draft)
   - Provider doesn't surface this to users
3. **Include Inclusion Data**: Not used
   - OpenAPI parameter: `include_inclusion_data`
   - Could show where content block is used
4. **Pagination**: List endpoint supports offset
   - Provider doesn't implement pagination
   - Could be added to list resource

### Code Quality
- Follows Go best practices
- Uses latest Terraform Plugin Framework features
- Proper use of contexts throughout
- Thread-safe test handler (mutex)
- Clear naming conventions

## 12. File Structure Summary

```
internal/
├── braze-client-go/
│   ├── openapi/
│   │   ├── openapi.yml                    # Main OpenAPI spec
│   │   ├── responses/error.yml            # Error response schema
│   │   └── schemas/content_blocks/        # Content Block schemas
│   │       ├── create/
│   │       │   ├── request.yml
│   │       │   └── response.yml
│   │       ├── info/response.yml
│   │       ├── list/response.yml
│   │       └── update/
│   │           ├── request.yml
│   │           └── response.yml
│   ├── testing/
│   │   ├── error.go                       # Error types for tests
│   │   ├── handler.go                     # Mock handler
│   │   ├── handler_content_blocks.go      # Content block operations
│   │   ├── server.go                      # Mock server
│   │   └── server_content_blocks.go       # Helper methods
│   ├── braze_client.go                    # Constants
│   ├── ogen.go                            # Code generation directive
│   ├── ogen.yml                           # ogen configuration
│   ├── optnilstring.go                    # Nullable optional helpers
│   ├── optstring.go                       # Optional helpers
│   └── oas_*_gen.go                       # Generated code (13 files)
└── provider/
    ├── braze_api_key_security_source.go   # API key auth
    ├── braze_content_block_list_resource.go
    ├── braze_content_block_list_resource_test.go
    ├── braze_content_block_model.go       # Terraform model
    ├── braze_content_block_model_create_request.go
    ├── braze_content_block_model_response.go
    ├── braze_content_block_model_update_request.go
    ├── braze_content_block_resource.go    # Main resource
    ├── braze_content_block_resource_schema.go
    ├── braze_content_block_resource_test.go
    ├── braze_provider.go                  # Provider implementation
    ├── braze_provider_data.go
    ├── braze_provider_option.go           # Provider options
    ├── braze_provider_test.go
    ├── braze_provider_testing_test.go     # Test helpers
    ├── error_detail.go                    # Error helpers
    ├── http_client_user_agent.go          # User-Agent injection
    ├── id_identity_model.go               # Identity model
    ├── provider_data.go                   # Provider data helpers
    ├── provider_data_test.go
    ├── state.go                           # State helpers
    ├── typed_list.go                      # Generic list type
    ├── typed_list_string.go               # String list helpers
    ├── typed_list_string_test.go
    ├── typed_list_test.go
    ├── typed_list_type.go                 # List type implementation
    └── typed_list_type_test.go
```

## 13. Dependencies

**Key Libraries:**
- `github.com/ogen-go/ogen`: OpenAPI code generation
- `github.com/hashicorp/terraform-plugin-framework`: Terraform provider SDK
- `github.com/hashicorp/terraform-plugin-testing`: Testing framework
- `github.com/hashicorp/go-retryablehttp`: HTTP retry logic
- `github.com/go-faster/jx`: Fast JSON encoding/decoding
- `github.com/google/uuid`: UUID generation

## Conclusion

This codebase demonstrates excellent software engineering practices:
- Clear separation between generated and hand-written code
- Comprehensive testing with mock infrastructure
- Proper handling of optional/nullable fields
- Type-safe abstractions (TypedList)
- OpenAPI-driven development
- Following Terraform Plugin Framework best practices

The implementation is production-ready and maintainable, with room for future enhancements such as pagination support, exposing the state field, and additional resources.
