# Provider releases

The [release workflow](../.github/workflows/release.yml) runs on pushes of `v*`
tags. It requires an existing published GitHub release, with both its name and
tag matching the pushed tag. Drafts and prereleases are rejected before the
signing key is imported.

## Repository setup

Configure the `release` GitHub environment and its protection rules before the
first release using this workflow. Referencing an environment in a workflow does
not configure reviewer or tag restrictions. The workflow uses the repository's
existing `GPG_PRIVATE_KEY` and `PASSPHRASE` secret names; these can instead be
provided as environment secrets with the same names. The signing key must match
the public key registered for this provider in the Terraform Registry.

## Publish a version

1. Select the reviewed commit and confirm its required CI checks passed. Prepare
   release notes describing practitioner-visible changes and any upgrade steps.
2. Create a published GitHub release for a new `vX.Y.Z` tag at that exact commit,
   with the title `vX.Y.Z`. Creating the release and new tag together ensures the
   release exists when the tag-triggered workflow checks it. Do not push a tag
   first and expect the workflow to create its release.
3. Review the `release` environment approval if one is configured, then follow
   the release workflow run through completion.
4. Confirm the version becomes available in the Terraform Registry. A successful
   GitHub workflow verifies the uploaded release assets; it does not establish
   Registry ingestion or successful installation.

[GoReleaser](../.goreleaser.yml), pinned to v2.18.0 in the workflow, builds the
provider archives, signs the checksum file, and uploads the archives, Registry
manifest, checksums, and signature to the existing release. The workflow verifies
the checksums and GPG signature, attests all four artifact types, and compares the
published asset count and SHA-256 digests with the local files. Release jobs are
serialized.

## Failed runs and local checks

A published release can remain incomplete if the workflow fails. Inspect the
failure before rerunning the same workflow; reruns replace matching artifacts
and repeat verification. Keep the tag at its original reviewed commit. This
process requires a release that permits asset updates; enabling immutable
releases requires changing the publishing process first.

Validate changes with the same GoReleaser version used by CI:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish,sign
```

The snapshot command builds and packages all configured platforms without
publishing or accessing the signing key. Run it in a disposable checkout: `--clean`
replaces its `dist/` directory. A snapshot does not test GitHub permissions,
environment protections, signing, attestations, or Registry ingestion.
