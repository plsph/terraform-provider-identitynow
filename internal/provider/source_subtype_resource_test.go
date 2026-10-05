package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSourceSubtypeClient(t *testing.T) {
	const subtypeJSON = `{"id":"st-1","sourceId":"src-1","technicalName":"svc","displayName":"Service","description":"d","type":"MACHINE","created":"c","modified":"m","systemManaged":false}`
	var gotMethod, gotPath, gotBody, gotContentType, gotExperimental, gotFilter string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		gotContentType, gotExperimental = r.Header.Get("Content-Type"), r.Header.Get("X-SailPoint-Experimental")
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/source-subtypes":
			gotFilter = r.URL.Query().Get("filters")
			_, _ = w.Write([]byte(`[{"id":"st-0","technicalName":"other","source":{"id":"src-1"}},` + subtypeJSON + `]`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(subtypeJSON))
		default:
			_, _ = w.Write([]byte(subtypeJSON))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	if _, err := client.CreateSourceSubtype(ctx, &SourceSubtype{SourceID: "src-1", TechnicalName: "svc", DisplayName: "Service", Description: "d"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/source-subtypes" || gotExperimental != "true" ||
		gotBody != `{"sourceId":"src-1","technicalName":"svc","displayName":"Service","description":"d"}` {
		t.Fatalf("unexpected create request %s %s %s %s", gotMethod, gotPath, gotExperimental, gotBody)
	}

	plan := SourceSubtypeModel{DisplayName: types.StringValue("Service Account"), Description: types.StringValue("d")}
	state := SourceSubtypeModel{DisplayName: types.StringValue("Service"), Description: types.StringValue("d")}
	if _, err := client.PatchSourceSubtype(ctx, "st-1", sourceSubtypePatch(plan, state)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/source-subtypes/st-1" || gotExperimental != "true" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/displayName","value":"Service Account"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s %s", gotMethod, gotPath, gotExperimental, gotContentType, gotBody)
	}

	found, err := client.GetSourceSubtypeByTechnicalName(ctx, "src-1", "svc")
	if err != nil || found.ID != "st-1" || gotFilter != `source.id eq "src-1"` || gotExperimental != "true" {
		t.Fatalf("unexpected lookup %+v %q (%v)", found, gotFilter, err)
	}
	if _, err := client.GetSourceSubtypeByTechnicalName(ctx, "src-1", "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := client.GetSourceSubtype(ctx, "st-1"); err != nil || gotPath != "/v2026/source-subtypes/st-1" || gotExperimental != "true" {
		t.Fatalf("unexpected get request %s %s (%v)", gotPath, gotExperimental, err)
	}
	if err := client.DeleteSourceSubtype(ctx, "st-1"); err != nil || gotMethod != http.MethodDelete || gotExperimental != "true" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotExperimental, err)
	}
}

func TestSourceSubtypeState(t *testing.T) {
	managed := true
	api := &SourceSubtype{ID: "st-1", Source: &SourceSubtypeSourceRef{ID: "src-1"}, TechnicalName: "svc", DisplayName: "Service", Description: "d", Type: "MACHINE", Created: "c", Modified: "m", SystemManaged: &managed}

	planned := SourceSubtypeModel{
		ID: types.StringUnknown(), SourceID: types.StringValue("src-1"), TechnicalName: types.StringValue("svc"), DisplayName: types.StringValue("Service"),
		Description: types.StringValue("d"), Type: types.StringUnknown(), SystemManaged: types.BoolUnknown(), Created: types.StringUnknown(), Modified: types.StringUnknown(),
	}
	setSourceSubtypeState(&planned, api, false)
	if planned.ID.ValueString() != "st-1" || planned.Type.ValueString() != "MACHINE" || !planned.SystemManaged.ValueBool() || planned.Modified.ValueString() != "m" {
		t.Fatalf("unexpected state after create %+v", planned)
	}

	imported := SourceSubtypeModel{ID: types.StringValue("st-1")}
	setSourceSubtypeState(&imported, api, true)
	if imported.SourceID.ValueString() != "src-1" || imported.TechnicalName.ValueString() != "svc" || imported.DisplayName.ValueString() != "Service" {
		t.Fatalf("unexpected state after import %+v", imported)
	}

	if ops := sourceSubtypePatch(planned, planned); len(ops) != 0 {
		t.Fatalf("expected no operations, got %+v", ops)
	}
}
