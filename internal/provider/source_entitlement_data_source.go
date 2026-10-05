package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Name         types.String `tfsdk:"name"`
	SourceID     types.String `tfsdk:"source_id"`
	Entitlements types.List   `tfsdk:"entitlements"`
}

type SourceEntitlementItemModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Attribute              types.String `tfsdk:"attribute"`
	Value                  types.String `tfsdk:"value"`
	SourceSchemaObjectType types.String `tfsdk:"source_schema_object_type"`
	Privileged             types.Bool   `tfsdk:"privileged"`
	Requestable            types.Bool   `tfsdk:"requestable"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
	Owner                  types.List   `tfsdk:"owner"`
	DirectPermissions      types.List   `tfsdk:"direct_permissions"`
}

func (d *SourceEntitlementDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_entitlement"
}

func (d *SourceEntitlementDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source Entitlement data source - looks up entitlements by source ID and name",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the entitlement.",
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the source.",
			},
			"entitlements": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of the matching entitlements.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                        schema.StringAttribute{Computed: true, MarkdownDescription: "Entitlement ID."},
						"name":                      schema.StringAttribute{Computed: true, MarkdownDescription: "Entitlement name."},
						"description":               schema.StringAttribute{Computed: true, MarkdownDescription: "Entitlement description."},
						"attribute":                 schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the account attribute the entitlement is derived from, e.g. `memberOf`."},
						"value":                     schema.StringAttribute{Computed: true, MarkdownDescription: "Value of the entitlement, e.g. the group distinguished name."},
						"source_schema_object_type": schema.StringAttribute{Computed: true, MarkdownDescription: "Schema object type of the entitlement, e.g. `group`."},
						"privileged":                schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the entitlement is privileged."},
						"requestable":               schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the entitlement is requestable."},
						"created":                   schema.StringAttribute{Computed: true, MarkdownDescription: "The date and time the entitlement was created."},
						"modified":                  schema.StringAttribute{Computed: true, MarkdownDescription: "The date and time the entitlement was last modified."},
						"owner": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "List with the owner of the entitlement, null when the entitlement has no owner.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Owner ID."},
									"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner type."},
									"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner name."},
								},
							},
						},
						"direct_permissions": schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Direct permissions of the entitlement, each formatted as a string."},
					},
				},
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
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Entitlement with name %s not found in source %s", data.Name.ValueString(), data.SourceID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(entitlements) == 0 {
		tflog.Warn(ctx, fmt.Sprintf("Entitlement with name %s not found in source %s, returning null values", data.Name.ValueString(), data.SourceID.ValueString()))
		setEntitlementNullState(ctx, &data, resp)
		return
	}

	// Build entitlements list
	ownerObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"type": types.StringType,
			"name": types.StringType,
		},
	}

	entitlementObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                        types.StringType,
			"name":                      types.StringType,
			"description":               types.StringType,
			"attribute":                 types.StringType,
			"value":                     types.StringType,
			"source_schema_object_type": types.StringType,
			"privileged":                types.BoolType,
			"requestable":               types.BoolType,
			"created":                   types.StringType,
			"modified":                  types.StringType,
			"owner":                     types.ListType{ElemType: ownerObjType},
			"direct_permissions":        types.ListType{ElemType: types.StringType},
		},
	}

	entModels := []SourceEntitlementItemModel{}
	for _, e := range entitlements {
		// owner
		var ownerList types.List
		if e.Owner != nil {
			if ownerMap, ok := e.Owner.(map[string]interface{}); ok {
				ownerID := ""
				ownerType := ""
				ownerName := ""
				if v, ok := ownerMap["id"].(string); ok {
					ownerID = v
				}
				if v, ok := ownerMap["type"].(string); ok {
					ownerType = v
				}
				if v, ok := ownerMap["name"].(string); ok {
					ownerName = v
				}
				ownerObj := OwnerModel{ID: types.StringValue(ownerID), Type: types.StringValue(ownerType), Name: types.StringValue(ownerName)}
				ol, diags := types.ListValueFrom(ctx, ownerObjType, []OwnerModel{ownerObj})
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				ownerList = ol
			} else {
				ownerList = types.ListNull(ownerObjType)
			}
		} else {
			ownerList = types.ListNull(ownerObjType)
		}

		// direct permissions
		var permList types.List
		if e.DirectPermissions != nil {
			perms := make([]string, len(e.DirectPermissions))
			for i, p := range e.DirectPermissions {
				perms[i] = fmt.Sprintf("%v", p)
			}
			pl, diags := types.ListValueFrom(ctx, types.StringType, perms)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			permList = pl
		} else {
			permList = types.ListNull(types.StringType)
		}

		// build model
		desc := types.StringNull()
		if e.Description != nil {
			if s, ok := e.Description.(string); ok {
				desc = types.StringValue(s)
			} else {
				desc = types.StringValue(fmt.Sprintf("%v", e.Description))
			}
		}
		created := types.StringNull()
		if e.Created != nil {
			created = types.StringValue(fmt.Sprintf("%v", e.Created))
		}
		modified := types.StringNull()
		if e.Modified != nil {
			modified = types.StringValue(fmt.Sprintf("%v", e.Modified))
		}

		item := SourceEntitlementItemModel{
			ID:                     types.StringValue(e.ID),
			Name:                   types.StringValue(e.Name),
			Description:            desc,
			Attribute:              types.StringValue(e.Attribute),
			Value:                  types.StringValue(e.Value),
			SourceSchemaObjectType: types.StringValue(e.SourceSchemaObjectType),
			Privileged:             types.BoolValue(e.Privileged),
			Requestable:            types.BoolValue(e.Requestable),
			Created:                created,
			Modified:               modified,
			Owner:                  ownerList,
			DirectPermissions:      permList,
		}
		entModels = append(entModels, item)
	}

	entList, diags := types.ListValueFrom(ctx, entitlementObjType, entModels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Attach the computed entitlements list to the model
	data.Entitlements = entList

	// Write the entire data model to state (includes entitlements)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func setEntitlementNullState(ctx context.Context, data *SourceEntitlementDataSourceModel, resp *datasource.ReadResponse) {
	// Return an empty entitlements list
	ownerObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"type": types.StringType,
			"name": types.StringType,
		},
	}

	entitlementObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                        types.StringType,
			"name":                      types.StringType,
			"description":               types.StringType,
			"attribute":                 types.StringType,
			"value":                     types.StringType,
			"source_schema_object_type": types.StringType,
			"privileged":                types.BoolType,
			"requestable":               types.BoolType,
			"created":                   types.StringType,
			"modified":                  types.StringType,
			"owner":                     types.ListType{ElemType: ownerObjType},
			"direct_permissions":        types.ListType{ElemType: types.StringType},
		},
	}

	emptyList, diags := types.ListValueFrom(ctx, entitlementObjType, []SourceEntitlementItemModel{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Entitlements = emptyList
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
