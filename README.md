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

## Capability Surface

The current release exposes operation-log, epoch/member, space-state lifecycle,
memory and explicit file-backed state custody, proof-report, proof-policy,
verified-append, and redacted proof-evidence primitives backed by
`encrypted-spaces-go`. It also includes a
rooms/eventbus/audit composition scenario for private collaboration apps.

Fake/no-proof modes are for application composition tests and conformance
harnesses only. Production deployments should require proof reports whose
`ProductionReady` field is true.

## Module Types

- `encrypted_space.store`
- `encrypted_space.verifier`
- `encrypted_space.proof_policy`
- `encrypted_space.state_store`

## Step Types

- `step.encrypted_space_append`
- `step.encrypted_space_fast_forward`
- `step.encrypted_space_epoch_rotate`
- `step.encrypted_space_member_update`
- `step.encrypted_space_state_init`
- `step.encrypted_space_state_load`
- `step.encrypted_space_state_save`
- `step.encrypted_space_state_update`
- `step.encrypted_space_member_check`
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

`step.encrypted_space_state_init`, `step.encrypted_space_state_update`, and
`step.encrypted_space_member_check` expose `SpaceState` snapshots for Workflow
applications that need to enroll, revoke, and check members before append/proof
flows. `encrypted_space.state_store` with
`step.encrypted_space_state_load`/`step.encrypted_space_state_save` provides
named snapshot custody for application composition and scenario proofs. The
default `memory` backend is ephemeral. The explicit `file` backend writes a
schema-versioned and checksummed state snapshot through a temp-file plus rename
sequence, rejects corrupt or unsupported snapshots on start, and stores only
space IDs, epochs, members, and removed members. It does not store plaintext,
proof secrets, private keys, ciphertext bodies, or operation-log payloads.
Production policy mode rejects the local `file` backend; production hosts
should bind this contract to their managed storage boundary.

## Scenarios

- `scenarios/encrypted-space-proof-workflow`: local proof-gated append flow that
  composes this plugin with released `workflow-plugin-rooms`,
  `workflow-plugin-eventbus`, and `workflow-plugin-audit` pins.
- `testdata/pipelines/encrypted-space-file-state.yaml`: local file-state
  workflow fixture that initializes a caller-supplied space, saves state,
  reloads it, and rejects a removed member.

## Module

Go module: `github.com/GoCodeAlone/workflow-plugin-encrypted-spaces`
