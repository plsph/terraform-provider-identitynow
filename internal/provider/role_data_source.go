package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &RoleDataSource{}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

type RoleDataSource struct {
	client *Config
}

type RoleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Owner          types.List   `tfsdk:"owner"`
	AccessProfiles types.List   `tfsdk:"access_profiles"`
	Requestable    types.Bool   `tfsdk:"requestable"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Role data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the role.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the role.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the role.",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the owner of the role.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Owner ID."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner type."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner name."},
					},
				},
			},
			"access_profiles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The access profiles assigned to this role.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Access profile ID."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Access profile type."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Access profile name."},
					},
				},
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the role is requestable.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the role is enabled.",
			},
		},
	}
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Role data source", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	role, err := client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Role with ID %s not found", data.ID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(role.Name)
	data.Description = types.StringValue(role.Description)

	if role.Requestable != nil {
		data.Requestable = types.BoolValue(*role.Requestable)
	} else {
		data.Requestable = types.BoolNull()
	}

	if role.Enabled != nil {
		data.Enabled = types.BoolValue(*role.Enabled)
	} else {
		data.Enabled = types.BoolNull()
	}

	data.Owner = objectInfoListState(ctx, role.RoleOwner, &resp.Diagnostics)
	data.AccessProfiles = objectInfoSliceState(ctx, role.AccessProfiles, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
