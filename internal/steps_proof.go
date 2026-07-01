package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func ExecuteEncryptedSpaceAppendVerified(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.AppendVerifiedConfig, *contracts.AppendVerifiedInput],
) (*sdk.TypedStepResult[*contracts.AppendVerifiedOutput], error) {
	if req.Input == nil || req.Input.GetOperation() == nil {
		return nil, fmt.Errorf("encrypted space append verified: operation is required")
	}
	return nil, fmt.Errorf("encrypted space append verified: proof execution is not implemented")
}

func ExecuteEncryptedSpaceProofEvidence(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.ProofEvidenceConfig, *contracts.ProofEvidenceInput],
) (*sdk.TypedStepResult[*contracts.ProofEvidenceOutput], error) {
	if req.Input == nil || req.Input.GetCommitment() == nil {
		return nil, fmt.Errorf("encrypted space proof evidence: commitment is required")
	}
	return nil, fmt.Errorf("encrypted space proof evidence: proof execution is not implemented")
}
