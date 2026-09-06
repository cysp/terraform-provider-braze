# Plural SDK Authentication key resource

Status: implemented; reject plans requiring temporary primary promotion
Date: 2026-09-06

`braze_sdk_authentication_keys` manages the entire SDK Authentication key
collection for one Braze app. It uses an unordered set of objects that can be
reconstructed from Braze's response, with ordinary app-ID import. This is a
separate resource from the singular resource in PR #113.

## Configuration and lifecycle

```hcl
resource "braze_sdk_authentication_keys" "app" {
  app_id = var.app_id

  keys = [
    {
      rsa_public_key = file("current.pem")
      description    = "Current signing key"
      primary        = true
    },
    {
      rsa_public_key = file("next.pem")
      description    = "Next signing key"
      primary        = false
    },
  ]
}

import {
  to = braze_sdk_authentication_keys.app
  id = "01234567-89ab-cdef-0123-456789abcdef"
}
```

Use `for_each` on the resource to manage multiple apps. `/app_group` is an API
path prefix; all four operations require an individual `app_id`.

The resource takes exclusive ownership: a refreshed plan includes remote keys
omitted from configuration as members to remove. Do not manage the same app's
keys with singular resources, another plural resource, or another Terraform
state at the same time. Removing a member deletes that key. Destroying the
resource relinquishes management and retains **all** remote keys, with a clear
plan warning. Changing `app_id` replaces the management resource and consequently
retains keys in the old app.

For rotation, add the next key and apply, switch the primary designation when
appropriate, migrate JWT issuers while both keys remain registered, and remove
the previous key when it is no longer needed. These are documented workflow
steps, not provider-enforced phases. Combined additions, promotions, removals,
and member replacements are allowed subject to the operation-ordering contract
below. The provider never chooses a temporary primary on the practitioner's behalf.
Terraform cannot determine whether issuer migration or JWT expiry is complete.

## Schema

| Attribute | Contract |
| --- | --- |
| `app_id` | Required string; resource identity; changes replace the resource. |
| `keys` | Required `SetNestedAttribute`, one to three members. |
| `keys[*].rsa_public_key` | Required nonempty public-key string. |
| `keys[*].description` | Required nonempty string. |
| `keys[*].primary` | Required boolean; exactly one configured member is true. |
| `keys[*].id` | Computed Braze key ID; never an adoption argument. |

Identity import uses `{ app_id = "..." }`. String import accepts the app ID only.
There are no labels, custom import encodings, rotation timers, or private-key
attributes. Registration does not configure SDK Authentication enforcement.

Preserve imported descriptions verbatim. Braze's create API requires a nonempty
description, but its dashboard guide calls descriptions optional. Whether a
dashboard-created key can return an empty description is not established here.
If one is observed, the proposed nonempty configuration contract requires a
replacement with a description; import must not silently invent a description
or claim that every existing collection is configurable unchanged.

The plural resource can accept `primary = false` because it also knows which
other key must be primary. It performs promotion of that other key, not a direct
demotion API operation. This differs deliberately from the singular resource's
primary-role claim.

Validate known configuration values early and defer checks involving unknown
values until enough information is available. Duplicate descriptions are allowed.
Duplicate `(rsa_public_key, description)` pairs are ambiguous and must be
diagnosed before mutation, even when their primary flags differ. Do not add RSA
parsing or normalization rules beyond the established provider/API contract.

## Identity and Terraform planning

There are two identities: the management resource is identified by `app_id`, and
remote keys by Braze's generated IDs. The configured set describes desired public
material, descriptions, and primary status; its entries have no independent
Terraform resource addresses.

For reconciliation, match an existing member by the exact pair
`(rsa_public_key, description)`, independently of set ordering and primary status.
Retain its remote ID when promoting it. A change to either immutable public field
means creating a new member and retiring the old one. No nested attribute uses
`RequiresReplace`, since that would replace the entire management resource.

