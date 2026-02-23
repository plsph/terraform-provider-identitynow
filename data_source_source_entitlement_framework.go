package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &SourceEntitlementDataSource{}

func NewSourceEntitlementDataSource() datasource.DataSource {
	return &SourceEntitlementDataSource{}
}

type SourceEntitlementDataSource struct {
	client *Config
}

type SourceEntitlementDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	SourceID               types.String `tfsdk:"source_id"`
	SourceName             types.String `tfsdk:"source_name"`
	Description            types.String `tfsdk:"description"`
	Attribute              types.String `tfsdk:"attribute"`
	Value                  types.String `tfsdk:"value"`
	SourceSchemaObjectType types.String `tfsdk:"source_schema_object_type"`
	Privileged             types.Bool   `tfsdk:"privileged"`
	Requestable            types.Bool   `tfsdk:"requestable"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
	Owner                  types.String `tfsdk:"owner"`
	DirectPermissions      types.List   `tfsdk:"direct_permissions"`
}

func (d *SourceEntitlementDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_entitlement"
}

func (d *SourceEntitlementDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source Entitlement data source - looks up entitlements by source ID and name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitlement ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitlement name",
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source ID",
			},
			"source_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitlement description",
			},
			"attribute": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Attribute",
			},
			"value": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Value",
			},
			"source_schema_object_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source schema object type",
			},
			"privileged": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether privileged",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether requestable",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modified timestamp",
			},
			"owner": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owner",
			},
			"direct_permissions": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "Direct permissions",
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *SourceEntitlementDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceEntitlementDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceEntitlementDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Source Entitlement data source", map[string]interface{}{
		"source_id": data.SourceID.ValueString(),
		"name":      data.Name.ValueString(),
	})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	entitlements, err := client.GetSourceEntitlement(ctx, data.SourceID.ValueString(), data.Name.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Entitlement with name %s not found in source %s", data.Name.ValueString(), data.SourceID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(entitlements) == 0 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Entitlement with name %s not found in source %s", data.Name.ValueString(), data.SourceID.ValueString()))
		return
	}

	e := entitlements[0]
	data.ID = types.StringValue(e.ID)
	data.Name = types.StringValue(e.Name)
	data.Attribute = types.StringValue(e.Attribute)
	data.Value = types.StringValue(e.Value)
	data.SourceSchemaObjectType = types.StringValue(e.SourceSchemaObjectType)
	data.Privileged = types.BoolValue(e.Privileged)
	data.Requestable = types.BoolValue(e.Requestable)

	if e.Source != nil {
		data.SourceID = types.StringValue(e.Source.ID)
		data.SourceName = types.StringValue(e.Source.Name)
	}

	if e.Description != nil {
		if desc, ok := e.Description.(string); ok {
			data.Description = types.StringValue(desc)
		} else {
			data.Description = types.StringValue(fmt.Sprintf("%v", e.Description))
		}
	} else {
		data.Description = types.StringNull()
	}

	if e.Created != nil {
		data.Created = types.StringValue(fmt.Sprintf("%v", e.Created))
	} else {
		data.Created = types.StringNull()
	}

	if e.Modified != nil {
		data.Modified = types.StringValue(fmt.Sprintf("%v", e.Modified))
	} else {
		data.Modified = types.StringNull()
	}

	if e.Owner != nil {
		data.Owner = types.StringValue(fmt.Sprintf("%v", e.Owner))
	} else {
		data.Owner = types.StringNull()
	}

	// Handle direct permissions
	if e.DirectPermissions != nil {
		perms := make([]string, len(e.DirectPermissions))
		for i, p := range e.DirectPermissions {
			perms[i] = fmt.Sprintf("%v", p)
		}
		permList, diags := types.ListValueFrom(ctx, types.StringType, perms)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.DirectPermissions = permList
	} else {
		data.DirectPermissions = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
