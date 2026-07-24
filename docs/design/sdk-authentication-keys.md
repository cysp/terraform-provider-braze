# SDK Authentication key resource design

Status: accepted
Last verified against Braze documentation: 2026-07-24

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

A plural resource would make adding, promoting, and retiring one key changes to
one aggregate object. It would also couple independently owned key material and
make imports and staged rotation less precise.

Reads call the plural list endpoint and match the key ID. A successful list
that omits the ID means the key is absent. An error from the list endpoint,
including an HTTP 404, is not evidence that one particular key is absent.

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

The provider also rejects replacement of a currently primary key. This is a
provider safety policy rather than an additional Braze rule:

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
the provider preserves the returned ID and planned public values in state and
emits a warning. Returning a create error without state could orphan the remote
key and cause an unsafe duplicate on the next apply.

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
- [SDK Authentication developer guide](https://www.braze.com/docs/developer_guide/platform_wide/sdk_authentication/)
- [Terraform lifecycle reference](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle)
- [Terraform Plugin Framework plan modification](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification)
- [Terraform resource identity](https://developer.hashicorp.com/terraform/plugin/framework/resources/identity)
- [Removing resources from state](https://developer.hashicorp.com/terraform/language/state/remove)