Initially leave computed IDs unknown when the collection changes instead of
adding complex nested ID plan modifiers. An unknown planned ID can resolve to
the existing ID during a promotion-only update. Terraform may display such an
update as removed/added set elements; the provider must not interpret that display
as an instruction to recreate the keys. Tests must prove that promotion sends
no create or delete requests.

Keep IDs inside the state set. They distinguish otherwise identical remote
objects and prevent an extra key from disappearing through set deduplication.
Preserve every returned ID on refresh. If multiple remote IDs have the same
immutable pair, diagnose the ambiguous reconciliation instead of arbitrarily
choosing one to promote or delete. Such a collection requires explicit resolution
before it can be reconciled; it must not be advertised as a lossless configurable
set. Descriptions alone are not unique identifiers.

## Reconciliation

Use a small deterministic planner with no HTTP or Terraform dependencies. Its
inputs are the observed collection and desired configuration; its output is an
ordered sequence of creates, promotions, and deletes, or an explanation that no
supported sequence exists. New members can be identified by their immutable
pair until create returns their remote ID.

Validate configuration independently from observed state. When Terraform's
effective plan equals the observed state, no reconciliation is needed; preserve
that plan even when `ignore_changes` retains an empty collection, no primary,
empty descriptions, or ambiguous immutable pairs. Those observations need not
satisfy the constraints for a desired collection that will be applied.

When `app_id` changes, skip reconciliation of the old app's effective collection
before checking desired-collection constraints. Ignored observations still refer
to the old app at that stage; Terraform plans the replacement's creation using
the configured keys. Configuration validation remains independent, and the old
app's keys are retained.

For changes requiring reconciliation, simulate the complete sequence during
planning when the relevant values are known, and preflight it again before
sending writes:

1. Validate the desired collection and establish unambiguous existing matches.
2. If the desired primary already exists, promote it first when necessary.
3. Prefer creating additions before retiring keys. Create the desired new primary
   before other additions, with `make_primary = true`.
4. When all three slots are occupied, retire only an explicitly unwanted,
   non-primary key to make space. This can be necessary even though additions
   normally precede deletion.
5. Once the desired primary is established, retire the remaining unwanted keys.

Only promote the configured final primary. Reject a plan that requires promoting
any other key temporarily; disclosing such a promotion in a warning does not
make it acceptable. Never delete a member that remains desired.
Stable ID ordering can break ties between equally valid deletions; input set
ordering must not affect the outcome.

For example, suppose A is primary and A, B, C occupy all three slots. The desired
collection is B, C, D with D primary. A is the only removable key, but cannot be
deleted before another key becomes primary, and there is no slot for D. Reject
this transition with a plan error before writing anything. The practitioner can
explicitly promote B or C in a preceding apply, or retire another key to free a
slot. In contrast,
if C is also being retired, delete C, create D as primary, then delete A.

This restriction is an explicit-selection policy, not an API impossibility:
promoting B temporarily would permit deleting A and then creating D as primary.
The error explains the capacity and primary-deletion constraints. The practitioner
can resolve it through two ordinary resource plans, explicitly selecting the
intermediate primary:

| Plan | Configured collection | Mutations |
| --- | --- | --- |
| 1 | A, B, C; B primary | Promote B; retain all three IDs. |
| 2, after applying plan 1 | B, C, D; D primary | Delete A to free a slot; create D as primary. |

This requires two applies for this transition. It does not impose staged rotation
on changes that can reach the configured primary without an intermediate one.
When all relevant inputs are known, report the unsupported direct transition
during planning, identifying the current primary and explaining how to explicitly
select a surviving key in a preceding apply. If inputs remain unknown, defer
the decision and still preflight before any writes: never introduce a temporary
promotion during apply. Single-apply changes remain supported whenever operation
ordering can reach the desired collection by promoting only its final primary.
No transition selector, automatic fallback, or dedicated action is needed.

## Resource operations

