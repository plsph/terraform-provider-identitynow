package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AccountSchemaResource{}
var _ resource.ResourceWithImportState = &AccountSchemaResource{}

func NewAccountSchemaResource() resource.Resource {
	return &AccountSchemaResource{}
}

type AccountSchemaResource struct {
	client *Config
}

type AccountSchemaResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	SourceID           types.String `tfsdk:"source_id"`
	SchemaID           types.String `tfsdk:"schema_id"`
	DisplayAttribute   types.String `tfsdk:"display_attribute"`
	IdentityAttribute  types.String `tfsdk:"identity_attribute"`
	NativeObjectType   types.String `tfsdk:"native_object_type"`
	HierarchyAttribute types.String `tfsdk:"hierarchy_attribute"`
	IncludePermissions types.Bool   `tfsdk:"include_permissions"`
	Modified           types.String `tfsdk:"modified"`
	Created            types.String `tfsdk:"created"`
	Attributes         types.List   `tfsdk:"attributes"`
}

type AccountSchemaAttributeModel struct {
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Description   types.String `tfsdk:"description"`
	IsGroup       types.Bool   `tfsdk:"is_group"`
	IsMultiValued types.Bool   `tfsdk:"is_multi_valued"`
	IsEntitlement types.Bool   `tfsdk:"is_entitlement"`
	Schema        types.List   `tfsdk:"schema"`
}

type AccountSchemaAttributeSchemaModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (r *AccountSchemaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account_schema"
}

func (r *AccountSchemaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Account Schema resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Account Schema ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Account Schema name. Cannot be changed after the schema is created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Schema ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_attribute": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display attribute",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_attribute": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Identity attribute",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"native_object_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Native object type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hierarchy_attribute": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Hierarchy attribute",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"include_permissions": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Include permissions",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last modified timestamp",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"attributes": schema.ListNestedBlock{
				MarkdownDescription: "Schema attributes. When at least one attribute is configured the list is authoritative: attributes that are not configured are removed from the schema. Without attribute blocks the existing attributes are left unchanged.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Attribute name",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Attribute type",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Attribute description",
						},
						"is_group": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Whether this is a group attribute",
						},
						"is_multi_valued": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Whether this attribute is multi-valued",
						},
						"is_entitlement": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Whether this is an entitlement attribute",
						},
					},
					Blocks: map[string]schema.Block{
						"schema": schema.ListNestedBlock{
							MarkdownDescription: "Schema reference",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Schema ID",
									},
									"name": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Schema name",
									},
									"type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Schema type",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *AccountSchemaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *AccountSchemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccountSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	r.apply(ctx, &data, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, accountSchemaAttributesUnmanagedKey, attributesUnmanagedMarker(data.Attributes))...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// apply merges the planned schema settings and attributes into the existing schema and saves it.
// Existing attributes keep fields that are not managed by Terraform, such as nativeName.
func (r *AccountSchemaResource) apply(ctx context.Context, data *AccountSchemaResourceModel, client *Client, diags *diag.Diagnostics) {
	sourceID := data.SourceID.ValueString()
	schemaID := data.SchemaID.ValueString()

	existingSchema, err := client.GetAccountSchema(ctx, sourceID, schemaID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to get account schema %s of source %s: %s", schemaID, sourceID, err))
		return
	}
	existingSchema.SourceID = sourceID
	existingSchema.ID = schemaID

	setIfKnown := func(value types.String, target *string) {
		if !value.IsNull() && !value.IsUnknown() {
			*target = value.ValueString()
		}
	}
	setIfKnown(data.DisplayAttribute, &existingSchema.DisplayAttribute)
	setIfKnown(data.IdentityAttribute, &existingSchema.IdentityAttribute)
	setIfKnown(data.NativeObjectType, &existingSchema.NativeObjectType)
	setIfKnown(data.HierarchyAttribute, &existingSchema.HierarchyAttribute)
	if !data.IncludePermissions.IsNull() && !data.IncludePermissions.IsUnknown() {
		existingSchema.IncludePermissions = data.IncludePermissions.ValueBool()
	}

	existingSchema.Attributes = r.mergeAttributes(ctx, *data, existingSchema.Attributes, diags)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Account Schema", map[string]interface{}{"source_id": sourceID, "schema_id": schemaID})

	accountSchemaResponse, err := client.UpdateAccountSchema(ctx, existingSchema)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update account schema: %s", err))
		return
	}

	accountSchemaResponse.SourceID = sourceID
	r.setStateFromAPI(ctx, data, accountSchemaResponse, false, diags)
}

