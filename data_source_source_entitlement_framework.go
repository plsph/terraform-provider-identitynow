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
				MarkdownDescription: "Entitlement name",
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source ID",
			},
			"entitlements": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching entitlements",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                        schema.StringAttribute{Computed: true},
						"name":                      schema.StringAttribute{Computed: true},
						"description":               schema.StringAttribute{Computed: true},
						"attribute":                 schema.StringAttribute{Computed: true},
						"value":                     schema.StringAttribute{Computed: true},
						"source_schema_object_type": schema.StringAttribute{Computed: true},
						"privileged":                schema.BoolAttribute{Computed: true},
						"requestable":               schema.BoolAttribute{Computed: true},
						"created":                   schema.StringAttribute{Computed: true},
						"modified":                  schema.StringAttribute{Computed: true},
						"owner": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id":   schema.StringAttribute{Computed: true},
									"type": schema.StringAttribute{Computed: true},
									"name": schema.StringAttribute{Computed: true},
								},
							},
						},
						"direct_permissions": schema.ListAttribute{Computed: true, ElementType: types.StringType},
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
		if _, notFound := err.(*NotFoundError); notFound {
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
	entRaw := make([]map[string]interface{}, 0, len(entitlements))
	for _, e := range entitlements {
		// owner
		var ownerList types.List
		ownerRaw := []map[string]interface{}{}
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
				ownerRaw = append(ownerRaw, map[string]interface{}{"id": ownerID, "type": ownerType, "name": ownerName})
			} else {
				ownerList = types.ListNull(ownerObjType)
			}
		} else {
			ownerList = types.ListNull(ownerObjType)
		}

		// direct permissions
		var permList types.List
		permsRaw := []string{}
		if e.DirectPermissions != nil {
			perms := make([]string, len(e.DirectPermissions))
			for i, p := range e.DirectPermissions {
				perms[i] = fmt.Sprintf("%v", p)
				permsRaw = append(permsRaw, perms[i])
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
		entRaw = append(entRaw, map[string]interface{}{
			"id":   e.ID,
			"name": e.Name,
			"description": func() string {
				if e.Description == nil {
					return ""
				}
				if s, ok := e.Description.(string); ok {
					return s
				}
				return fmt.Sprintf("%v", e.Description)
			}(),
			"attribute":                 e.Attribute,
			"value":                     e.Value,
			"source_schema_object_type": e.SourceSchemaObjectType,
			"privileged":                e.Privileged,
			"requestable":               e.Requestable,
			"created": func() string {
				if e.Created == nil {
					return ""
				}
				return fmt.Sprintf("%v", e.Created)
			}(),
			"modified": func() string {
				if e.Modified == nil {
					return ""
				}
				return fmt.Sprintf("%v", e.Modified)
			}(),
			"owner":              ownerRaw,
			"direct_permissions": permsRaw,
		})
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
