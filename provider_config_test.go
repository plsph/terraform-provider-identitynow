package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInt64FromEnv(t *testing.T) {
	tests := []struct {
		name    string
		value   types.Int64
		env     string
		want    int64
		wantErr bool
	}{
		{"configured value wins", types.Int64Value(5), "7", 5, false},
		{"env is parsed", types.Int64Null(), "7", 7, false},
		{"default when unset", types.Int64Null(), "", 3, false},
		{"invalid env", types.Int64Null(), "abc", 3, true},
		{"zero env", types.Int64Null(), "0", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IDENTITYNOW_TEST_INT", tt.env)
			var diags diag.Diagnostics
			got := int64FromEnv(tt.value, "IDENTITYNOW_TEST_INT", "test", 3, &diags)
			if got != tt.want || diags.HasError() != tt.wantErr {
				t.Fatalf("got %d (errors %v), want %d (errors %v)", got, diags, tt.want, tt.wantErr)
			}
		})
	}
}

func TestStringFromEnv(t *testing.T) {
	t.Setenv("IDENTITYNOW_TEST_STRING", "from-env")
	if got := stringFromEnv(types.StringValue("configured"), "IDENTITYNOW_TEST_STRING"); got != "configured" {
		t.Errorf("expected configured value, got %q", got)
	}
	if got := stringFromEnv(types.StringNull(), "IDENTITYNOW_TEST_STRING"); got != "from-env" {
		t.Errorf("expected env value, got %q", got)
	}
}
