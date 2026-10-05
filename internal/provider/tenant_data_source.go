package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerDataSource(NewTenantDataSource)
}

// Tenant is the tenant information returned by /v2026/tenant.
type Tenant struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	FullName    string           `json:"fullName"`
	Pod         string           `json:"pod"`
	Region      string           `json:"region"`
	Description string           `json:"description"`
	Products    []*TenantProduct `json:"products"`
}

type TenantProduct struct {
	ProductName     string                 `json:"productName"`
	URL             string                 `json:"url"`
	ProductTenantID string                 `json:"productTenantId"`
	ProductRegion   string                 `json:"productRegion"`
	ProductRight    string                 `json:"productRight"`
	APIURL          *string                `json:"apiUrl"`
	Licenses        []*TenantLicense       `json:"licenses"`
	Attributes      map[string]interface{} `json:"attributes"`
	Zone            string                 `json:"zone"`
	Status          string                 `json:"status"`
	StatusDateTime  string                 `json:"statusDateTime"`
	Reason          string                 `json:"reason"`
	Notes           string                 `json:"notes"`
	DateCreated     *string                `json:"dateCreated"`
	LastUpdated     *string                `json:"lastUpdated"`
	OrgType         *string                `json:"orgType"`
}

type TenantLicense struct {
	LicenseID         string `json:"licenseId"`
	LegacyFeatureName string `json:"legacyFeatureName"`
}

func (c *Client) GetTenant(ctx context.Context) (*Tenant, error) {
	var tenant Tenant
	if err := c.doJSON(ctx, http.MethodGet, "/v2026/tenant", nil, &tenant); err != nil {
		return nil, err
	}
	return &tenant, nil
}

var _ datasource.DataSource = &TenantDataSource{}

func NewTenantDataSource() datasource.DataSource {
	return &TenantDataSource{}
}

type TenantDataSource struct {
	client *Config
}

type TenantModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	FullName    types.String `tfsdk:"full_name"`
	Pod         types.String `tfsdk:"pod"`
	Region      types.String `tfsdk:"region"`
	Description types.String `tfsdk:"description"`
	Products    types.List   `tfsdk:"products"`
}

