package tenancy_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/faustbrian/go-tenancy/v2"
	tenantjsonrpc "github.com/faustbrian/go-tenancy/v2/jsonrpc"
)

func TestSecurityAdmissionBeforeNamespaceInputCopy(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("allocation characterization runs only in hosted CI")
	}
	integration, err := tenancy.NewIntegration(tenancy.BoundaryQueue, tenancy.PropagationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := tenancy.NewNamespaceEncoder(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := tenancy.NewTenantScope(tenancy.MustTenantID("tenant-a"), tenancy.Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	logicalKey := strings.Repeat("x", 1<<20)
	var key string
	var resultErr error
	allocations := testing.AllocsPerRun(10, func() {
		key, resultErr = integration.Key(encoder, scope, logicalKey)
	})
	if key != "" || !errors.Is(resultErr, tenancy.ErrInvalidNamespaceInput) {
		t.Fatal("oversized integration key did not refuse without output")
	}
	if allocations != 0 {
		t.Fatalf("rejected integration key allocated: got %f want zero", allocations)
	}
}

func TestSecurityAdmissionJSONRPCOutput(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("native codec characterization runs only in hosted CI")
	}
	metadata := []byte(`{"trace":"safe"}`)
	codec, err := tenantjsonrpc.New(tenantjsonrpc.Options{
		MaxMetadataBytes: len(metadata),
		Trust:            func(context.Context) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := tenancy.NewTenantScope(tenancy.MustTenantID("tenant-a"), tenancy.Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Inject(metadata, scope)
	if encoded != nil || !errors.Is(err, tenantjsonrpc.ErrOversizedMetadata) {
		t.Fatal("encoded metadata exceeding the configured allowance was not refused without output")
	}
}

func TestSecurityAdmissionJSONRPCInclusiveOutput(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("native codec characterization runs only in hosted CI")
	}
	const expected = `{"tenant_id":"tenant-a"}`
	scope, err := tenancy.NewTenantScope(tenancy.MustTenantID("tenant-a"), tenancy.Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	for _, allowance := range []int{len(expected) - 1, len(expected)} {
		codec, err := tenantjsonrpc.New(tenantjsonrpc.Options{
			MaxMetadataBytes: allowance,
			Trust:            func(context.Context) bool { return true },
		})
		if err != nil {
			t.Fatal(err)
		}
		metadata := []byte(`{}`)
		encoded, err := codec.Inject(metadata, scope)
		if allowance < len(expected) {
			if encoded != nil || !errors.Is(err, tenantjsonrpc.ErrOversizedMetadata) {
				t.Fatal("one-byte-short output allowance was not refused without output")
			}
		} else if err != nil || string(encoded) != expected {
			t.Fatal("inclusive output allowance did not preserve the encoded tenant field")
		}
		if string(metadata) != `{}` {
			t.Fatal("injection mutated caller-owned metadata")
		}
	}
}
