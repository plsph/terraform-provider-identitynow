package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestTenantDataSourceDecodesProducts(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2026/tenant" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"2c91808568c529c60168cca6f90c1324","name":"acme","fullName":"Acme, Inc","pod":"stg01-useast1","region":"us-east-1",
			"products":[{"productName":"idn","url":"https://acme.identitynow.com","productTenantId":"acme-idn","productRight":"idn:ui:view","apiUrl":null,
			"licenses":[{"licenseId":"idn:access-request","legacyFeatureName":"ACCESS_REQUEST"}],"attributes":{"domain":"acme.com"},"orgType":"production","dateCreated":"2020-01-01T00:00:00Z"}]}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	tenant, err := client.GetTenant(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var data TenantModel
	var diags diag.Diagnostics
	setTenantState(context.Background(), &data, tenant, &diags)
	if diags.HasError() || data.Name.ValueString() != "acme" || len(data.Products.Elements()) != 1 {
		t.Fatalf("unexpected state %+v (%v)", data, diags)
	}
	var products []TenantProductModel
	diags.Append(data.Products.ElementsAs(context.Background(), &products, false)...)
	p := products[0]
	if !p.APIURL.IsNull() || p.OrgType.ValueString() != "production" || p.AttributesJSON.ValueString() != `{"domain":"acme.com"}` || len(p.Licenses.Elements()) != 1 || !p.LastUpdated.IsNull() {
		t.Fatalf("unexpected product %+v", p)
	}
}
