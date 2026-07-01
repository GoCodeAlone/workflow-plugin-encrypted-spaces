package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoCodeAlone/encrypted-spaces-go/keytrans"
	"github.com/GoCodeAlone/encrypted-spaces-go/operationlog"
	spacesproof "github.com/GoCodeAlone/encrypted-spaces-go/proof"
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
	if req.Input.GetExpectedCommitment() == nil {
		return nil, fmt.Errorf("encrypted space append verified: expected commitment is required")
	}
	policy := spacesproof.VectorPolicy()
	operation := operationFromContract(req.Input.GetOperation())
	expected := commitmentFromContract(req.Input.GetExpectedCommitment())
	if _, err := policy.VerifyOperationCommitment(operation, expected); err != nil {
		return nil, fmt.Errorf("encrypted space append verified: %w", err)
	}

	reports := []spacesproof.Report{{
		Domain:          "operationlog.commitment",
		Accepted:        true,
		ProductionReady: true,
	}}
	if membership := req.Input.GetMembership(); membership != nil {
		report, err := policy.VerifyMembership(spacesproof.MembershipProof{
			GroupID:      membership.GetGroupId(),
			MemberID:     membership.GetMemberId(),
			Issuer:       membership.GetIssuer(),
			ExpiresAt:    membership.GetExpiresAt(),
			ProofDigest:  membership.GetProofDigest(),
			UpstreamPath: membership.GetUpstreamPath(),
		})
		if err != nil {
			return nil, fmt.Errorf("encrypted space append verified: membership proof: %w", err)
		}
		reports = append(reports, report)
	}
	if checkpoint := req.Input.GetCheckpoint(); checkpoint != nil {
		report, err := policy.VerifyCheckpoint(spacesproof.CheckpointProof{
			Checkpoint: keytrans.Checkpoint{
				CheckpointID: checkpoint.GetCheckpointId(),
				TreeHead:     checkpoint.GetTreeHead(),
				TreeSize:     checkpoint.GetTreeSize(),
				ProofDigest:  checkpoint.GetProofDigest(),
				UpstreamPath: checkpoint.GetUpstreamPath(),
			},
			PreviousTreeSize: checkpoint.GetPreviousTreeSize(),
		})
		if err != nil {
			return nil, fmt.Errorf("encrypted space append verified: checkpoint proof: %w", err)
		}
		reports = append(reports, report)
	}
	if req.Config.GetPolicy().GetRequireVectorBacked() {
		for _, report := range reports {
			if !report.ProductionReady {
				return nil, fmt.Errorf("encrypted space append verified: proof report %s is not production ready", report.Domain)
			}
		}
	}

	evidence := spacesproof.NewOperationEvidence(expected, reports)
	audit, err := auditEventJSON("encrypted_space.append_verified", expected, reports)
	if err != nil {
		return nil, fmt.Errorf("encrypted space append verified: audit payload: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.AppendVerifiedOutput]{
		Output: &contracts.AppendVerifiedOutput{
			Commitment:     commitmentToContract(expected),
			Reports:        proofReportsToContract(reports),
			Evidence:       evidenceToContract(evidence),
			AuditEventJson: audit,
		},
	}, nil
}

func ExecuteEncryptedSpaceProofEvidence(
	_ context.Context,
	req sdk.TypedStepRequest[*contracts.ProofEvidenceConfig, *contracts.ProofEvidenceInput],
) (*sdk.TypedStepResult[*contracts.ProofEvidenceOutput], error) {
	if req.Input == nil || req.Input.GetCommitment() == nil {
		return nil, fmt.Errorf("encrypted space proof evidence: commitment is required")
	}
	commitment := commitmentFromContract(req.Input.GetCommitment())
	reports := proofReportsFromContract(req.Input.GetReports())
	evidence := spacesproof.NewOperationEvidence(commitment, reports)
	raw, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("encrypted space proof evidence: marshal evidence: %w", err)
	}
	audit, err := auditEventJSON("encrypted_space.proof_evidence", commitment, reports)
	if err != nil {
		return nil, fmt.Errorf("encrypted space proof evidence: audit payload: %w", err)
	}
	return &sdk.TypedStepResult[*contracts.ProofEvidenceOutput]{
		Output: &contracts.ProofEvidenceOutput{
			Evidence:       evidenceToContract(evidence),
			Json:           string(raw),
			AuditEventJson: audit,
		},
	}, nil
}

func commitmentFromContract(commitment *contracts.OperationCommitment) operationlog.OperationCommitment {
	if commitment == nil {
		return operationlog.OperationCommitment{}
	}
	return operationlog.OperationCommitment{
		SpaceID:         operationlog.SpaceID(commitment.GetSpaceId()),
		OperationID:     operationlog.OperationID(commitment.GetOperationId()),
		Sequence:        commitment.GetSequence(),
		Digest:          commitment.GetDigest(),
		KeyEpoch:        operationlog.KeyEpoch(commitment.GetKeyEpoch()),
		MembershipEpoch: operationlog.MembershipEpoch(commitment.GetMembershipEpoch()),
		CiphertextSize:  int(commitment.GetCiphertextSize()),
	}
}

func proofReportsToContract(reports []spacesproof.Report) []*contracts.ProofReport {
	out := make([]*contracts.ProofReport, 0, len(reports))
	for _, report := range reports {
		out = append(out, &contracts.ProofReport{
			Domain:          report.Domain,
			Accepted:        report.Accepted,
			ProductionReady: report.ProductionReady,
			UpstreamPath:    report.UpstreamPath,
		})
	}
	return out
}

func proofReportsFromContract(reports []*contracts.ProofReport) []spacesproof.Report {
	out := make([]spacesproof.Report, 0, len(reports))
	for _, report := range reports {
		out = append(out, spacesproof.Report{
			Domain:          report.GetDomain(),
			Accepted:        report.GetAccepted(),
			ProductionReady: report.GetProductionReady(),
			UpstreamPath:    report.GetUpstreamPath(),
		})
	}
	return out
}

func evidenceToContract(evidence spacesproof.OperationEvidence) *contracts.ProofEvidence {
	return &contracts.ProofEvidence{
		SpaceId:         string(evidence.SpaceID),
		OperationId:     string(evidence.OperationID),
		Sequence:        evidence.Sequence,
		Digest:          evidence.Digest,
		KeyEpoch:        uint64(evidence.KeyEpoch),
		MembershipEpoch: uint64(evidence.MembershipEpoch),
		CiphertextSize:  uint64(evidence.CiphertextSize),
		Reports:         proofReportsToContract(evidence.Reports),
	}
}

func auditEventJSON(eventType string, commitment operationlog.OperationCommitment, reports []spacesproof.Report) (string, error) {
	payload := struct {
		EventType   string   `json:"event_type"`
		Plugin      string   `json:"plugin"`
		OperationID string   `json:"operation_id"`
		Digest      string   `json:"digest"`
		Domains     []string `json:"domains"`
		Timestamp   string   `json:"timestamp"`
	}{
		EventType:   eventType,
		Plugin:      "workflow-plugin-encrypted-spaces",
		OperationID: string(commitment.OperationID),
		Digest:      commitment.Digest,
		Domains:     make([]string, 0, len(reports)),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, report := range reports {
		payload.Domains = append(payload.Domains, report.Domain)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
