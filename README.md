# workflow-plugin-encrypted-spaces

Workflow plugin for Encrypted Spaces collaboration primitives.

This plugin exposes Workflow module contracts for encrypted-space operation
storage and proof-verifier configuration. The initial scaffold intentionally
does not provide production proof verification or live service egress.

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

Fake/no-proof modes are for application composition tests and conformance
harnesses only. Production deployments should require proof reports whose
`ProductionReady` field is true.

## Module Types

- `encrypted_space.store`
- `encrypted_space.verifier`

## Step Types

- `step.encrypted_space_append`
- `step.encrypted_space_fast_forward`
- `step.encrypted_space_epoch_rotate`
- `step.encrypted_space_member_update`

## Module

Go module: `github.com/GoCodeAlone/workflow-plugin-encrypted-spaces`
