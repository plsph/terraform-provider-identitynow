package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AccountSchemaResource{}

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
				MarkdownDescription: "Account Schema name",
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source ID",
			},
			"schema_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Schema ID",
			},
			"display_attribute": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Display attribute",
			},
			"identity_attribute": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Identity attribute",
			},
			"native_object_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Native object type",
			},
			"hierarchy_attribute": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Hierarchy attribute",
			},
			"include_permissions": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Include permissions",
			},
			"modified": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Last modified timestamp",
			},
			"created": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Creation timestamp",
			},
		},
		Blocks: map[string]schema.Block{
			"attributes": schema.ListNestedBlock{
				MarkdownDescription: "Schema attributes",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Attribute name",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Attribute type",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Attribute description",
						},
						"is_group": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether this is a group attribute",
						},
						"is_multi_valued": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether this attribute is multi-valued",
						},
						"is_entitlement": schema.BoolAttribute{
							Optional:            true,
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

	sourceID := data.SourceID.ValueString()
	schemaID := data.SchemaID.ValueString()

	// Get existing account schema
	existingSchema, err := client.GetAccountSchema(ctx, sourceID, schemaID)
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get account schema: %s", err))
		return
	}

	existingSchema.SourceID = sourceID
	existingSchema.ID = schemaID

	// Build attributes from plan
	attrs := r.buildAttributes(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deduplicate attributes
	seen := make(map[string]bool)
	var result []*AccountSchemaAttribute
	for _, attr := range attrs {
		if _, ok := seen[attr.Name]; !ok {
			seen[attr.Name] = true
			result = append(result, attr)
		}
	}
	existingSchema.Attributes = result

	tflog.Info(ctx, "Creating Account Schema Attribute", map[string]interface{}{"source_id": sourceID})

	accountSchemaResponse, err := client.UpdateAccountSchema(ctx, existingSchema)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update account schema: %s", err))
		return
	}

	accountSchemaResponse.SourceID = sourceID
	r.setStateFromAPI(ctx, &data, accountSchemaResponse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account schema: %s", err))
		return
	}

	if accountSchema.Attributes == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	accountSchema.SourceID = sourceID
	r.setStateFromAPI(ctx, &data, accountSchema)

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

	accountSchema := &AccountSchema{
		ID:                 data.SchemaID.ValueString(),
		SourceID:           sourceID,
		Name:               data.Name.ValueString(),
		DisplayAttribute:   data.DisplayAttribute.ValueString(),
		IdentityAttribute:  data.IdentityAttribute.ValueString(),
		NativeObjectType:   data.NativeObjectType.ValueString(),
		HierarchyAttribute: data.HierarchyAttribute.ValueString(),
	}

	if !data.IncludePermissions.IsNull() {
		accountSchema.IncludePermissions = data.IncludePermissions.ValueBool()
	}

	attrs := r.buildAttributes(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	accountSchema.Attributes = attrs

	_, err = client.UpdateAccountSchema(ctx, accountSchema)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update account schema: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountSchemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccountSchemaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := data.SourceID.ValueString()
	schemaID := data.SchemaID.ValueString()

	tflog.Info(ctx, "Deleting Account Schema", map[string]interface{}{"source_id": sourceID})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	accountSchema, err := client.GetAccountSchema(ctx, sourceID, schemaID)
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get account schema: %s", err))
		return
	}

	accountSchema.SourceID = sourceID

	err = client.DeleteAccountSchema(ctx, accountSchema)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete account schema: %s", err))
		return
	}
}

func (r *AccountSchemaResource) buildAttributes(ctx context.Context, data AccountSchemaResourceModel, diags *diag.Diagnostics) []*AccountSchemaAttribute {
	if data.Attributes.IsNull() {
		return nil
	}

	var attrModels []AccountSchemaAttributeModel
	diags.Append(data.Attributes.ElementsAs(ctx, &attrModels, false)...)
	if diags.HasError() {
		return nil
	}

	var attrs []*AccountSchemaAttribute
	for _, am := range attrModels {
		attr := &AccountSchemaAttribute{
			Name:          am.Name.ValueString(),
			Type:          am.Type.ValueString(),
			Description:   am.Description.ValueString(),
			IsGroup:       am.IsGroup.ValueBool(),
			IsMultiValued: am.IsMultiValued.ValueBool(),
			IsEntitlement: am.IsEntitlement.ValueBool(),
		}

		if !am.Schema.IsNull() {
			var schemaModels []AccountSchemaAttributeSchemaModel
			diags.Append(am.Schema.ElementsAs(ctx, &schemaModels, false)...)
			if diags.HasError() {
				return nil
			}
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

func (r *AccountSchemaResource) setStateFromAPI(ctx context.Context, data *AccountSchemaResourceModel, as *AccountSchema) {
	data.ID = types.StringValue(as.ID)
	data.Name = types.StringValue(as.Name)
	data.SourceID = types.StringValue(as.SourceID)
	data.SchemaID = types.StringValue(as.ID)
	data.DisplayAttribute = types.StringValue(as.DisplayAttribute)
	data.IdentityAttribute = types.StringValue(as.IdentityAttribute)
	data.NativeObjectType = types.StringValue(as.NativeObjectType)
	data.HierarchyAttribute = types.StringValue(as.HierarchyAttribute)
	data.IncludePermissions = types.BoolValue(as.IncludePermissions)
	data.Modified = types.StringValue(as.Modified)
	data.Created = types.StringValue(as.Created)
}
