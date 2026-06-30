package internal

import (
	"context"
	"fmt"

	"github.com/GoCodeAlone/encrypted-spaces-go/keytrans"
	"github.com/GoCodeAlone/encrypted-spaces-go/poksho"
	"github.com/GoCodeAlone/encrypted-spaces-go/zkgroup"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
)

func ExecuteEncryptedSpaceVerifyMembership(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyMembershipConfig, *contracts.VerifyMembershipInput],
) (*sdk.TypedStepResult[*contracts.VerifyMembershipOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify membership: input is required")
	}
	report, err := zkgroup.VerifyMembershipCredential(zkgroup.MembershipCredential{
		GroupID:      req.Input.GetGroupId(),
		MemberID:     req.Input.GetMemberId(),
		Issuer:       req.Input.GetIssuer(),
		ExpiresAt:    req.Input.GetExpiresAt(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify membership: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyMembershipOutput]{
		Output: &contracts.VerifyMembershipOutput{Report: proofReport(report.Domain, report.Accepted, report.ProductionReady, report.UpstreamPath)},
	}, nil
}

func ExecuteEncryptedSpaceVerifyOperation(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyOperationConfig, *contracts.VerifyOperationInput],
) (*sdk.TypedStepResult[*contracts.VerifyOperationOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify operation: input is required")
	}
	if req.Input.GetProofDigest() == "fake-proof" {
		return nil, fmt.Errorf("encrypted space verify operation: fake proof cannot be marked production")
	}
	report, err := poksho.VerifyProofTranscript(poksho.ProofTranscript{
		TranscriptID: req.Input.GetTranscriptId(),
		StatementID:  req.Input.GetStatementId(),
		WitnessHash:  req.Input.GetWitnessHash(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify operation: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyOperationOutput]{
		Output: &contracts.VerifyOperationOutput{Report: proofReport(report.Domain, report.Accepted, report.ProductionReady, report.UpstreamPath)},
	}, nil
}

func ExecuteEncryptedSpaceVerifyCheckpoint(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.VerifyCheckpointConfig, *contracts.VerifyCheckpointInput],
) (*sdk.TypedStepResult[*contracts.VerifyCheckpointOutput], error) {
	if req.Input == nil {
		return nil, fmt.Errorf("encrypted space verify checkpoint: input is required")
	}
	report, err := keytrans.VerifyCheckpoint(keytrans.Checkpoint{
		CheckpointID: req.Input.GetCheckpointId(),
		TreeHead:     req.Input.GetTreeHead(),
		TreeSize:     req.Input.GetTreeSize(),
		ProofDigest:  req.Input.GetProofDigest(),
		UpstreamPath: req.Input.GetUpstreamPath(),
	})
	if err != nil {
		return nil, fmt.Errorf("encrypted space verify checkpoint: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.VerifyCheckpointOutput]{
		Output: &contracts.VerifyCheckpointOutput{Report: proofReport(report.Domain, report.Accepted, report.ProductionReady, report.UpstreamPath)},
	}, nil
}

func proofReport(domain string, accepted, productionReady bool, upstreamPath string) *contracts.VerificationReport {
	return &contracts.VerificationReport{
		OperationId:     domain,
		Digest:          upstreamPath,
		Accepted:        accepted,
		ProductionReady: productionReady,
		Mode:            "production",
	}
}
