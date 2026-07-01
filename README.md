# workflow-plugin-encrypted-spaces

Workflow plugin for Encrypted Spaces collaboration primitives.

This plugin exposes Workflow module contracts for encrypted-space operation
storage, proof-verifier configuration, operation-log steps, epoch/member
updates, and production-mode proof verification reports. It does not provide
live Signal service egress.

## Installation

```sh
wfctl plugin install workflow-plugin-encrypted-spaces
```

## Development

```sh
# Build
make build

# Test
make test

# Install locally
make install-local
```

## Release Stages

- `v0.1.0`: append, fast-forward, epoch, member-update, trigger, and audit
  primitives backed by `encrypted-spaces-go` fake/no-proof verification.
- `v0.2.0`: production proof verification primitives backed by upstream
  vector-tested proof ports.
- `v0.3.0`: vector coverage report step backed by `encrypted-spaces-go v0.3.0`;
  backup and SVR proof domains remain explicitly deferred and block production
  equivalence claims.
- `v0.4.0`: proof policy, verified append, and redacted proof-evidence
  workflow contracts backed by `encrypted-spaces-go v0.4.0`, with a
  rooms/eventbus/audit composition scenario for private collaboration apps.

Fake/no-proof modes are for application composition tests and conformance
harnesses only. Production deployments should require proof reports whose
`ProductionReady` field is true.

## Module Types

- `encrypted_space.store`
- `encrypted_space.verifier`
- `encrypted_space.proof_policy`

## Step Types

- `step.encrypted_space_append`
- `step.encrypted_space_fast_forward`
- `step.encrypted_space_epoch_rotate`
- `step.encrypted_space_member_update`
- `step.encrypted_space_verify_membership`
- `step.encrypted_space_verify_operation`
- `step.encrypted_space_verify_checkpoint`
- `step.encrypted_space_vector_report`
- `step.encrypted_space_append_verified`
- `step.encrypted_space_proof_evidence`

`step.encrypted_space_vector_report` returns per-domain coverage rows with
`vector-backed`, `deterministic-only`, or `deferred` status values. When
`require_production_equivalence` is true, the step fails if any required domain
is not vector-backed.

`step.encrypted_space_append_verified` verifies operation commitments,
membership reports, and checkpoint reports before accepting an append. The
`encrypted_space.proof_policy` module can require vector-backed proof evidence
and keeps message-backup and SVR/SVRB proof domains explicitly deferred until
upstream vectors are available.

`step.encrypted_space_proof_evidence` emits redacted proof evidence and an audit
event payload shaped for `workflow-plugin-audit`; plaintext and key material are
not copied into evidence output.

## Scenarios

- `scenarios/encrypted-space-proof-workflow`: local proof-gated append flow that
  composes this plugin with released `workflow-plugin-rooms`,
  `workflow-plugin-eventbus`, and `workflow-plugin-audit` pins.

## Module

Go module: `github.com/GoCodeAlone/workflow-plugin-encrypted-spaces`
