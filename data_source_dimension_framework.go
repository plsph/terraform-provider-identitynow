package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &DimensionDataSource{}

func NewDimensionDataSource() datasource.DataSource {
	return &DimensionDataSource{}
}

type DimensionDataSource struct {
	client *Config
}

type DimensionDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleID         types.String `tfsdk:"role_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Owner          types.List   `tfsdk:"owner"`
	AccessProfiles types.List   `tfsdk:"access_profiles"`
	Entitlements   types.List   `tfsdk:"entitlements"`
}

func (d *DimensionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dimension"
}

func (d *DimensionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Dimension data source. A dimension is a sub-division of a role that allows fine-grained access grouping.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Dimension ID",
			},
			"role_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the role this dimension belongs to",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Dimension name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Dimension description",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Dimension owner",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"access_profiles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access profiles assigned to this dimension",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"entitlements": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Entitlements assigned to this dimension",
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

func (d *DimensionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DimensionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DimensionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Dimension data source", map[string]interface{}{
		"id":      data.ID.ValueString(),
		"role_id": data.RoleID.ValueString(),
	})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	dimension, err := client.GetDimension(ctx, data.RoleID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(dimension.Name)
	data.Description = types.StringValue(dimension.Description)

	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
		"name": types.StringType,
	}}

	if dimension.Owner != nil {
		ownerModels := []OwnerModel{
			{
				ID:   types.StringValue(fmt.Sprintf("%v", dimension.Owner.ID)),
				Type: types.StringValue(dimension.Owner.Type),
				Name: types.StringValue(dimension.Owner.Name),
			},
		}
		ownerList, diags := types.ListValueFrom(ctx, objType, ownerModels)
		resp.Diagnostics.Append(diags...)
		data.Owner = ownerList
	} else {
		data.Owner, _ = types.ListValue(objType, []attr.Value{})
	}

	if dimension.AccessProfiles != nil {
		apModels := make([]AccessProfileRefModel, len(dimension.AccessProfiles))
		for i, ap := range dimension.AccessProfiles {
			apModels[i] = AccessProfileRefModel{
				ID:   types.StringValue(fmt.Sprintf("%v", ap.ID)),
				Type: types.StringValue(ap.Type),
				Name: types.StringValue(ap.Name),
			}
		}
		apList, diags := types.ListValueFrom(ctx, objType, apModels)
		resp.Diagnostics.Append(diags...)
		data.AccessProfiles = apList
	} else {
		data.AccessProfiles, _ = types.ListValue(objType, []attr.Value{})
	}

	if dimension.Entitlements != nil {
		entModels := make([]EntitlementRefModel, len(dimension.Entitlements))
		for i, e := range dimension.Entitlements {
			entModels[i] = EntitlementRefModel{
				ID:   types.StringValue(fmt.Sprintf("%v", e.ID)),
				Type: types.StringValue(e.Type),
				Name: types.StringValue(e.Name),
			}
		}
		entList, diags := types.ListValueFrom(ctx, objType, entModels)
		resp.Diagnostics.Append(diags...)
		data.Entitlements = entList
	} else {
		data.Entitlements, _ = types.ListValue(objType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
