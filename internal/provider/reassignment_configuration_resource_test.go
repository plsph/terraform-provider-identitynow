package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const reassignmentConfigurationTestResponse = `{"identity":{"id":"id-1","name":"John"},"configDetails":[
	{"configType":"CERTIFICATIONS","targetIdentity":{"id":"other"},"startDate":"2026-01-01T00:00:00Z"},
	{"configType":"ACCESS_REQUESTS","targetIdentity":{"id":"id-2","name":"Jane"},"startDate":"2026-10-01T00:00:00.000Z","endDate":"","auditDetails":{"created":"c","modified":"m"}}]}`

func TestReassignmentConfigurationClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotExperimental string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotExperimental = r.Method, r.URL.Path, string(body), r.Header.Get("X-SailPoint-Experimental")
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(reassignmentConfigurationTestResponse))
		default:
			_, _ = w.Write([]byte(reassignmentConfigurationTestResponse))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	plan := ReassignmentConfigurationModel{
		ID:             types.StringUnknown(),
		IdentityID:     types.StringValue("id-1"),
		ConfigType:     types.StringValue("ACCESS_REQUESTS"),
		ReassignedToID: types.StringValue("id-2"),
		StartDate:      types.StringUnknown(),
		EndDate:        types.StringNull(),
		Created:        types.StringUnknown(),
		Modified:       types.StringUnknown(),
	}
	created, err := client.CreateReassignmentConfiguration(ctx, reassignmentConfigurationFromModel(plan))
	want := `{"reassignedFromId":"id-1","reassignedToId":"id-2","configType":"ACCESS_REQUESTS","endDate":null}`
	if err != nil || gotMethod != http.MethodPost || gotPath != "/v2026/reassignment-configurations" || gotBody != want || gotExperimental != "true" {
		t.Fatalf("unexpected create request %s %s %s %q (%v)", gotMethod, gotPath, gotBody, gotExperimental, err)
	}
	resolveReassignmentConfigurationComputed(&plan, ReassignmentConfigurationModel{}, reassignmentConfigurationDetail(created, "ACCESS_REQUESTS"))
	if plan.ID.ValueString() != "id-1/ACCESS_REQUESTS" || plan.StartDate.ValueString() != "2026-10-01T00:00:00.000Z" || plan.Created.ValueString() != "c" || plan.Modified.ValueString() != "m" || !plan.EndDate.IsNull() {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	plan.EndDate = types.StringValue("2026-12-31T00:00:00Z")
	if _, err := client.UpdateReassignmentConfiguration(ctx, reassignmentConfigurationFromModel(plan)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want = `{"reassignedFromId":"id-1","reassignedToId":"id-2","configType":"ACCESS_REQUESTS","startDate":"2026-10-01T00:00:00.000Z","endDate":"2026-12-31T00:00:00Z"}`
	if gotMethod != http.MethodPut || gotPath != "/v2026/reassignment-configurations/id-1" || gotBody != want || gotExperimental != "true" {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	if _, err := client.GetReassignmentConfiguration(ctx, "id-1"); err != nil || gotMethod != http.MethodGet || gotPath != "/v2026/reassignment-configurations/id-1" || gotExperimental != "true" {
		t.Fatalf("unexpected get request %s %s (%v)", gotMethod, gotPath, err)
	}
	if err := client.DeleteReassignmentConfiguration(ctx, "id-1", "ACCESS_REQUESTS"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/reassignment-configurations/id-1/ACCESS_REQUESTS" || gotExperimental != "true" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestReassignmentConfigurationStateKeepsEquivalentDates(t *testing.T) {
	detail := &ReassignmentConfigurationDetail{ConfigType: "ACCESS_REQUESTS", TargetIdentity: &ReassignmentConfigurationIdentity{ID: "id-3"},
		StartDate: "2026-10-01T02:00:00.000+02:00", EndDate: ""}
	data := ReassignmentConfigurationModel{StartDate: types.StringValue("2026-10-01T00:00:00Z"), EndDate: types.StringNull()}
	setReassignmentConfigurationState(&data, "id-1", detail)
	if data.StartDate.ValueString() != "2026-10-01T00:00:00Z" || !data.EndDate.IsNull() || data.ReassignedToID.ValueString() != "id-3" || data.ID.ValueString() != "id-1/ACCESS_REQUESTS" {
		t.Fatalf("unexpected state %+v", data)
	}
	detail.StartDate = "2026-11-01T00:00:00Z"
	detail.EndDate = "2026-12-31T00:00:00Z"
	setReassignmentConfigurationState(&data, "id-1", detail)
	if data.StartDate.ValueString() != "2026-11-01T00:00:00Z" || data.EndDate.ValueString() != "2026-12-31T00:00:00Z" {
		t.Fatalf("expected changed dates to be detected, got %+v", data)
	}
	if reassignmentConfigurationDetail(&ReassignmentConfiguration{}, "ACCESS_REQUESTS") != nil {
		t.Fatalf("expected no detail")
	}
}
