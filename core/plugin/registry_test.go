package plugin_test

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/builtwithtofu/sigil/core/plugin/mockplugin"
	"github.com/google/go-cmp/cmp"
)

func TestRegistryRegisterAndLookup(t *testing.T) {
	registry := plugin.NewRegistry()
	awsPlugin := &testAWSPlugin{}

	if err := registry.Register(awsPlugin); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	capability := registry.Lookup("aws.secrets")
	if capability == nil {
		t.Fatal("Lookup() = nil, want capability")
	}
	if diff := cmp.Diff(plugin.KindValue, capability.Kind()); diff != "" {
		t.Fatalf("Lookup() kind mismatch (-want +got):\n%s", diff)
	}

	transport := registry.Lookup("aws.instance.connect")
	if transport == nil {
		t.Fatal("Lookup() transport = nil, want capability")
	}
	if diff := cmp.Diff(plugin.KindTransport, transport.Kind()); diff != "" {
		t.Fatalf("Lookup() transport kind mismatch (-want +got):\n%s", diff)
	}
}

func TestRegistryListNamespace(t *testing.T) {
	registry := plugin.NewRegistry()
	if err := registry.Register(&testAWSPlugin{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	paths := registry.ListNamespace("aws")
	if diff := cmp.Diff([]string{"aws.instance.connect", "aws.secrets"}, paths); diff != "" {
		t.Fatalf("ListNamespace() mismatch (-want +got):\n%s", diff)
	}
}

func TestRegistryRejectsDuplicateNamespace(t *testing.T) {
	registry := plugin.NewRegistry()
	if err := registry.Register(&testAWSPlugin{}); err != nil {
		t.Fatalf("Register() first error = %v", err)
	}
	if err := registry.Register(&testAWSPlugin{}); err == nil {
		t.Fatal("Register() second error = nil, want duplicate namespace error")
	}
}

type testAWSPlugin struct{}

func (p *testAWSPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "aws", Version: "1.0.0", APIVersion: 1}
}

func (p *testAWSPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{mockplugin.AWSSecretsCapability{}, mockplugin.AWSInstanceConnectCapability{}}
}
