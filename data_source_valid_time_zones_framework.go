package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerDataSource(NewValidTimeZonesDataSource)
}

const (
	validTimeZonesID = "valid-time-zones"
	// validTimeZonesPageSize is the maximum limit of the valid time zones API.
	validTimeZonesPageSize = 50
)

// GetValidTimeZones lists all time zones that can be set in the org configuration. The API is
// experimental.
func (c *Client) GetValidTimeZones(ctx context.Context) ([]string, error) {
	zones, err := listAllPagesSized[string](ctx, c, "/v2026/org-config/valid-time-zones", nil, validTimeZonesPageSize, withExperimental())
	if zones == nil && err == nil {
		zones = []string{}
	}
	return zones, err
}

var _ datasource.DataSource = &ValidTimeZonesDataSource{}

func NewValidTimeZonesDataSource() datasource.DataSource {
	return &ValidTimeZonesDataSource{}
}

type ValidTimeZonesDataSource struct {
	client *Config
}

type ValidTimeZonesModel struct {
	ID        types.String `tfsdk:"id"`
	TimeZones types.List   `tfsdk:"time_zones"`
}

func (d *ValidTimeZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_valid_time_zones"
}

func (d *ValidTimeZonesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Lists the time zones that can be set in the org configuration (`identitynow_org_config`).",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{MarkdownDescription: "Always `" + validTimeZonesID + "`", Computed: true},
			"time_zones": dsschema.ListAttribute{
				MarkdownDescription: "Valid time zone names, e.g. `Europe/Warsaw`",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ValidTimeZonesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ValidTimeZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	zones, err := client.GetValidTimeZones(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read valid time zones: %s", err))
		return
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, zones)
	resp.Diagnostics.Append(diags...)
	data := ValidTimeZonesModel{ID: types.StringValue(validTimeZonesID), TimeZones: list}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
