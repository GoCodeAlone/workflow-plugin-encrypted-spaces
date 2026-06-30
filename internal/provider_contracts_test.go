package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	"github.com/GoCodeAlone/workflow/plugin/external/sdk"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestEncryptedSpacesProviderDeclaresStrictScaffoldContracts(t *testing.T) {
	provider := NewEncryptedSpacesProvider()
	moduleProvider, ok := any(provider).(sdk.TypedModuleProvider)
	if !ok {
		t.Fatal("expected typed module provider")
	}
	stepProvider, ok := any(provider).(sdk.TypedStepProvider)
	if !ok {
		t.Fatal("expected typed step provider")
	}
	contractProvider, ok := any(provider).(sdk.ContractProvider)
	if !ok {
		t.Fatal("expected contract provider")
	}

	assertStringSet(t, moduleProvider.TypedModuleTypes(), []string{
		"encrypted_space.store",
		"encrypted_space.verifier",
	})
	assertStringSet(t, stepProvider.TypedStepTypes(), []string{
		"step.encrypted_space_append",
		"step.encrypted_space_fast_forward",
		"step.encrypted_space_epoch_rotate",
		"step.encrypted_space_member_update",
	})

	registry := contractProvider.ContractRegistry()
	if registry == nil {
		t.Fatal("expected contract registry")
	}
	if registry.FileDescriptorSet == nil || len(registry.FileDescriptorSet.File) == 0 {
		t.Fatal("expected descriptor set")
	}
	files, err := protodesc.NewFiles(registry.FileDescriptorSet)
	if err != nil {
		t.Fatalf("descriptor set: %v", err)
	}

	contractsByKey := map[string]*pb.ContractDescriptor{}
	for _, descriptor := range registry.Contracts {
		if descriptor.Mode != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %s, want strict proto", contractKey(descriptor), descriptor.Mode)
		}
		switch descriptor.Kind {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByKey["module:"+descriptor.ModuleType] = descriptor
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByKey["step:"+descriptor.StepType] = descriptor
		default:
			t.Fatalf("%s kind = %s, want module or step", contractKey(descriptor), descriptor.Kind)
		}
		for _, name := range []string{descriptor.ConfigMessage, descriptor.InputMessage, descriptor.OutputMessage} {
			if name == "" {
				continue
			}
			if _, err := files.FindDescriptorByName(protoreflect.FullName(name)); err != nil {
				t.Fatalf("%s references unknown message %s: %v", contractKey(descriptor), name, err)
			}
		}
	}

	for _, key := range []string{
		"module:encrypted_space.store",
		"module:encrypted_space.verifier",
		"step:step.encrypted_space_append",
		"step:step.encrypted_space_fast_forward",
		"step:step.encrypted_space_epoch_rotate",
		"step:step.encrypted_space_member_update",
	} {
		if _, ok := contractsByKey[key]; !ok {
			t.Fatalf("missing contract %s", key)
		}
	}
}

func TestPluginJSONCapabilitiesMatchRuntimeProvider(t *testing.T) {
	provider := NewEncryptedSpacesProvider()
	moduleProvider := any(provider).(sdk.TypedModuleProvider)
	stepProvider := any(provider).(sdk.TypedStepProvider)

	var manifest struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		MinEngine    string `json:"minEngineVersion"`
		Capabilities struct {
			ModuleTypes  []string `json:"moduleTypes"`
			StepTypes    []string `json:"stepTypes"`
			TriggerTypes []string `json:"triggerTypes"`
		} `json:"capabilities"`
	}
	raw, err := os.ReadFile(filepath.Join("..", "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode plugin.json: %v", err)
	}
	if manifest.Name != "workflow-plugin-encrypted-spaces" {
		t.Fatalf("name = %q, want workflow-plugin-encrypted-spaces", manifest.Name)
	}
	if manifest.Type != "external" {
		t.Fatalf("type = %q, want external", manifest.Type)
	}
	if manifest.MinEngine == "" {
		t.Fatal("minEngineVersion is empty")
	}
	assertStringSet(t, manifest.Capabilities.ModuleTypes, moduleProvider.TypedModuleTypes())
	assertStringSet(t, manifest.Capabilities.StepTypes, stepProvider.TypedStepTypes())
	assertStringSet(t, manifest.Capabilities.TriggerTypes, nil)
}

func contractKey(descriptor *pb.ContractDescriptor) string {
	switch descriptor.Kind {
	case pb.ContractKind_CONTRACT_KIND_MODULE:
		return "module:" + descriptor.ModuleType
	case pb.ContractKind_CONTRACT_KIND_STEP:
		return "step:" + descriptor.StepType
	default:
		return descriptor.Kind.String()
	}
}

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()
	seen := make(map[string]int, len(got))
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		if seen[value] != 1 {
			t.Fatalf("values = %v, want exactly one %q", got, value)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("values = %v, want %v", got, want)
	}
}