type TenantProductModel struct {
	ProductName     types.String `tfsdk:"product_name"`
	URL             types.String `tfsdk:"url"`
	ProductTenantID types.String `tfsdk:"product_tenant_id"`
	ProductRegion   types.String `tfsdk:"product_region"`
	ProductRight    types.String `tfsdk:"product_right"`
	APIURL          types.String `tfsdk:"api_url"`
	Licenses        types.List   `tfsdk:"licenses"`
	AttributesJSON  types.String `tfsdk:"attributes_json"`
	Zone            types.String `tfsdk:"zone"`
	Status          types.String `tfsdk:"status"`
	StatusDateTime  types.String `tfsdk:"status_date_time"`
	Reason          types.String `tfsdk:"reason"`
	Notes           types.String `tfsdk:"notes"`
	DateCreated     types.String `tfsdk:"date_created"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	OrgType         types.String `tfsdk:"org_type"`
}

type TenantLicenseModel struct {
	LicenseID         types.String `tfsdk:"license_id"`
	LegacyFeatureName types.String `tfsdk:"legacy_feature_name"`
}

var tenantLicenseObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"license_id":          types.StringType,
	"legacy_feature_name": types.StringType,
}}

var tenantProductObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"product_name":      types.StringType,
	"url":               types.StringType,
	"product_tenant_id": types.StringType,
	"product_region":    types.StringType,
	"product_right":     types.StringType,
	"api_url":           types.StringType,
	"licenses":          types.ListType{ElemType: tenantLicenseObjectType},
	"attributes_json":   types.StringType,
	"zone":              types.StringType,
	"status":            types.StringType,
	"status_date_time":  types.StringType,
	"reason":            types.StringType,
	"notes":             types.StringType,
	"date_created":      types.StringType,
	"last_updated":      types.StringType,
	"org_type":          types.StringType,
}}

func (d *TenantDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (d *TenantDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	str := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Reads information about the current tenant and its products.",
		Attributes: map[string]dsschema.Attribute{
			"id":          str("Tenant ID"),
			"name":        str("Abbreviated tenant name"),
			"full_name":   str("Human-readable tenant name"),
			"pod":         str("Deployment pod of the tenant"),
			"region":      str("Deployment region of the tenant"),
			"description": str("Tenant description"),
			"products": dsschema.ListNestedAttribute{
				MarkdownDescription: "Products of the tenant",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"product_name":      str("Product name"),
						"url":               str("Product URL"),
						"product_tenant_id": str("ID of the product-tenant combination"),
						"product_region":    str("Product region"),
						"product_right":     str("Right needed for the product"),
						"api_url":           str("API URL of the product"),
						"licenses": dsschema.ListNestedAttribute{
							MarkdownDescription: "Licenses of the product",
							Computed:            true,
							NestedObject: dsschema.NestedAttributeObject{
								Attributes: map[string]dsschema.Attribute{
									"license_id":          str("License name"),
									"legacy_feature_name": str("Legacy license name"),
								},
							},
						},
						"attributes_json":  str("Additional product attributes as a JSON object"),
						"zone":             str("Zone"),
						"status":           str("Product status"),
						"status_date_time": str("Date of the status"),
						"reason":           str("Description of the provisioning failure, if any"),
						"notes":            str("Notes added during tenant provisioning"),
						"date_created":     str("Creation date of the product"),
						"last_updated":     str("Last update date of the product"),
						"org_type":         str("Org type, e.g. `production` or `sandbox`"),
					},
				},
			},
		},
	}
}

func (d *TenantDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func tenantOptionalString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func setTenantState(ctx context.Context, data *TenantModel, tenant *Tenant, diags *diag.Diagnostics) {
	data.ID = types.StringValue(tenant.ID)
	data.Name = types.StringValue(tenant.Name)
	data.FullName = types.StringValue(tenant.FullName)
	data.Pod = types.StringValue(tenant.Pod)
	data.Region = types.StringValue(tenant.Region)
	data.Description = types.StringValue(tenant.Description)
	products := make([]TenantProductModel, 0, len(tenant.Products))
	for _, p := range tenant.Products {
		if p == nil {
			continue
		}
		licenses := make([]TenantLicenseModel, 0, len(p.Licenses))
		for _, l := range p.Licenses {
			if l != nil {
				licenses = append(licenses, TenantLicenseModel{LicenseID: types.StringValue(l.LicenseID), LegacyFeatureName: types.StringValue(l.LegacyFeatureName)})
			}
		}
		licenseList, d := types.ListValueFrom(ctx, tenantLicenseObjectType, licenses)
		diags.Append(d...)
		attributes := types.StringNull()
		if len(p.Attributes) > 0 {
			if encoded, err := json.Marshal(p.Attributes); err == nil {
				attributes = types.StringValue(string(encoded))
			}
		}
		products = append(products, TenantProductModel{
			ProductName:     types.StringValue(p.ProductName),
			URL:             types.StringValue(p.URL),
			ProductTenantID: types.StringValue(p.ProductTenantID),
			ProductRegion:   types.StringValue(p.ProductRegion),
			ProductRight:    types.StringValue(p.ProductRight),
			APIURL:          tenantOptionalString(p.APIURL),
			Licenses:        licenseList,
			AttributesJSON:  attributes,
			Zone:            types.StringValue(p.Zone),
			Status:          types.StringValue(p.Status),
			StatusDateTime:  types.StringValue(p.StatusDateTime),
			Reason:          types.StringValue(p.Reason),
			Notes:           types.StringValue(p.Notes),
			DateCreated:     tenantOptionalString(p.DateCreated),
			LastUpdated:     tenantOptionalString(p.LastUpdated),
			OrgType:         tenantOptionalString(p.OrgType),
		})
	}
	list, d := types.ListValueFrom(ctx, tenantProductObjectType, products)
	diags.Append(d...)
	data.Products = list
}

func (d *TenantDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	tenant, err := client.GetTenant(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tenant: %s", err))
		return
	}
	var data TenantModel
	setTenantState(ctx, &data, tenant, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
