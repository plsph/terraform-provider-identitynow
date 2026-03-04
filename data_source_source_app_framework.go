package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &SourceAppDataSource{}

func NewSourceAppDataSource() datasource.DataSource {
	return &SourceAppDataSource{}
}

type SourceAppDataSource struct {
	client *Config
}

type SourceAppDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	MatchAllAccounts types.Bool   `tfsdk:"match_all_accounts"`
	Source           types.List   `tfsdk:"source"`
}

func (d *SourceAppDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_app"
}

func (d *SourceAppDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source App data source - looks up a source app by name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source App ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source App name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source App description",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether enabled",
			},
			"match_all_accounts": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether to match all accounts",
			},
			"source": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Account source",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SourceAppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceAppDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Source App data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	sourceApps, err := client.GetSourceAppByName(ctx, data.Name.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Source App with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(sourceApps) == 0 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Source App with name %s not found", data.Name.ValueString()))
		return
	}

	sa := sourceApps[0]
	data.ID = types.StringValue(sa.ID)
	data.Name = types.StringValue(sa.Name)
	data.Description = types.StringValue(sa.Description)

	if sa.Enabled != nil {
		data.Enabled = types.BoolValue(*sa.Enabled)
	} else {
		data.Enabled = types.BoolNull()
	}

	if sa.MatchAllAccounts != nil {
		data.MatchAllAccounts = types.BoolValue(*sa.MatchAllAccounts)
	} else {
		data.MatchAllAccounts = types.BoolNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
