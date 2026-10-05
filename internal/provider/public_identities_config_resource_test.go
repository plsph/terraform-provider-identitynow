package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPublicIdentitiesConfig(t *testing.T) {
	var gotMethod, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotBody = r.Method, string(body)
		_, _ = w.Write([]byte(`{"attributes":[{"key":"country","name":"Country"}],"modified":"2026-01-01T00:00:00Z"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	// Without attribute blocks the configuration is cleared.
	config := publicIdentitiesConfigFromModel(ctx, PublicIdentitiesConfigModel{Attributes: types.ListNull(publicIdentityAttributeObjectType)}, &diags)
	if _, err := client.UpdatePublicIdentitiesConfig(ctx, config); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotBody != `{"attributes":[]}` {
		t.Fatalf("unexpected request %s %s", gotMethod, gotBody)
	}

	read, err := client.GetPublicIdentitiesConfig(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var data PublicIdentitiesConfigModel
	setPublicIdentitiesConfigState(ctx, &data, read, &diags)
	if diags.HasError() || data.ID.ValueString() != publicIdentitiesConfigID || len(data.Attributes.Elements()) != 1 {
		t.Fatalf("unexpected state %+v (%v)", data, diags)
	}
}
