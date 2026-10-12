package tenancy_test

import (
	"errors"
	"testing"

	"github.com/faustbrian/go-tenancy/v2"
)

// FuzzTenantIDRoundTrip checks hostile raw identity strings: rejection leaves
// invalid state, while accepted values survive text decoding and redact diagnostics.
func FuzzTenantIDRoundTrip(f *testing.F) {
	for _, seed := range []string{"tenant-a", "A", "0", "a/b:c.d_e-f", "", " tenant", "t\x00x", "é"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		id, err := tenancy.ParseTenantID(raw)
		if err != nil {
			if id.Valid() {
				t.Fatal("invalid input produced a valid tenant ID")
			}
			return
		}
		text, err := id.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var decoded tenancy.TenantID
		if err := decoded.UnmarshalText(text); err != nil || !decoded.Equal(id) {
			t.Fatalf("round trip = %v, %v", decoded, err)
		}
		if id.String() == raw {
			t.Fatal("diagnostic formatting disclosed the raw tenant ID")
		}
	})
}

// FuzzPropagationExtraction varies raw IDs, repeated-value count and trust.
// Acceptance requires exactly one trusted unchanged tenant; missing stays distinct.
func FuzzPropagationExtraction(f *testing.F) {
	for _, seed := range []string{"tenant-a", "", "tenant-a\x00", "tenant/a", "é"} {
		f.Add(seed, uint8(1), true)
	}
	f.Fuzz(func(t *testing.T, raw string, count uint8, trusted bool) {
		codec, _ := tenancy.NewPropagationCodec(tenancy.PropagationOptions{})
		values := make([]string, int(count%10))
		for index := range values {
			values[index] = raw
		}
		carrier := tenancy.MapCarrier{tenancy.DefaultTenantField: values}
		scope, err := codec.Extract(carrier, trusted)
		if err == nil {
			if len(values) != 1 || !trusted || scope.TenantID().Value() != raw {
				t.Fatalf("unexpected accepted scope %#v", scope)
			}
			return
		}
		if len(values) == 0 && !errors.Is(err, tenancy.ErrTenantMetadataMissing) {
			t.Fatalf("missing metadata error = %v", err)
		}
	})
}
