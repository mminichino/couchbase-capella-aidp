# Releasing to the Terraform Registry

This provider publishes signed release assets with GoReleaser. The Terraform Registry
pulls those assets from GitHub Releases under `mminichino/couchbase-capella-aidp`.

## One-time setup

1. **GPG signing key** for release checksums:
   - Generate a GPG key (or reuse an existing one).
   - Export the private key and add GitHub repository secrets:
     - `GPG_PRIVATE_KEY` — armored private key
     - `PASSPHRASE` — key passphrase (empty string if none)
2. **Publish the provider** on [registry.terraform.io](https://registry.terraform.io/publish/provider):
   - Namespace: `mminichino`
   - Provider name: `couchbase-capella-aidp`
   - Link the GitHub repository `mminichino/couchbase-capella-aidp`
3. **Acceptance test secrets** (optional, for CI):
   - `CAPELLA_AUTHENTICATION_TOKEN`
   - `CAPELLA_ORGANIZATION_ID`
   - `CAPELLA_HOST` (optional; defaults to production)
   - `CAPELLA_PROJECT_ID` / `CAPELLA_CLUSTER_ID` (for workflow tests)
   - `OPENAI_API_KEY` (for AIDP provider resource tests)
   - `CAPELLA_MODEL_REGION` (for model API key tests, e.g. `us-east-1`)
   - Repository variable `ACC_ENABLE_MODEL=true` to enable expensive model deploy tests

## Cut a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

The `Release` GitHub Action runs GoReleaser, creates a GitHub Release with:

- platform zip archives
- `SHA256SUMS` (+ detached GPG signature)
- `terraform-registry-manifest.json`

Terraform Registry will pick up the new version automatically once the provider listing is published.
