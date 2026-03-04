package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &GovernanceGroupDataSource{}

func NewGovernanceGroupDataSource() datasource.DataSource {
	return &GovernanceGroupDataSource{}
}

type GovernanceGroupDataSource struct {
	client *Config
}

type GovernanceGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Owner       types.List   `tfsdk:"owner"`
}

func (d *GovernanceGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_governance_group"
}

func (d *GovernanceGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Governance Group data source - looks up a governance group by name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Governance Group ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Governance Group name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Governance Group description",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Governance Group owner",
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

func (d *GovernanceGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GovernanceGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GovernanceGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Governance Group data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	governanceGroups, err := client.GetGovernanceGroupByName(ctx, data.Name.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Governance Group with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(governanceGroups) == 0 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Governance Group with name %s not found", data.Name.ValueString()))
		return
	}

	gg := governanceGroups[0]
	data.ID = types.StringValue(gg.ID)
	data.Name = types.StringValue(gg.Name)
	data.Description = types.StringValue(gg.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
