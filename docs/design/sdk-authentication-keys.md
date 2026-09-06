# SDK Authentication key resource design

Status: accepted
Last verified against Braze documentation: 2026-09-06

## Context

Braze SDK Authentication verifies practitioner-generated JWTs with RSA public
keys registered for an app. The private key remains outside Braze and outside
this provider. Registering a public key does not configure the app's separate
SDK Authentication enforcement mode.

Braze publishes these management operations:

| Operation | Method and path |
| --- | --- |
| Create | `POST /app_group/sdk_authentication/create` |
| List | `GET /app_group/sdk_authentication/keys?app_id=...` |
| Make primary | `PUT /app_group/sdk_authentication/primary` |
| Delete | `DELETE /app_group/sdk_authentication/delete` |

The provider design relies on the following documented constraints:

- Each app can have at most three keys.
- Create returns a generated key ID.
- List returns the app's complete key collection, including public-key
  material, description, and primary status.
- Primary and delete also return the complete resulting collection.
- There is no single-key read endpoint.
- There is no general update endpoint for public-key material or description.
- Braze exposes promotion, not direct demotion or clearing of the primary role.
- Whichever key is currently primary cannot be deleted. Another key must be
  promoted before deleting it.

## Decisions

### Model one key per resource

`braze_sdk_authentication_key` represents one member of Braze's bounded
per-app collection. This gives each key a stable Terraform address, which is
necessary for an explicit overlap period during rotation.

A plural resource can represent an exclusively owned collection and still support
staged rotation by retaining old and new members across applies. It needs a
separate contract for member identity, adoption, partial failure, and destroy;
this singular resource remains the building block for independently owned keys.

Reads call the plural list endpoint and match the key ID. A successful list
that omits the ID means the key is absent. An error from the list endpoint,
including an HTTP 404, is not evidence that one particular key is absent.
Collection responses must contain at most three keys with nonempty, unique IDs
and at most one primary. Validate that structure before interpreting absence,
promotion, or deletion. An empty collection and a collection without a primary
are valid observations.

### Use a composite identity

Braze addresses later operations with both the app ID and generated key ID.
Resource identity and string import therefore use `(app_id, key_id)`, with the
string form `<app_id>/<key_id>`.

Import observes remote primary status but does not create a primary-role claim.

### Treat non-primary fields as immutable

`app_id`, `rsa_public_key`, and `description` require replacement because the
documented API provides no operation to update them.

The provider accepts public-key text without imposing a narrower PEM parser or
minimum key size than Braze documents. Braze recommends 2048-bit RSA keys but
does not fully specify its accepted encodings or validation limits.

### Model primary as a claim and an observation

`primary = true` claims that this key should be primary. It promotes the key
during creation and repairs later drift.

Omitting `primary` leaves selection unmanaged while state still reports
Braze's current primary status. `primary = false` is invalid because Braze
cannot directly demote a key; demotion is a consequence of promoting another
key.

At most one resource per app should claim `primary = true`. Unmanaged peer keys
are allowed, but only one Terraform state or external controller should own the
primary-role decision.

### Separate the Braze deletion rule from provider replacement policy

Braze rejects deletion of whichever key it currently reports as primary, even
when other keys exist. The provider reports this known failure during planning.
Promote another key and apply first; after refresh observes the old key as
non-primary, remove it in a later apply.

A destroy of a managed set containing the currently primary key cannot complete
through the Braze API. A `removed` block with `destroy = false` is the explicit
way to relinquish Terraform management without deleting that remote key.

The provider rejects immutable-attribute changes to a currently primary key
during planning. Forced or tainted replacements can reach apply and fail at
deletion; Terraform does not expose every replacement reason to this plan hook.
The configuration-driven restriction is a provider policy rather than another
Braze rule:

- Terraform normally destroys the old object before creating its replacement,
  which Braze is guaranteed to reject while the old object is primary.
- `create_before_destroy` compresses creation, promotion, and deletion into one
  apply and cannot represent a controlled JWT issuer-migration interval.
- Replacement can exceed the three-key limit.
- Changing `app_id` cannot demote the primary key in the old app.

Safe rotation uses stable resource addresses:

1. Retain the old key and add the replacement.
2. Promote the replacement and apply.
3. Migrate JWT issuers during the overlap period.
4. Refresh so the old key is observed as non-primary.
5. Remove the old resource in a later apply.

