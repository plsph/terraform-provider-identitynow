package main

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"identitynow": providerserver.NewProtocol6WithError(New("test")()),
}

func TestProvider(t *testing.T) {
	// Verify the provider can be instantiated without error
	p := New("test")()
	if p == nil {
		t.Fatal("Provider returned nil")
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("IDENTITYNOW_URL"); v == "" {
		t.Fatal("IDENTITYNOW_URL must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_CLIENT_ID"); v == "" {
		t.Fatal("IDENTITYNOW_CLIENT_ID must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_CLIENT_SECRET"); v == "" {
		t.Fatal("IDENTITYNOW_CLIENT_SECRET must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_EXTERNAL_OWNER_ID"); v == "" {
		t.Fatal("IDENTITYNOW_EXTERNAL_OWNER_ID must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_OWNER_ID"); v == "" {
		t.Fatal("IDENTITYNOW_OWNER_ID must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_OWNER_NAME"); v == "" {
		t.Fatal("IDENTITYNOW_OWNER_NAME must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_CLUSTER_ID"); v == "" {
		t.Fatal("IDENTITYNOW_CLUSTER_ID must be set for acceptance tests")
	}
	if v := os.Getenv("IDENTITYNOW_CLUSTER_NAME"); v == "" {
		t.Fatal("IDENTITYNOW_CLUSTER_NAME must be set for acceptance tests")
	}
}
