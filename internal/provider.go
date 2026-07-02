package internal

import (
	"context"
	"fmt"

	contracts "github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// Version is injected by the release build so runtime manifests report the tag.
var Version = "dev"

// EncryptedSpacesProvider implements sdk.PluginProvider, sdk.TypedModuleProvider,
// sdk.TypedStepProvider, and sdk.ContractProvider.
type EncryptedSpacesProvider struct{}

// NewEncryptedSpacesProvider creates a new EncryptedSpacesProvider.
func NewEncryptedSpacesProvider() *EncryptedSpacesProvider {
	return &EncryptedSpacesProvider{}
}

// Manifest implements sdk.PluginProvider.
func (p *EncryptedSpacesProvider) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-encrypted-spaces",
		Version:     sdk.ResolveBuildVersion(Version),
		Author:      "GoCodeAlone",
		Description: "Workflow plugin for Encrypted Spaces collaboration primitives",
	}
}

var encryptedSpacesModuleTypes = []string{
	"encrypted_space.proof_policy",
	"encrypted_space.state_store",
	"encrypted_space.store",
	"encrypted_space.verifier",
}

var encryptedSpacesStepTypes = []string{
	"step.encrypted_space_append",
	"step.encrypted_space_append_verified",
	"step.encrypted_space_fast_forward",
	"step.encrypted_space_epoch_rotate",
	"step.encrypted_space_member_update",
	"step.encrypted_space_proof_evidence",
	"step.encrypted_space_state_init",
	"step.encrypted_space_state_load",
	"step.encrypted_space_state_save",
	"step.encrypted_space_state_update",
	"step.encrypted_space_member_check",
	"step.encrypted_space_verify_membership",
	"step.encrypted_space_verify_operation",
	"step.encrypted_space_verify_checkpoint",
	"step.encrypted_space_vector_report",
}

// TypedModuleTypes implements sdk.TypedModuleProvider.
func (p *EncryptedSpacesProvider) TypedModuleTypes() []string {
	return append([]string(nil), encryptedSpacesModuleTypes...)
}

// CreateTypedModule implements sdk.TypedModuleProvider.
func (p *EncryptedSpacesProvider) CreateTypedModule(typeName, name string, config *anypb.Any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "encrypted_space.proof_policy":
		factory := sdk.NewTypedModuleFactory(typeName, &contracts.ProofPolicyConfig{}, func(name string, cfg *contracts.ProofPolicyConfig) (sdk.ModuleInstance, error) {
			return &spacesModule{name: name}, nil
		})
		return factory.CreateTypedModule(typeName, name, config)
	case "encrypted_space.state_store":
		factory := sdk.NewTypedModuleFactory(typeName, &contracts.StateStoreConfig{}, func(name string, cfg *contracts.StateStoreConfig) (sdk.ModuleInstance, error) {
			return newEncryptedSpaceStateStoreModuleFromConfig(name, cfg)
		})
		return factory.CreateTypedModule(typeName, name, config)
	case "encrypted_space.store":
		factory := sdk.NewTypedModuleFactory(typeName, &contracts.SpaceStoreConfig{}, func(name string, cfg *contracts.SpaceStoreConfig) (sdk.ModuleInstance, error) {
			return &spacesModule{name: name}, nil
		})
		return factory.CreateTypedModule(typeName, name, config)
	case "encrypted_space.verifier":
		factory := sdk.NewTypedModuleFactory(typeName, &contracts.VerifierConfig{}, func(name string, cfg *contracts.VerifierConfig) (sdk.ModuleInstance, error) {
			return &spacesModule{name: name}, nil
		})
		return factory.CreateTypedModule(typeName, name, config)
	}
	return nil, fmt.Errorf("%w: module type %q", sdk.ErrTypedContractNotHandled, typeName)
}

// TypedStepTypes implements sdk.TypedStepProvider.
func (p *EncryptedSpacesProvider) TypedStepTypes() []string {
	return append([]string(nil), encryptedSpacesStepTypes...)
}

// CreateTypedStep implements sdk.TypedStepProvider.
func (p *EncryptedSpacesProvider) CreateTypedStep(typeName, name string, config *anypb.Any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.encrypted_space_append":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.AppendConfig{},
			&contracts.AppendInput{},
			ExecuteEncryptedSpaceAppend,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_append_verified":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.AppendVerifiedConfig{},
			&contracts.AppendVerifiedInput{},
			ExecuteEncryptedSpaceAppendVerified,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_fast_forward":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.FastForwardConfig{},
			&contracts.FastForwardInput{},
			ExecuteEncryptedSpaceFastForward,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_epoch_rotate":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.EpochRotateConfig{},
			&contracts.EpochRotateInput{},
			ExecuteEncryptedSpaceEpochRotate,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_member_update":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.MemberUpdateConfig{},
			&contracts.MemberUpdateInput{},
			ExecuteEncryptedSpaceMemberUpdate,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_proof_evidence":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.ProofEvidenceConfig{},
			&contracts.ProofEvidenceInput{},
			ExecuteEncryptedSpaceProofEvidence,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_state_init":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.StateInitConfig{},
			&contracts.StateInitInput{},
			ExecuteEncryptedSpaceStateInit,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_state_load":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.StateLoadConfig{},
			&contracts.StateLoadInput{},
			ExecuteEncryptedSpaceStateLoad,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_state_save":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.StateSaveConfig{},
			&contracts.StateSaveInput{},
			ExecuteEncryptedSpaceStateSave,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_state_update":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.StateUpdateConfig{},
			&contracts.StateUpdateInput{},
			ExecuteEncryptedSpaceStateUpdate,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_member_check":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.MemberCheckConfig{},
			&contracts.MemberCheckInput{},
			ExecuteEncryptedSpaceMemberCheck,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_verify_membership":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.VerifyMembershipConfig{},
			&contracts.VerifyMembershipInput{},
			ExecuteEncryptedSpaceVerifyMembership,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_verify_operation":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.VerifyOperationConfig{},
			&contracts.VerifyOperationInput{},
			ExecuteEncryptedSpaceVerifyOperation,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_verify_checkpoint":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.VerifyCheckpointConfig{},
			&contracts.VerifyCheckpointInput{},
			ExecuteEncryptedSpaceVerifyCheckpoint,
		)
		return factory.CreateTypedStep(typeName, name, config)
	case "step.encrypted_space_vector_report":
		factory := sdk.NewTypedStepFactory(
			typeName,
			&contracts.VectorReportConfig{},
			&contracts.VectorReportInput{},
			ExecuteEncryptedSpaceVectorReport,
		)
		return factory.CreateTypedStep(typeName, name, config)
	}
	return nil, fmt.Errorf("%w: step type %q", sdk.ErrTypedContractNotHandled, typeName)
}

