package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestAuthProfileClient(t *testing.T) {
	var experimental []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		experimental = append(experimental, r.Header.Get("X-SailPoint-Experimental"))
		switch r.URL.Path {
		case "/v2026/auth-profiles":
			_, _ = w.Write([]byte(`[{"tenant":"acme","id":"ap-1"},{"tenant":"acme","id":"ap-2"},{"tenant":"acme","id":"ap-3"}]`))
		case "/v2026/auth-profiles/ap-1":
			_, _ = w.Write([]byte(`{"name":"Default","offNetwork":false,"untrustedGeography":false,"applicationId":null,"applicationName":null,"type":"PTA","strongAuthLogin":false}`))
		case "/v2026/auth-profiles/ap-2", "/v2026/auth-profiles/ap-3":
			_, _ = w.Write([]byte(`{"name":"Block","offNetwork":true,"untrustedGeography":true,"applicationId":"app-1","applicationName":"App","type":"BLOCK","strongAuthLogin":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	id, profile, err := client.GetAuthProfileByName(ctx, "Default")
	if err != nil || id != "ap-1" || profile.Type != "PTA" {
		t.Fatalf("unexpected lookup %s %+v (%v)", id, profile, err)
	}
	for _, header := range experimental {
		if header != "true" {
			t.Fatalf("expected the experimental header on every request, got %v", experimental)
		}
	}

	if _, _, err := client.GetAuthProfileByName(ctx, "Block"); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("expected a duplicate name error, got %v", err)
	}
	if _, _, err := client.GetAuthProfileByName(ctx, "Missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}

	profile, err = client.GetAuthProfile(ctx, "ap-2")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var data AuthProfileModel
	setAuthProfileState(&data, "ap-2", profile)
	if data.ID.ValueString() != "ap-2" || data.ApplicationID.ValueString() != "app-1" || !data.OffNetwork.ValueBool() || !data.StrongAuthLogin.ValueBool() {
		t.Fatalf("unexpected state %+v", data)
	}
	setAuthProfileState(&data, "ap-1", &AuthProfile{Name: "Default"})
	if !data.ApplicationID.IsNull() || !data.ApplicationName.IsNull() {
		t.Fatalf("expected null application for an unset value, got %+v", data)
	}
}