func (r *AccountSchemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := data.SourceID.ValueString()
	schemaID := data.SchemaID.ValueString()

	tflog.Info(ctx, "Reading Account Schema", map[string]interface{}{"source_id": sourceID})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	accountSchema, err := client.GetAccountSchema(ctx, sourceID, schemaID)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account schema: %s", err))
		return
	}

	accountSchema.SourceID = sourceID
	unmanaged, d := req.Private.GetKey(ctx, accountSchemaAttributesUnmanagedKey)
	resp.Diagnostics.Append(d...)
	priorAttributes := data.Attributes
	r.setStateFromAPI(ctx, &data, accountSchema, true, &resp.Diagnostics)
	if string(unmanaged) == "true" {
		// Attributes are not configured, so they are not tracked.
		data.Attributes = priorAttributes
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountSchemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccountSchemaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := data.SourceID.ValueString()

	tflog.Info(ctx, "Updating Account Schema", map[string]interface{}{"source_id": sourceID})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	r.apply(ctx, &data, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, accountSchemaAttributesUnmanagedKey, attributesUnmanagedMarker(data.Attributes))...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The resource manages attributes of a schema that belongs to the source, it does not create the schema.
	// Deleting the schema would break the source, so destroy only removes the resource from state.
	tflog.Info(ctx, "Removing Account Schema from state", map[string]interface{}{
		"source_id": data.SourceID.ValueString(),
		"schema_id": data.SchemaID.ValueString(),
	})
	resp.Diagnostics.AddWarning(
		"Account schema left in place",
		fmt.Sprintf("The account schema %s of source %s was removed from Terraform state only. Its attributes were not changed in IdentityNow.",
			data.SchemaID.ValueString(), data.SourceID.ValueString()),
	)
}

// mergeAttributes returns the planned attributes, each starting from the existing attribute with the
// same name so that fields not managed by Terraform are kept. Duplicate names are ignored.
func (r *AccountSchemaResource) mergeAttributes(ctx context.Context, data AccountSchemaResourceModel, existing []*AccountSchemaAttribute, diags *diag.Diagnostics) []*AccountSchemaAttribute {
	if data.Attributes.IsNull() || data.Attributes.IsUnknown() {
		return existing
	}

	var attrModels []AccountSchemaAttributeModel
	diags.Append(data.Attributes.ElementsAs(ctx, &attrModels, false)...)
	if diags.HasError() {
		return nil
	}
	// Without attribute blocks the attributes are not managed, so the existing ones are kept.
	if len(attrModels) == 0 {
		return existing
	}

	existingByName := make(map[string]*AccountSchemaAttribute, len(existing))
	for _, a := range existing {
		existingByName[a.Name] = a
	}

	seen := make(map[string]bool)
	attrs := make([]*AccountSchemaAttribute, 0, len(attrModels))
	for _, am := range attrModels {
		name := am.Name.ValueString()
		if seen[name] {
			continue
		}
		seen[name] = true

		attr := &AccountSchemaAttribute{Name: name}
		if current, ok := existingByName[name]; ok {
			copied := *current
			attr = &copied
		}
		if !am.Type.IsNull() && !am.Type.IsUnknown() {
			attr.Type = am.Type.ValueString()
		}
		if !am.Description.IsNull() && !am.Description.IsUnknown() {
			attr.Description = am.Description.ValueString()
		}
		if !am.IsGroup.IsNull() && !am.IsGroup.IsUnknown() {
			attr.IsGroup = am.IsGroup.ValueBool()
		}
		if !am.IsMultiValued.IsNull() && !am.IsMultiValued.IsUnknown() {
			attr.IsMultiValued = am.IsMultiValued.ValueBool()
		}
		if !am.IsEntitlement.IsNull() && !am.IsEntitlement.IsUnknown() {
			attr.IsEntitlement = am.IsEntitlement.ValueBool()
		}

		attr.Schema = nil
		if !am.Schema.IsNull() && !am.Schema.IsUnknown() {
			var schemaModels []AccountSchemaAttributeSchemaModel
			diags.Append(am.Schema.ElementsAs(ctx, &schemaModels, false)...)
			if len(schemaModels) > 0 {
				attr.Schema = &AccountSchemaAttributeSchema{
					ID:   schemaModels[0].ID.ValueString(),
					Name: schemaModels[0].Name.ValueString(),
					Type: schemaModels[0].Type.ValueString(),
				}
			}
		}

		attrs = append(attrs, attr)
	}
	return attrs
}

func accountSchemaAttributeSchemaObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"type": types.StringType,
	}}
}

func accountSchemaAttributeObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":            types.StringType,
		"type":            types.StringType,
		"description":     types.StringType,
		"is_group":        types.BoolType,
		"is_multi_valued": types.BoolType,
		"is_entitlement":  types.BoolType,
		"schema":          types.ListType{ElemType: accountSchemaAttributeSchemaObjectType()},
	}}
}

// setStateFromAPI maps an API account schema onto the model. With refresh set (Read), API values
// replace the model values and attributes follow the prior order. Otherwise (Create, Update) only
// unknown values are resolved, so the state matches the plan.
func (r *AccountSchemaResource) setStateFromAPI(ctx context.Context, data *AccountSchemaResourceModel, as *AccountSchema, refresh bool, diags *diag.Diagnostics) {
	data.ID = types.StringValue(as.ID)
	data.SourceID = types.StringValue(as.SourceID)
	data.SchemaID = types.StringValue(as.ID)
	data.Modified = types.StringValue(as.Modified)
	if refresh || data.Created.IsUnknown() {
		data.Created = types.StringValue(as.Created)
	}

	stringFromAPI := func(current types.String, value string) types.String {
		if refresh || current.IsUnknown() {
			return types.StringValue(value)
		}
		return current
	}
	data.Name = stringFromAPI(data.Name, as.Name)
	data.DisplayAttribute = stringFromAPI(data.DisplayAttribute, as.DisplayAttribute)
	data.IdentityAttribute = stringFromAPI(data.IdentityAttribute, as.IdentityAttribute)
	data.NativeObjectType = stringFromAPI(data.NativeObjectType, as.NativeObjectType)
	data.HierarchyAttribute = stringFromAPI(data.HierarchyAttribute, as.HierarchyAttribute)
	if refresh || data.IncludePermissions.IsUnknown() {
		data.IncludePermissions = types.BoolValue(as.IncludePermissions)
	}

	apiByName := make(map[string]*AccountSchemaAttribute, len(as.Attributes))
	for _, a := range as.Attributes {
		apiByName[a.Name] = a
	}

	var prior []AccountSchemaAttributeModel
	if !data.Attributes.IsNull() && !data.Attributes.IsUnknown() {
		diags.Append(data.Attributes.ElementsAs(ctx, &prior, false)...)
	}

	var models []AccountSchemaAttributeModel
	if refresh {
		// Keep the prior order, then append attributes added outside Terraform.
		seen := make(map[string]bool, len(as.Attributes))
		for _, p := range prior {
			if a, ok := apiByName[p.Name.ValueString()]; ok && !seen[a.Name] {
				seen[a.Name] = true
				models = append(models, accountSchemaAttributeModelFromAPI(ctx, a, diags))
			}
		}
		for _, a := range as.Attributes {
			if !seen[a.Name] {
				seen[a.Name] = true
				models = append(models, accountSchemaAttributeModelFromAPI(ctx, a, diags))
			}
		}
	} else {
		for _, p := range prior {
			if a, ok := apiByName[p.Name.ValueString()]; ok {
				fromAPI := accountSchemaAttributeModelFromAPI(ctx, a, diags)
				if p.Type.IsUnknown() {
					p.Type = fromAPI.Type
				}
				if p.Description.IsUnknown() {
					p.Description = fromAPI.Description
				}
				if p.IsGroup.IsUnknown() {
					p.IsGroup = fromAPI.IsGroup
				}
				if p.IsMultiValued.IsUnknown() {
					p.IsMultiValued = fromAPI.IsMultiValued
				}
				if p.IsEntitlement.IsUnknown() {
					p.IsEntitlement = fromAPI.IsEntitlement
				}
			} else {
				// Duplicate or rejected attributes are not in the response, resolve unknowns to empty values.
				for _, v := range []*types.String{&p.Type, &p.Description} {
					if v.IsUnknown() {
						*v = types.StringValue("")
					}
				}
				for _, v := range []*types.Bool{&p.IsGroup, &p.IsMultiValued, &p.IsEntitlement} {
					if v.IsUnknown() {
						*v = types.BoolValue(false)
					}
				}
			}
			models = append(models, p)
		}
	}

	if !refresh && data.Attributes.IsNull() {
		return
	}
	attrList, d := types.ListValueFrom(ctx, accountSchemaAttributeObjectType(), models)
	diags.Append(d...)
	if models == nil {
		attrList, _ = types.ListValue(accountSchemaAttributeObjectType(), []attr.Value{})
	}
	data.Attributes = attrList
}