// ContractRegistry implements sdk.ContractProvider.
func (p *EncryptedSpacesProvider) ContractRegistry() *pb.ContractRegistry {
	const pkg = "workflow.plugins.encryptedspaces.v1."
	return &pb.ContractRegistry{
		FileDescriptorSet: &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{
				protodesc.ToFileDescriptorProto(contracts.File_internal_contracts_spaces_proto),
			},
		},
		Contracts: []*pb.ContractDescriptor{
			moduleContract("encrypted_space.proof_policy", pkg+"ProofPolicyConfig"),
			moduleContract("encrypted_space.state_store", pkg+"StateStoreConfig"),
			moduleContract("encrypted_space.store", pkg+"SpaceStoreConfig"),
			moduleContract("encrypted_space.verifier", pkg+"VerifierConfig"),
			stepContract("step.encrypted_space_append", pkg+"AppendConfig", pkg+"AppendInput", pkg+"AppendOutput"),
			stepContract("step.encrypted_space_append_verified", pkg+"AppendVerifiedConfig", pkg+"AppendVerifiedInput", pkg+"AppendVerifiedOutput"),
			stepContract("step.encrypted_space_fast_forward", pkg+"FastForwardConfig", pkg+"FastForwardInput", pkg+"FastForwardOutput"),
			stepContract("step.encrypted_space_epoch_rotate", pkg+"EpochRotateConfig", pkg+"EpochRotateInput", pkg+"EpochRotateOutput"),
			stepContract("step.encrypted_space_member_update", pkg+"MemberUpdateConfig", pkg+"MemberUpdateInput", pkg+"MemberUpdateOutput"),
			stepContract("step.encrypted_space_proof_evidence", pkg+"ProofEvidenceConfig", pkg+"ProofEvidenceInput", pkg+"ProofEvidenceOutput"),
			stepContract("step.encrypted_space_state_init", pkg+"StateInitConfig", pkg+"StateInitInput", pkg+"StateInitOutput"),
			stepContract("step.encrypted_space_state_load", pkg+"StateLoadConfig", pkg+"StateLoadInput", pkg+"StateLoadOutput"),
			stepContract("step.encrypted_space_state_save", pkg+"StateSaveConfig", pkg+"StateSaveInput", pkg+"StateSaveOutput"),
			stepContract("step.encrypted_space_state_update", pkg+"StateUpdateConfig", pkg+"StateUpdateInput", pkg+"StateUpdateOutput"),
			stepContract("step.encrypted_space_member_check", pkg+"MemberCheckConfig", pkg+"MemberCheckInput", pkg+"MemberCheckOutput"),
			stepContract("step.encrypted_space_verify_membership", pkg+"VerifyMembershipConfig", pkg+"VerifyMembershipInput", pkg+"VerifyMembershipOutput"),
			stepContract("step.encrypted_space_verify_operation", pkg+"VerifyOperationConfig", pkg+"VerifyOperationInput", pkg+"VerifyOperationOutput"),
			stepContract("step.encrypted_space_verify_checkpoint", pkg+"VerifyCheckpointConfig", pkg+"VerifyCheckpointInput", pkg+"VerifyCheckpointOutput"),
			stepContract("step.encrypted_space_vector_report", pkg+"VectorReportConfig", pkg+"VectorReportInput", pkg+"VectorReportOutput"),
		},
	}
}

type spacesModule struct {
	name string
}

func (m *spacesModule) Init() error {
	return nil
}

func (m *spacesModule) Start(ctx context.Context) error {
	return nil
}

func (m *spacesModule) Stop(ctx context.Context) error {
	return nil
}

func moduleContract(moduleType, configMessage string) *pb.ContractDescriptor {
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_MODULE,
		ModuleType:    moduleType,
		ConfigMessage: configMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}

func stepContract(stepType, configMessage, inputMessage, outputMessage string) *pb.ContractDescriptor {
	return &pb.ContractDescriptor{
		Kind:          pb.ContractKind_CONTRACT_KIND_STEP,
		StepType:      stepType,
		ConfigMessage: configMessage,
		InputMessage:  inputMessage,
		OutputMessage: outputMessage,
		Mode:          pb.ContractMode_CONTRACT_MODE_STRICT_PROTO,
	}
}
