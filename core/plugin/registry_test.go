package plugin_test

import (
	"sync"
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
	entry := registry.LookupEntry("aws.secrets")
	if entry == nil {
		t.Fatal("LookupEntry() = nil, want entry")
	}
	if !entry.IsValue() {
		t.Fatal("LookupEntry().IsValue() = false, want true")
	}

	transport := registry.Lookup("aws.instance.connect")
	if transport == nil {
		t.Fatal("Lookup() transport = nil, want capability")
	}
	transportEntry := registry.LookupEntry("aws.instance.connect")
	if transportEntry == nil {
		t.Fatal("LookupEntry() transport = nil, want entry")
	}
	if !transportEntry.IsTransport() {
		t.Fatal("LookupEntry().IsTransport() = false, want true")
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

func TestRegistryConcurrentLookupDuringRegister(t *testing.T) {
	registry := plugin.NewRegistry()

	start := make(chan struct{})
	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 2000; j++ {
				_ = registry.Lookup("aws.secrets")
				_ = registry.LookupEntry("aws.secrets")
				_ = registry.ListNamespace("aws")
			}
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	close(start)
	if err := registry.Register(&testAWSPlugin{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	<-done
}

type testAWSPlugin struct{}

func (p *testAWSPlugin) Identity() plugin.PluginIdentity {
	return plugin.PluginIdentity{Name: "aws", Version: "1.0.0", APIVersion: 1}
}

func (p *testAWSPlugin) Capabilities() []plugin.Capability {
	return []plugin.Capability{mockplugin.AWSSecretsCapability{}, mockplugin.AWSInstanceConnectCapability{}}
}