- **Create:** list first. Require import if any keys already exist, so adopting
  an occupied collection cannot hide removal decisions inside a create plan.
  Otherwise reconcile the empty collection to the configured set.
- **Read:** list the whole collection and store all remote members and observed
  primary flags. Empty success is an existing empty collection, not grounds to
  remove the management resource from state. API errors, including endpoint 404,
  preserve state and produce an error. Do not cache snapshots across operations.
- **Update:** re-read before writes, compare with the snapshot used for planning,
  and execute the preflighted sequence. If external changes introduce different
  identities, material, descriptions, or primary status, require a fresh plan.
- **Delete:** perform no Braze mutations; relinquish management. State-only
  destruction must also work when an observed collection contains ambiguous
  duplicate members.
- **Import:** record the app identity and hydrate the full collection through
  Read. Import performs no mutation. A collection with no primary can be observed;
  the subsequent configuration must choose one. Multiple reported primaries are
  an invalid response, not a reason to select one arbitrarily.

After each mutation, validate the returned or subsequently read collection
against the expected effect. Primary and delete already return full collections;
create returns only an ID and requires a follow-up list. Stop when the response
contradicts the operation or introduces unexpected concurrent changes. Never
prune an additional, unreviewed ID discovered midway through apply.

Braze documents no compare-and-swap token or atomic replace-all operation. These
checks reduce races but cannot guarantee isolation from another writer.

## Errors and recovery

Reuse the current provider's shared HTTP transport, bounded throttling handling,
structured diagnostics, and payload-free operation logging. Do not replay an
ambiguous POST in the transport or continue with other mutations after it fails.
Do not infer ownership of an unknown created ID through approximate PEM matching.

On partial failure, return an error and retain every confirmed generated ID and
the best known partial state. Do not return the whole desired set as if it were
applied. A create response with an ID permits retaining that member's accepted
request values while verification is unavailable; primary flags affected by an
unverified promotion must be null rather than fabricated. A later successful
read replaces provisional values with observations. Persist no unknown values.
Do not roll back by deleting successfully created keys.

A failed initial Create can taint the aggregate. Because destruction retains
keys and Create rejects an occupied collection, a subsequent replacement is not
an automatic recovery mechanism. Inspect the remote keys and retained state,
resolve the original error, then deliberately untaint and refresh/apply, or
relinquish the state entry and import the app again. This explicit recovery
contract was accepted during design. Forced replacement of the same app likewise
does not mean "reset the keys"; use import to resume management.

After a lost create response, even an immediate empty list cannot prove that the
request will never take effect. Manual reconciliation is needed before a later
apply. The provider cannot promise exactly-once creation across process crashes
or separate applies without an API idempotency mechanism.

## Implementation and verification

1. Add the schema, model, app identity import, and fixture-backed Read/Delete.
2. Implement the pure reconciliation planner and independent transition tests.
3. Add a collection adapter around the existing generated SDK endpoints. Keep
   full mutation responses available to the executor; preserve singular-resource
   behavior when sharing API helpers. Do not force this aggregate through the
   singular resource's primary-deletion plan guard.
4. Implement Create/Update and partial-state recovery around that planner.
5. Add examples for initial setup, staged rotation, combined changes, ordinary
   and identity import, recovery, and retain-on-destroy. Generate Registry docs.

Acceptance tests must run actual Terraform against local HTTP fixtures and verify
remote results and requests, not only state attributes. Cover:

- Initial creation, no-op apply, both import forms, and arbitrary API ordering.
- Promotion and primary drift with existing IDs retained and no POST/DELETE.
- Member additions, removals, and changes to each immutable public field.
- Missing-key recreation and removal of unexpected remote members.
- Full-capacity ordering that succeeds in one apply without temporary promotion;
  known transitions requiring temporary promotion rejected during planning with
  zero writes; and the two-plan explicit-promotion sequence.
- Duplicate descriptions with different public material, ambiguous duplicate
  immutable pairs, duplicate response IDs, and zero/multiple primary responses.