The provider never silently selects another key to promote.

### Preserve identity after partial create success

Create returns only the new key ID, so the provider follows it with a list read
to hydrate and verify the key. If create succeeds but that verification fails,
the provider preserves the returned ID and accepted request values in state and
returns an error, matching the surrounding provider's create-recovery contract.
Unknown primary status becomes null in this recovery state. If a successful read
contradicts an explicit primary claim, the provider retains the observed status
and returns an error rather than fabricating the requested primary status.
Returning an error without state could orphan the remote key.

A create error can taint the key. A later refresh does not clear taint, and an
attempt to replace a key now observed as primary fails because Braze cannot
delete it. After fixing verification and checking the registered public key and
primary status, relinquish this resource with a `removed` block using
`destroy = false`, apply, restore its configuration, and import the same
`<app_id>/<key_id>`. A deliberate `terraform untaint` after independent verification
is another recovery option. Never automatically untaint or delete an unverified
key. Mocked Terraform acceptance tests cover failed verification, the rejected
replacement, state-only removal, and re-import without a second create.

The SDK adapter uses the common provider HTTP transport: ambiguous POST failures
are not replayed; explicit throttling follows the bounded retry policy. Operation
logs contain event names and identifiers, not request/response payloads.

Promotion responses are accepted only when the target is the sole reported
primary. Successful deletion responses are accepted only when the returned
collection omits the target.

### Do not encode unverified behavior

The implementation intentionally does not:

- retry list reads for an endpoint-specific propagation delay that Braze has
  not documented or that the project has not observed;
- reject RSA encodings or sizes beyond documented constraints;
- assume the first key becomes primary when `make_primary` is omitted;
- assume duplicate-key behavior, list ordering, or exact endpoint-specific
  status and error bodies;
- expose a plural data source without a demonstrated practitioner workflow; or
- retain list results in an in-memory cache, which could hide out-of-band drift.

At most three resource reads can share an app. If measurements later justify
optimizing those calls, coalescing concurrent in-flight list requests is safer
than retaining results across reads.

## Evidence boundary

No live Braze account was available for this implementation. The testserver is
deterministic but does not claim to establish undocumented service behavior.
Future changes must distinguish Braze documentation, direct live observations,
and testserver conventions.

Behavior still requiring live evidence or Braze clarification includes:

- first-key primary selection;
- accepted PEM encodings, normalization, and exact RSA limits;
- duplicate-key handling;
- idempotency of promoting the existing primary;
- concurrent mutation behavior;
- endpoint-specific propagation timing; and
- exact success codes and validation error bodies.

## Primary references

- [Create SDK Authentication key](https://www.braze.com/docs/api/endpoints/sdk_authentication/post_create_sdk_authentication_key/)
- [List SDK Authentication keys](https://www.braze.com/docs/api/endpoints/sdk_authentication/get_sdk_authentication_keys/)
- [Set primary SDK Authentication key](https://www.braze.com/docs/api/endpoints/sdk_authentication/put_primary_sdk_authentication_key/)
- [Delete SDK Authentication key](https://www.braze.com/docs/api/endpoints/sdk_authentication/delete_sdk_authentication_key/)
- [SDK Authentication developer guide](https://www.braze.com/docs/developer_guide/sdk_integration/authentication/)
- [Terraform lifecycle reference](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle)
- [Terraform Plugin Framework plan modification](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [Terraform resource identity](https://developer.hashicorp.com/terraform/plugin/framework/resources/identity)
- [Removing resources from state](https://developer.hashicorp.com/terraform/language/state/remove)

## Refresh against current provider

PR #113 was reconstructed on `240e4f0` rather than retaining its generated client
or surrounding implementation from July. The refresh uses the current ogen
version and structured error schema, propagates configuration diagnostics,
shares the current transport and logging behavior, and preserves failed-create
identities through the same model-plus-error contract as other resources.
SDK reads still require a successful collection response to establish absence;
the generic single-object HTTP 404 classifier is deliberately not reused.

Verification includes real Terraform against local HTTP fixtures, remote deletion
and rotation outcomes, disappearance/recreation, both import forms, configuration
diagnostics, failed-create recovery, and SDK-specific ambiguous-create/throttling
checks. These tests establish provider behavior, not undocumented Braze behavior.