func accountSchemaAttributeModelFromAPI(ctx context.Context, a *AccountSchemaAttribute, diags *diag.Diagnostics) AccountSchemaAttributeModel {
	schemaList, _ := types.ListValue(accountSchemaAttributeSchemaObjectType(), []attr.Value{})
	if a.Schema != nil {
		sl, d := types.ListValueFrom(ctx, accountSchemaAttributeSchemaObjectType(), []AccountSchemaAttributeSchemaModel{{
			ID:   types.StringValue(a.Schema.ID),
			Name: types.StringValue(a.Schema.Name),
			Type: types.StringValue(a.Schema.Type),
		}})
		diags.Append(d...)
		schemaList = sl
	}
	return AccountSchemaAttributeModel{
		Name:          types.StringValue(a.Name),
		Type:          types.StringValue(a.Type),
		Description:   types.StringValue(a.Description),
		IsGroup:       types.BoolValue(a.IsGroup),
		IsMultiValued: types.BoolValue(a.IsMultiValued),
		IsEntitlement: types.BoolValue(a.IsEntitlement),
		Schema:        schemaList,
	}
}

func (r *AccountSchemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceID, schemaID, ok := strings.Cut(req.ID, "/")
	if !ok || sourceID == "" || schemaID == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <source_id>/<schema_id>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_id"), sourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema_id"), schemaID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), schemaID)...)
}

// accountSchemaAttributesUnmanagedKey is the private state key that records that no attribute
// blocks are configured. The framework represents an empty block list as null, the same as
// after import, so the marker distinguishes "not managed" from "not read yet".
const accountSchemaAttributesUnmanagedKey = "attributes_unmanaged"

func attributesUnmanagedMarker(attributes types.List) []byte {
	if attributes.IsNull() || len(attributes.Elements()) == 0 {
		return []byte("true")
	}
	return []byte("false")
}