- Retain-all destroy, app changes, nonempty-create rejection, and forced replacement.
- Failures after every mutation boundary, confirmed IDs surviving failed reads,
  create taint recovery, ambiguous POSTs, and concurrent changes before/during apply.
- Unknown configuration values and Terraform set correlation during primary flips.
- Ignored membership changes preserve observed state without writes, including
  observations that do not satisfy desired-collection constraints. Removing
  `ignore_changes` resumes normal reconciliation and preflight validation.
- App replacement with ignored observations uses configured keys in the new app
  and retains every key in the old app.
- Independent configuration errors do not suppress member validation or produce
  a false primary-selection diagnostic.

The planner tests should enumerate bounded collection transitions and compare
execution with an independent API-constraint oracle. Run the existing full Go
and mocked acceptance suites, current lint/format checks, and generation
consistency checks. The implementation includes both independent planner tests
and actual Terraform acceptance tests against local HTTP fixtures. These cover
16 failure boundaries across full-capacity updates, promotion-first updates, and
initial multi-key creation, including persisted partial state and explicit
recovery. Empty-collection and empty-description imports are also covered.

## Evidence and design review

Braze's [create](https://www.braze.com/docs/api/endpoints/sdk_authentication/post_create_sdk_authentication_key/),
[list](https://www.braze.com/docs/api/endpoints/sdk_authentication/get_sdk_authentication_keys/),
[primary](https://www.braze.com/docs/api/endpoints/sdk_authentication/put_primary_sdk_authentication_key/),
and [delete](https://www.braze.com/docs/api/endpoints/sdk_authentication/delete_sdk_authentication_key/)
documentation establish the per-app boundary, three-key limit, generated IDs,
response collections, and restriction on deleting the primary. The
[SDK guide](https://www.braze.com/docs/developer_guide/sdk_integration/authentication/)
distinguishes registration from authentication enforcement. There is no live
Braze validation of this plan; duplicate acceptance, key normalization, exact
error bodies, propagation, and concurrent behavior remain unverified.

[Auth0's public-key rotation guide](https://raw.githubusercontent.com/auth0/terraform-provider-auth0/main/docs/guides/client_secret_rotation.md)
demonstrates staged overlap within a plural credential configuration.
[AWS exclusive-policy management](https://raw.githubusercontent.com/hashicorp/terraform-provider-aws/main/website/docs/r/iam_role_policies_exclusive.html.markdown)
provides precedent for enforcing membership on apply and retaining objects when
management is destroyed. Neither establishes Braze behavior.

HashiCorp documents [nested sets](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes/set-nested),
[resource import](https://developer.hashicorp.com/terraform/plugin/framework/resources/import),
[plan modification](https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification),
[Create taint](https://developer.hashicorp.com/terraform/plugin/framework/resources/create),
and [partial Update state](https://developer.hashicorp.com/terraform/plugin/framework/resources/update).
The operation ordering and matching rules above are this provider's design,
derived from those contracts rather than promised by Terraform or Braze.

The design debate removed three sources of unnecessary machinery: enforced
rotation phases, artificial map labels with custom import mappings, and
configurable generated IDs used for adoption. The resulting set keeps exactly
one configured primary while making ordinary imports reconstructible from the
API. Explicit partial-create recovery remains necessary.

Independent API and Terraform-contract reviews found no blocker in the set
representation or partial-state approach. They identified two corrections:
qualify import coverage for potentially empty dashboard descriptions, and make
the temporary-primary policy explicit rather than calling that transition
API-infeasible. The final policy rejects plans requiring temporary promotion and
allows single-apply operation ordering otherwise. The implementation follows
these decisions without schema or lifecycle deviations. Independent implementation
reviews identified documentation and acceptance-coverage omissions that were
addressed. Subsequent regression coverage verifies Terraform's `ignore_changes`
behavior, and both resource adapters share structural collection validation.
