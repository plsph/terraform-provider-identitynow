package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVerifiedFromAddressClient(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		gotQuery = r.URL.Query().Get("filters")
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"a-1","email":"Sender@Example.com","isVerifiedByDomain":false,"verificationStatus":"SUCCESS","region":"us-east-1"}]`))
		default:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"a-1","email":"sender@example.com","verificationStatus":"PENDING","region":"us-east-1"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	created, err := client.CreateVerifiedFromAddress(ctx, &VerifiedFromAddress{Email: "sender@example.com"})
	if err != nil || gotMethod != http.MethodPost || gotPath != "/v2026/verified-from-addresses" || gotBody != `{"email":"sender@example.com"}` {
		t.Fatalf("unexpected create request %s %s %s (%v)", gotMethod, gotPath, gotBody, err)
	}
	data := VerifiedFromAddressModel{Email: types.StringValue("sender@example.com")}
	setVerifiedFromAddressState(&data, created)
	if data.ID.ValueString() != "a-1" || data.VerificationStatus.ValueString() != "PENDING" || data.IsVerifiedByDomain.ValueBool() {
		t.Fatalf("unexpected state %+v", data)
	}

	read, err := client.GetVerifiedFromAddress(ctx, "a-1", "sender@example.com")
	if err != nil || gotQuery != `email eq "sender@example.com"` {
		t.Fatalf("unexpected read %q (%v)", gotQuery, err)
	}
	setVerifiedFromAddressState(&data, read)
	if data.Email.ValueString() != "sender@example.com" || data.VerificationStatus.ValueString() != "SUCCESS" {
		t.Fatalf("expected the configured email to be kept and the status refreshed, got %+v", data)
	}

	// After an import only the ID is known and all addresses are listed.
	if _, err := client.GetVerifiedFromAddress(ctx, "a-1", ""); err != nil || gotQuery != "" {
		t.Fatalf("unexpected read without email %q (%v)", gotQuery, err)
	}
	if _, err := client.GetVerifiedFromAddress(ctx, "missing", ""); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if found, err := client.GetVerifiedFromAddressByEmail(ctx, "sender@example.com"); err != nil || found.ID != "a-1" {
		t.Fatalf("unexpected lookup by email %+v (%v)", found, err)
	}

	if err := client.DeleteVerifiedFromAddress(ctx, "a-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/verified-from-addresses/a-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

// The email filter of the API may be case-sensitive while the state keeps the configured case:
// when the filtered list misses the address, all addresses are listed.
func TestVerifiedFromAddressFallsBackToUnfilteredList(t *testing.T) {
	var queries []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("filters")
		queries = append(queries, filter)
		if filter != "" && filter != `email eq "Sender@Example.com"` {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[{"id":"a-1","email":"Sender@Example.com","verificationStatus":"SUCCESS"}]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	read, err := client.GetVerifiedFromAddress(ctx, "a-1", "sender@example.com")
	if err != nil || read.ID != "a-1" || len(queries) != 2 || queries[0] != `email eq "sender@example.com"` || queries[1] != "" {
		t.Fatalf("expected a filtered and an unfiltered list, got %v (%+v, %v)", queries, read, err)
	}
	queries = nil
	if found, err := client.GetVerifiedFromAddressByEmail(ctx, "sender@example.com"); err != nil || found.ID != "a-1" || len(queries) != 2 {
		t.Fatalf("unexpected lookup by email %+v %v (%v)", found, queries, err)
	}
	queries = nil
	if _, err := client.GetVerifiedFromAddress(ctx, "a-1", "Sender@Example.com"); err != nil || len(queries) != 1 {
		t.Fatalf("a filtered match must not list all addresses, got %v (%v)", queries, err)
	}
	if _, err := client.GetVerifiedFromAddress(ctx, "missing", "sender@example.com"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
