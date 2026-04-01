package main

import (
	"context"
	"fmt"

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

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource defines the resource implementation
type RoleResource struct {
	client *Config
}

// RoleResourceModel describes the resource data model
type RoleResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Owner               types.List   `tfsdk:"owner"`
	AccessProfiles      types.List   `tfsdk:"access_profiles"`
	Entitlements        types.List   `tfsdk:"entitlements"`
	Membership          types.List   `tfsdk:"membership"`
	AccessModelMetadata types.List   `tfsdk:"access_model_metadata"`
	AccessRequestConfig types.List   `tfsdk:"access_request_config"`
	Requestable         types.Bool   `tfsdk:"requestable"`
	Dimensional         types.Bool   `tfsdk:"dimensional"`
	Enabled             types.Bool   `tfsdk:"enabled"`
}

type AccessModelMetadataModel struct {
	Attributes types.List `tfsdk:"attributes"`
}

type AccessModelMetadataAttributeModel struct {
	Key    types.String `tfsdk:"key"`
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type AccessModelMetadataValueModel struct {
	Value  types.String `tfsdk:"value"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

type RoleAccessRequestConfigModel struct {
	CommentsRequired       types.Bool `tfsdk:"comments_required"`
	DenialCommentsRequired types.Bool `tfsdk:"denial_comments_required"`
	ApprovalSchemes        types.List `tfsdk:"approval_schemes"`
	DimensionSchema        types.List `tfsdk:"dimension_schema"`
}

type RoleDimensionSchemaModel struct {
	DimensionAttributes types.List `tfsdk:"dimension_attributes"`
}

type DimensionAttributeRefModel struct {
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Derived     types.Bool   `tfsdk:"derived"`
}

type OwnerModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

type AccessProfileRefModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

type MembershipModel struct {
	Type     types.String `tfsdk:"type"`
	Criteria types.List   `tfsdk:"criteria"`
}

type CriteriaModel struct {
	Operation   types.String `tfsdk:"operation"`
	StringValue types.String `tfsdk:"string_value"`
	Key         types.List   `tfsdk:"key"`
	Children    types.List   `tfsdk:"children"`
}

type CriteriaChildModel struct {
	Operation   types.String `tfsdk:"operation"`
	StringValue types.String `tfsdk:"string_value"`
	Key         types.List   `tfsdk:"key"`
	Children    types.List   `tfsdk:"children"`
}

type CriteriaLeafModel struct {
	Operation   types.String `tfsdk:"operation"`
	StringValue types.String `tfsdk:"string_value"`
	Key         types.List   `tfsdk:"key"`
}

type CriteriaKeyModel struct {
	Type     types.String `tfsdk:"type"`
	Property types.String `tfsdk:"property"`
	SourceId types.String `tfsdk:"source_id"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Role resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Role ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Role name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				}},
			"description": schema.StringAttribute{
				MarkdownDescription: "Role description",
				Optional:            true,
			},
			"requestable": schema.BoolAttribute{
				MarkdownDescription: "Whether this role is requestable",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dimensional": schema.BoolAttribute{
				MarkdownDescription: "Whether this role is dimensional",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether this role is enabled",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Role owner",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Owner ID",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Owner type",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Owner name",
							Required:            true,
						},
					},
				},
			},
			"access_profiles": schema.ListNestedBlock{
				MarkdownDescription: "Access profiles assigned to this role",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Access profile ID",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Access profile type",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Access profile name",
							Required:            true,
						},
					},
				},
			},
			"entitlements": schema.ListNestedBlock{
				MarkdownDescription: "Entitlements assigned to this role",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Entitlement ID",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Entitlement type",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Entitlement name",
							Required:            true,
						},
					},
				},
			},
			"access_model_metadata": schema.ListNestedBlock{
				MarkdownDescription: "Access model metadata for this role",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"attributes": schema.ListNestedBlock{
							MarkdownDescription: "Metadata attributes",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										MarkdownDescription: "Unique identifier for the metadata type (e.g. iscPrivacy)",
										Required:            true,
									},
									"name": schema.StringAttribute{
										MarkdownDescription: "Human readable name of the metadata attribute",
										Required:            true,
									},
								},
								Blocks: map[string]schema.Block{
									"values": schema.ListNestedBlock{
										MarkdownDescription: "Values assigned to this metadata attribute",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"value": schema.StringAttribute{
													MarkdownDescription: "The metadata value",
													Required:            true,
												},
												"name": schema.StringAttribute{
													MarkdownDescription: "Human readable name of the value",
													Required:            true,
												},
												"status": schema.StringAttribute{
													MarkdownDescription: "Status of the value (e.g. active)",
													Optional:            true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"access_request_config": schema.ListNestedBlock{
				MarkdownDescription: "Access request configuration for this role",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"comments_required": schema.BoolAttribute{
							MarkdownDescription: "Whether comments are required when requesting access",
							Optional:            true,
						},
						"denial_comments_required": schema.BoolAttribute{
							MarkdownDescription: "Whether comments are required when denying access",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"approval_schemes": schema.ListNestedBlock{
							MarkdownDescription: "Approval schemes for this role",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"approver_type": schema.StringAttribute{
										MarkdownDescription: "Type of approver (e.g. APP_OWNER, MANAGER, GOVERNANCE_GROUP)",
										Required:            true,
									},
									"approver_id": schema.StringAttribute{
										MarkdownDescription: "ID of the approver (required for GOVERNANCE_GROUP type)",
										Optional:            true,
									},
								},
							},
						},
						"dimension_schema": schema.ListNestedBlock{
							MarkdownDescription: "Dimension schema for dimensional roles",
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"dimension_attributes": schema.ListNestedBlock{
										MarkdownDescription: "Dimension attributes that define this dimension",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													MarkdownDescription: "The attribute name",
													Required:            true,
												},
												"display_name": schema.StringAttribute{
													MarkdownDescription: "The display name of the attribute",
													Optional:            true,
													Computed:            true,
												},
												"derived": schema.BoolAttribute{
													MarkdownDescription: "Whether the attribute is derived",
													Optional:            true,
													Computed:            true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"membership": schema.ListNestedBlock{
				MarkdownDescription: "Role membership definition. Defines how identities are assigned to this role.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Membership type (STANDARD or IDENTITY_LIST)",
							Required:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"criteria": schema.ListNestedBlock{
							MarkdownDescription: "Membership criteria",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"operation": schema.StringAttribute{
										MarkdownDescription: "Criteria operation (EQUALS, NOT_EQUALS, CONTAINS, AND, OR, etc.)",
										Required:            true,
									},
									"string_value": schema.StringAttribute{
										MarkdownDescription: "Value to match against",
										Optional:            true,
									},
								},
								Blocks: map[string]schema.Block{
									"key": schema.ListNestedBlock{
										MarkdownDescription: "Criteria key identifying the identity attribute",
										NestedObject:        criteriaKeyBlockObject(),
									},
									"children": schema.ListNestedBlock{
										MarkdownDescription: "Child criteria (level 2)",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"operation": schema.StringAttribute{
													MarkdownDescription: "Criteria operation",
													Required:            true,
												},
												"string_value": schema.StringAttribute{
													MarkdownDescription: "Value to match against",
													Optional:            true,
												},
											},
											Blocks: map[string]schema.Block{
												"key": schema.ListNestedBlock{
													MarkdownDescription: "Criteria key identifying the identity attribute",
													NestedObject:        criteriaKeyBlockObject(),
												},
												"children": schema.ListNestedBlock{
													MarkdownDescription: "Child criteria (level 3)",
													NestedObject: schema.NestedBlockObject{
														Attributes: map[string]schema.Attribute{
															"operation": schema.StringAttribute{
																MarkdownDescription: "Criteria operation",
																Required:            true,
															},
															"string_value": schema.StringAttribute{
																MarkdownDescription: "Value to match against",
																Optional:            true,
															},
														},
														Blocks: map[string]schema.Block{
															"key": schema.ListNestedBlock{
																MarkdownDescription: "Criteria key identifying the identity attribute",
																NestedObject:        criteriaKeyBlockObject(),
															},
														},
													},
												},
											},
										},
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

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build Role object
	role := &Role{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	// Parse owner
	var owners []OwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		role.RoleOwner = &ObjectInfo{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	// Parse access profiles
	if !data.AccessProfiles.IsNull() {
		var aps []AccessProfileRefModel
		resp.Diagnostics.Append(data.AccessProfiles.ElementsAs(ctx, &aps, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		role.AccessProfiles = make([]*ObjectInfo, len(aps))
		for i, ap := range aps {
			role.AccessProfiles[i] = &ObjectInfo{
				ID:   ap.ID.ValueString(),
				Type: ap.Type.ValueString(),
				Name: ap.Name.ValueString(),
			}
		}
	}

	// Parse entitlements
	if !data.Entitlements.IsNull() {
		var ents []EntitlementRefModel
		resp.Diagnostics.Append(data.Entitlements.ElementsAs(ctx, &ents, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		role.Entitlements = make([]*ObjectInfo, len(ents))
		for i, e := range ents {
			role.Entitlements[i] = &ObjectInfo{
				ID:   e.ID.ValueString(),
				Type: e.Type.ValueString(),
				Name: e.Name.ValueString(),
			}
		}
	}

	if !data.Requestable.IsNull() {
		requestable := data.Requestable.ValueBool()
		role.Requestable = &requestable
	}

	if !data.Dimensional.IsNull() {
		dimensional := data.Dimensional.ValueBool()
		role.Dimensional = &dimensional
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		role.Enabled = &enabled
	}

	// Parse membership
	if !data.Membership.IsNull() {
		var memberships []MembershipModel
		resp.Diagnostics.Append(data.Membership.ElementsAs(ctx, &memberships, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(memberships) > 0 {
			role.Membership = membershipModelToAPI(ctx, memberships[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Parse access model metadata
	if !data.AccessModelMetadata.IsNull() {
		role.AccessModelMetadata = accessModelMetadataModelToAPI(ctx, data.AccessModelMetadata, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Parse access request config
	if !data.AccessRequestConfig.IsNull() {
		role.AccessRequestConfig = roleAccessRequestConfigModelToAPI(ctx, data.AccessRequestConfig, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	tflog.Info(ctx, "Creating Role", map[string]interface{}{"name": role.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newRole, err := client.CreateRole(ctx, role)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role: %s", err))
		return
	}

	// Update state with returned values
	data.ID = types.StringValue(newRole.ID)
	if newRole.Requestable != nil {
		data.Requestable = types.BoolValue(*newRole.Requestable)
	}
	if newRole.Dimensional != nil {
		data.Dimensional = types.BoolValue(*newRole.Dimensional)
	}
	if newRole.Enabled != nil {
		data.Enabled = types.BoolValue(*newRole.Enabled)
	}

	// Map access request config from API response to resolve computed values
	data.AccessRequestConfig = roleAccessRequestConfigAPIToState(ctx, newRole.AccessRequestConfig, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map access model metadata from API response to resolve computed values
	data.AccessModelMetadata = accessModelMetadataAPIToState(ctx, newRole.AccessModelMetadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created a role resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Role", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	role, err := client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role: %s", err))
		return
	}

	// Update state from API response
	data.Name = types.StringValue(role.Name)
	if role.Description != "" {
		data.Description = types.StringValue(role.Description)
	} else if data.Description.IsNull() {
		// API returned null/empty and user didn't set description — keep null
	} else {
		data.Description = types.StringValue(role.Description)
	}

	if role.Requestable != nil {
		data.Requestable = types.BoolValue(*role.Requestable)
	}
	if role.Dimensional != nil {
		data.Dimensional = types.BoolValue(*role.Dimensional)
	}
	if role.Enabled != nil {
		data.Enabled = types.BoolValue(*role.Enabled)
	}

	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
		"name": types.StringType,
	}}

	if role.RoleOwner != nil {
		ownerModels := []OwnerModel{
			{
				ID:   types.StringValue(fmt.Sprintf("%v", role.RoleOwner.ID)),
				Type: types.StringValue(role.RoleOwner.Type),
				Name: types.StringValue(role.RoleOwner.Name),
			},
		}
		ownerList, diags := types.ListValueFrom(ctx, objType, ownerModels)
		resp.Diagnostics.Append(diags...)
		data.Owner = ownerList
	} else {
		data.Owner = types.ListNull(objType)
	}

	if role.AccessProfiles != nil {
		apModels := make([]AccessProfileRefModel, len(role.AccessProfiles))
		for i, ap := range role.AccessProfiles {
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
		data.AccessProfiles = types.ListNull(objType)
	}

	if role.Entitlements != nil {
		entModels := make([]EntitlementRefModel, len(role.Entitlements))
		for i, e := range role.Entitlements {
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
		data.Entitlements = types.ListNull(objType)
	}

	// Map access model metadata from API response
	data.AccessModelMetadata = accessModelMetadataAPIToState(ctx, role.AccessModelMetadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map access request config from API response
	data.AccessRequestConfig = roleAccessRequestConfigAPIToState(ctx, role.AccessRequestConfig, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map membership from API response
	data.Membership = membershipAPIToState(ctx, role.Membership, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Role", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	// Build update patches for all mutable fields
	updatePatches := []*UpdateRole{}

	if !data.Description.IsNull() {
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/description",
			Value: data.Description.ValueString(),
		})
	}

	// Patch owner
	var owners []OwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		updatePatches = append(updatePatches, &UpdateRole{
			Op:   "replace",
			Path: "/owner",
			Value: map[string]interface{}{
				"id":   owners[0].ID.ValueString(),
				"type": owners[0].Type.ValueString(),
				"name": owners[0].Name.ValueString(),
			},
		})
	}

	// Patch access profiles
	if !data.AccessProfiles.IsNull() {
		var aps []AccessProfileRefModel
		resp.Diagnostics.Append(data.AccessProfiles.ElementsAs(ctx, &aps, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apValues := make([]interface{}, len(aps))
		for i, ap := range aps {
			apValues[i] = map[string]interface{}{
				"id":   ap.ID.ValueString(),
				"type": ap.Type.ValueString(),
				"name": ap.Name.ValueString(),
			}
		}
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/accessProfiles",
			Value: apValues,
		})
	}

	// Patch entitlements
	if !data.Entitlements.IsNull() {
		var ents []EntitlementRefModel
		resp.Diagnostics.Append(data.Entitlements.ElementsAs(ctx, &ents, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		entValues := make([]interface{}, len(ents))
		for i, e := range ents {
			entValues[i] = map[string]interface{}{
				"id":   e.ID.ValueString(),
				"type": e.Type.ValueString(),
				"name": e.Name.ValueString(),
			}
		}
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/entitlements",
			Value: entValues,
		})
	}

	// Patch requestable
	if !data.Requestable.IsNull() {
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/requestable",
			Value: data.Requestable.ValueBool(),
		})
	}

	// Patch dimensional
	if !data.Dimensional.IsNull() {
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/dimensional",
			Value: data.Dimensional.ValueBool(),
		})
	}

	// Patch enabled
	if !data.Enabled.IsNull() {
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/enabled",
			Value: data.Enabled.ValueBool(),
		})
	}

	// Patch membership
	if !data.Membership.IsNull() {
		var memberships []MembershipModel
		resp.Diagnostics.Append(data.Membership.ElementsAs(ctx, &memberships, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(memberships) > 0 {
			membership := membershipModelToAPI(ctx, memberships[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
			updatePatches = append(updatePatches, &UpdateRole{
				Op:    "replace",
				Path:  "/membership",
				Value: membership,
			})
		}
	}

	// Patch access model metadata
	if !data.AccessModelMetadata.IsNull() {
		metadata := accessModelMetadataModelToAPI(ctx, data.AccessModelMetadata, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/accessModelMetadata",
			Value: metadata,
		})
	}

	// Patch access request config
	if !data.AccessRequestConfig.IsNull() {
		arcValue := roleAccessRequestConfigModelToAPI(ctx, data.AccessRequestConfig, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		updatePatches = append(updatePatches, &UpdateRole{
			Op:    "replace",
			Path:  "/accessRequestConfig",
			Value: arcValue,
		})
	}

	_, err = client.UpdateRole(ctx, updatePatches, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update role: %s", err))
		return
	}

	// Re-read the role to resolve computed values
	updatedRole, err := client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role after update: %s", err))
		return
	}

	// Map access request config from API response to resolve computed values
	data.AccessRequestConfig = roleAccessRequestConfigAPIToState(ctx, updatedRole.AccessRequestConfig, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map access model metadata from API response to resolve computed values
	data.AccessModelMetadata = accessModelMetadataAPIToState(ctx, updatedRole.AccessModelMetadata, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if updatedRole.Requestable != nil {
		data.Requestable = types.BoolValue(*updatedRole.Requestable)
	}
	if updatedRole.Dimensional != nil {
		data.Dimensional = types.BoolValue(*updatedRole.Dimensional)
	}
	if updatedRole.Enabled != nil {
		data.Enabled = types.BoolValue(*updatedRole.Enabled)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Role", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	role, err := client.GetRole(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get role: %s", err))
		return
	}

	_, err = client.DeleteRole(ctx, role)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role: %s", err))
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// criteriaKeyBlockObject returns the reusable schema for a criteria key block.
func criteriaKeyBlockObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Key type (IDENTITY or ACCOUNT)",
				Required:            true,
			},
			"property": schema.StringAttribute{
				MarkdownDescription: "Identity or account attribute name (e.g. attribute.department)",
				Required:            true,
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "Source ID (required when type is ACCOUNT)",
				Optional:            true,
			},
		},
	}
}

// membershipModelToAPI converts the Terraform MembershipModel to the API RoleMembership struct.
func membershipModelToAPI(ctx context.Context, m MembershipModel, diags *diag.Diagnostics) *RoleMembership {
	membership := &RoleMembership{
		Type: m.Type.ValueString(),
	}

	if !m.Criteria.IsNull() && len(m.Criteria.Elements()) > 0 {
		var criteriaModels []CriteriaModel
		diags.Append(m.Criteria.ElementsAs(ctx, &criteriaModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(criteriaModels) > 0 {
			membership.Criteria = criteriaModelToAPI(ctx, criteriaModels[0], diags)
			if diags.HasError() {
				return nil
			}
		}
	}

	return membership
}

// criteriaModelToAPI converts a top-level CriteriaModel to the API RoleMembershipCriteria.
func criteriaModelToAPI(ctx context.Context, c CriteriaModel, diags *diag.Diagnostics) *RoleMembershipCriteria {
	criteria := &RoleMembershipCriteria{
		Operation: c.Operation.ValueString(),
	}

	if !c.StringValue.IsNull() {
		criteria.StringValue = c.StringValue.ValueString()
	}

	// Parse key
	if !c.Key.IsNull() && len(c.Key.Elements()) > 0 {
		var keys []CriteriaKeyModel
		diags.Append(c.Key.ElementsAs(ctx, &keys, false)...)
		if diags.HasError() {
			return nil
		}
		if len(keys) > 0 {
			criteria.Key = criteriaKeyModelToAPI(keys[0])
		}
	}

	// Parse children (level 2)
	if !c.Children.IsNull() && len(c.Children.Elements()) > 0 {
		var childModels []CriteriaChildModel
		diags.Append(c.Children.ElementsAs(ctx, &childModels, false)...)
		if diags.HasError() {
			return nil
		}
		criteria.Children = make([]*RoleMembershipCriteria, len(childModels))
		for i, child := range childModels {
			criteria.Children[i] = criteriaChildModelToAPI(ctx, child, diags)
			if diags.HasError() {
				return nil
			}
		}
	}

	return criteria
}

// criteriaChildModelToAPI converts a level-2 CriteriaChildModel to the API type.
func criteriaChildModelToAPI(ctx context.Context, c CriteriaChildModel, diags *diag.Diagnostics) *RoleMembershipCriteria {
	criteria := &RoleMembershipCriteria{
		Operation: c.Operation.ValueString(),
	}

	if !c.StringValue.IsNull() {
		criteria.StringValue = c.StringValue.ValueString()
	}

	if !c.Key.IsNull() && len(c.Key.Elements()) > 0 {
		var keys []CriteriaKeyModel
		diags.Append(c.Key.ElementsAs(ctx, &keys, false)...)
		if diags.HasError() {
			return nil
		}
		if len(keys) > 0 {
			criteria.Key = criteriaKeyModelToAPI(keys[0])
		}
	}

	// Parse children (level 3 - leaf)
	if !c.Children.IsNull() && len(c.Children.Elements()) > 0 {
		var leafModels []CriteriaLeafModel
		diags.Append(c.Children.ElementsAs(ctx, &leafModels, false)...)
		if diags.HasError() {
			return nil
		}
		criteria.Children = make([]*RoleMembershipCriteria, len(leafModels))
		for i, leaf := range leafModels {
			criteria.Children[i] = criteriaLeafModelToAPI(ctx, leaf, diags)
			if diags.HasError() {
				return nil
			}
		}
	}

	return criteria
}

// criteriaLeafModelToAPI converts a level-3 CriteriaLeafModel to the API type.
func criteriaLeafModelToAPI(ctx context.Context, c CriteriaLeafModel, diags *diag.Diagnostics) *RoleMembershipCriteria {
	criteria := &RoleMembershipCriteria{
		Operation: c.Operation.ValueString(),
	}

	if !c.StringValue.IsNull() {
		criteria.StringValue = c.StringValue.ValueString()
	}

	if !c.Key.IsNull() && len(c.Key.Elements()) > 0 {
		var keys []CriteriaKeyModel
		diags.Append(c.Key.ElementsAs(ctx, &keys, false)...)
		if diags.HasError() {
			return nil
		}
		if len(keys) > 0 {
			criteria.Key = criteriaKeyModelToAPI(keys[0])
		}
	}

	return criteria
}

// criteriaKeyModelToAPI converts a CriteriaKeyModel to the API RoleKey.
func criteriaKeyModelToAPI(k CriteriaKeyModel) *RoleKey {
	key := &RoleKey{
		Type:     k.Type.ValueString(),
		Property: k.Property.ValueString(),
	}
	if !k.SourceId.IsNull() && !k.SourceId.IsUnknown() {
		key.SourceId = k.SourceId.ValueString()
	}
	return key
}

// membershipAPIToState converts the API RoleMembership to Terraform state list value.
func membershipAPIToState(ctx context.Context, m *RoleMembership, diags *diag.Diagnostics) types.List {
	membershipObjType := membershipObjectType()

	if m == nil {
		return types.ListNull(membershipObjType)
	}

	model := MembershipModel{
		Type: types.StringValue(m.Type),
	}

	if m.Criteria != nil {
		model.Criteria = criteriaAPIToState(ctx, m.Criteria, diags)
	} else {
		model.Criteria = criteriaEmptyList()
	}

	list, d := types.ListValueFrom(ctx, membershipObjType, []MembershipModel{model})
	diags.Append(d...)
	return list
}

// criteriaAPIToState converts API RoleMembershipCriteria to a Terraform list of CriteriaModel.
func criteriaAPIToState(ctx context.Context, c *RoleMembershipCriteria, diags *diag.Diagnostics) types.List {
	model := CriteriaModel{
		Operation: types.StringValue(c.Operation),
	}

	if c.StringValue != "" {
		model.StringValue = types.StringValue(c.StringValue)
	} else {
		model.StringValue = types.StringNull()
	}

	// Map key
	model.Key = criteriaKeyAPIToState(ctx, c.Key, diags)

	// Map children (level 2)
	if len(c.Children) > 0 {
		childModels := make([]CriteriaChildModel, len(c.Children))
		for i, child := range c.Children {
			childModels[i] = criteriaChildAPIToModel(ctx, child, diags)
			if diags.HasError() {
				return types.ListNull(criteriaObjectType())
			}
		}
		childList, d := types.ListValueFrom(ctx, criteriaChildObjectType(), childModels)
		diags.Append(d...)
		model.Children = childList
	} else {
		model.Children, _ = types.ListValue(criteriaChildObjectType(), []attr.Value{})
	}

	list, d := types.ListValueFrom(ctx, criteriaObjectType(), []CriteriaModel{model})
	diags.Append(d...)
	return list
}

// criteriaChildAPIToModel converts an API RoleMembershipCriteria (level 2) to CriteriaChildModel.
func criteriaChildAPIToModel(ctx context.Context, c *RoleMembershipCriteria, diags *diag.Diagnostics) CriteriaChildModel {
	model := CriteriaChildModel{
		Operation: types.StringValue(c.Operation),
	}

	if c.StringValue != "" {
		model.StringValue = types.StringValue(c.StringValue)
	} else {
		model.StringValue = types.StringNull()
	}

	model.Key = criteriaKeyAPIToState(ctx, c.Key, diags)

	// Map children (level 3 - leaf)
	if len(c.Children) > 0 {
		leafModels := make([]CriteriaLeafModel, len(c.Children))
		for i, child := range c.Children {
			leafModels[i] = criteriaLeafAPIToModel(ctx, child, diags)
			if diags.HasError() {
				return model
			}
		}
		leafList, d := types.ListValueFrom(ctx, criteriaLeafObjectType(), leafModels)
		diags.Append(d...)
		model.Children = leafList
	} else {
		model.Children, _ = types.ListValue(criteriaLeafObjectType(), []attr.Value{})
	}

	return model
}

// criteriaLeafAPIToModel converts an API RoleMembershipCriteria (level 3) to CriteriaLeafModel.
func criteriaLeafAPIToModel(ctx context.Context, c *RoleMembershipCriteria, diags *diag.Diagnostics) CriteriaLeafModel {
	model := CriteriaLeafModel{
		Operation: types.StringValue(c.Operation),
	}

	if c.StringValue != "" {
		model.StringValue = types.StringValue(c.StringValue)
	} else {
		model.StringValue = types.StringNull()
	}
	model.Key = criteriaKeyAPIToState(ctx, c.Key, diags)
	return model
}

// criteriaKeyAPIToState converts an API RoleKey to a Terraform list value.
func criteriaKeyAPIToState(ctx context.Context, k *RoleKey, diags *diag.Diagnostics) types.List {
	keyObjType := criteriaKeyObjectType()

	if k == nil {
		val, d := types.ListValue(keyObjType, []attr.Value{})
		diags.Append(d...)
		return val
	}

	keyModel := CriteriaKeyModel{
		Type:     types.StringValue(k.Type),
		Property: types.StringValue(fmt.Sprintf("%v", k.Property)),
	}
	if k.SourceId != nil && fmt.Sprintf("%v", k.SourceId) != "" && fmt.Sprintf("%v", k.SourceId) != "<nil>" {
		keyModel.SourceId = types.StringValue(fmt.Sprintf("%v", k.SourceId))
	} else {
		keyModel.SourceId = types.StringNull()
	}

	list, d := types.ListValueFrom(ctx, keyObjType, []CriteriaKeyModel{keyModel})
	diags.Append(d...)
	return list
}

// criteriaEmptyList returns an empty typed list for criteria.
func criteriaEmptyList() types.List {
	val, _ := types.ListValue(criteriaObjectType(), []attr.Value{})
	return val
}

// Object type definitions for Terraform Framework list value construction.

func membershipObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type":     types.StringType,
		"criteria": types.ListType{ElemType: criteriaObjectType()},
	}}
}

func criteriaObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation":    types.StringType,
		"string_value": types.StringType,
		"key":          types.ListType{ElemType: criteriaKeyObjectType()},
		"children":     types.ListType{ElemType: criteriaChildObjectType()},
	}}
}

func criteriaChildObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation":    types.StringType,
		"string_value": types.StringType,
		"key":          types.ListType{ElemType: criteriaKeyObjectType()},
		"children":     types.ListType{ElemType: criteriaLeafObjectType()},
	}}
}

func criteriaLeafObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation":    types.StringType,
		"string_value": types.StringType,
		"key":          types.ListType{ElemType: criteriaKeyObjectType()},
	}}
}

func criteriaKeyObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type":      types.StringType,
		"property":  types.StringType,
		"source_id": types.StringType,
	}}
}

// accessModelMetadataModelToAPI converts the Terraform state list to the API AttributeDTOList.
func accessModelMetadataModelToAPI(ctx context.Context, metadataList types.List, diags *diag.Diagnostics) *AttributeDTOList {
	if metadataList.IsNull() || len(metadataList.Elements()) == 0 {
		return nil
	}

	var metadataModels []AccessModelMetadataModel
	diags.Append(metadataList.ElementsAs(ctx, &metadataModels, false)...)
	if diags.HasError() {
		return nil
	}

	if len(metadataModels) == 0 {
		return nil
	}

	result := &AttributeDTOList{}
	m := metadataModels[0]

	if !m.Attributes.IsNull() && len(m.Attributes.Elements()) > 0 {
		var attrModels []AccessModelMetadataAttributeModel
		diags.Append(m.Attributes.ElementsAs(ctx, &attrModels, false)...)
		if diags.HasError() {
			return nil
		}

		result.Attributes = make([]*AccessModelMetadataAttribute, len(attrModels))
		for i, am := range attrModels {
			apiAttr := &AccessModelMetadataAttribute{
				Key:  am.Key.ValueString(),
				Name: am.Name.ValueString(),
			}

			if !am.Values.IsNull() && len(am.Values.Elements()) > 0 {
				var valModels []AccessModelMetadataValueModel
				diags.Append(am.Values.ElementsAs(ctx, &valModels, false)...)
				if diags.HasError() {
					return nil
				}

				apiAttr.Values = make([]*AccessModelMetadataValue, len(valModels))
				for j, vm := range valModels {
					apiVal := &AccessModelMetadataValue{
						Value: vm.Value.ValueString(),
						Name:  vm.Name.ValueString(),
					}
					if !vm.Status.IsNull() {
						apiVal.Status = vm.Status.ValueString()
					}
					apiAttr.Values[j] = apiVal
				}
			}

			result.Attributes[i] = apiAttr
		}
	}

	return result
}

// accessModelMetadataAPIToState converts the API AttributeDTOList to a Terraform state list.
func accessModelMetadataAPIToState(ctx context.Context, metadata *AttributeDTOList, diags *diag.Diagnostics) types.List {
	metadataObjType := accessModelMetadataObjectType()

	if metadata == nil || len(metadata.Attributes) == 0 {
		return types.ListNull(metadataObjType)
	}

	attrModels := make([]AccessModelMetadataAttributeModel, len(metadata.Attributes))
	for i, a := range metadata.Attributes {
		attrModel := AccessModelMetadataAttributeModel{
			Key:  types.StringValue(a.Key),
			Name: types.StringValue(a.Name),
		}

		if len(a.Values) > 0 {
			valModels := make([]AccessModelMetadataValueModel, len(a.Values))
			for j, v := range a.Values {
				valModel := AccessModelMetadataValueModel{
					Value: types.StringValue(v.Value),
					Name:  types.StringValue(v.Name),
				}
				if v.Status != "" {
					valModel.Status = types.StringValue(v.Status)
				} else {
					valModel.Status = types.StringNull()
				}
				valModels[j] = valModel
			}
			valList, d := types.ListValueFrom(ctx, accessModelMetadataValueObjectType(), valModels)
			diags.Append(d...)
			attrModel.Values = valList
		} else {
			attrModel.Values = types.ListNull(accessModelMetadataValueObjectType())
		}

		attrModels[i] = attrModel
	}

	attrList, d := types.ListValueFrom(ctx, accessModelMetadataAttributeObjectType(), attrModels)
	diags.Append(d...)

	model := AccessModelMetadataModel{
		Attributes: attrList,
	}

	list, d := types.ListValueFrom(ctx, metadataObjType, []AccessModelMetadataModel{model})
	diags.Append(d...)
	return list
}

func accessModelMetadataObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"attributes": types.ListType{ElemType: accessModelMetadataAttributeObjectType()},
	}}
}

func accessModelMetadataAttributeObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"key":    types.StringType,
		"name":   types.StringType,
		"values": types.ListType{ElemType: accessModelMetadataValueObjectType()},
	}}
}

func accessModelMetadataValueObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"value":  types.StringType,
		"name":   types.StringType,
		"status": types.StringType,
	}}
}

// roleAccessRequestConfigModelToAPI converts the Terraform access_request_config list to the API struct.
func roleAccessRequestConfigModelToAPI(ctx context.Context, configList types.List, diags *diag.Diagnostics) *RoleAccessRequestConfig {
	if configList.IsNull() || len(configList.Elements()) == 0 {
		return nil
	}

	var configModels []RoleAccessRequestConfigModel
	diags.Append(configList.ElementsAs(ctx, &configModels, false)...)
	if diags.HasError() {
		return nil
	}

	if len(configModels) == 0 {
		return nil
	}

	m := configModels[0]
	config := &RoleAccessRequestConfig{}

	if !m.CommentsRequired.IsNull() {
		v := m.CommentsRequired.ValueBool()
		config.CommentsRequired = &v
	}
	if !m.DenialCommentsRequired.IsNull() {
		v := m.DenialCommentsRequired.ValueBool()
		config.DenialCommentsRequired = &v
	}

	if !m.ApprovalSchemes.IsNull() && len(m.ApprovalSchemes.Elements()) > 0 {
		var schemes []ApprovalSchemeModel
		diags.Append(m.ApprovalSchemes.ElementsAs(ctx, &schemes, false)...)
		if diags.HasError() {
			return nil
		}
		config.ApprovalSchemes = make([]*ApprovalSchemes, len(schemes))
		for i, s := range schemes {
			scheme := &ApprovalSchemes{
				ApproverType: s.ApproverType.ValueString(),
			}
			if !s.ApproverID.IsNull() {
				scheme.ApproverId = s.ApproverID.ValueString()
			}
			config.ApprovalSchemes[i] = scheme
		}
	}

	if !m.DimensionSchema.IsNull() && len(m.DimensionSchema.Elements()) > 0 {
		var dsModels []RoleDimensionSchemaModel
		diags.Append(m.DimensionSchema.ElementsAs(ctx, &dsModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(dsModels) > 0 {
			ds := dsModels[0]
			dimSchema := &RoleDimensionSchema{}
			if !ds.DimensionAttributes.IsNull() && len(ds.DimensionAttributes.Elements()) > 0 {
				var daModels []DimensionAttributeRefModel
				diags.Append(ds.DimensionAttributes.ElementsAs(ctx, &daModels, false)...)
				if diags.HasError() {
					return nil
				}
				dimSchema.DimensionAttributes = make([]*DimensionAttributeRef, len(daModels))
				for j, da := range daModels {
					attrRef := &DimensionAttributeRef{
						Name:        da.Name.ValueString(),
						DisplayName: da.DisplayName.ValueString(),
					}
					v := da.Derived.ValueBool()
					attrRef.Derived = &v
					dimSchema.DimensionAttributes[j] = attrRef
				}
			}
			config.DimensionSchema = dimSchema
		}
	}

	return config
}

// roleAccessRequestConfigAPIToState converts the API RoleAccessRequestConfig to a Terraform state list.
func roleAccessRequestConfigAPIToState(ctx context.Context, config *RoleAccessRequestConfig, diags *diag.Diagnostics) types.List {
	arcObjType := roleAccessRequestConfigObjectType()

	if config == nil {
		return types.ListNull(arcObjType)
	}

	// Treat an empty config (no meaningful fields set) as null to avoid perpetual diffs
	if config.CommentsRequired == nil && config.DenialCommentsRequired == nil &&
		len(config.ApprovalSchemes) == 0 && config.DimensionSchema == nil {
		return types.ListNull(arcObjType)
	}

	model := RoleAccessRequestConfigModel{}

	if config.CommentsRequired != nil {
		model.CommentsRequired = types.BoolValue(*config.CommentsRequired)
	} else {
		model.CommentsRequired = types.BoolNull()
	}

	if config.DenialCommentsRequired != nil {
		model.DenialCommentsRequired = types.BoolValue(*config.DenialCommentsRequired)
	} else {
		model.DenialCommentsRequired = types.BoolNull()
	}

	approvalSchemeObjType := roleApprovalSchemeObjectType()
	if len(config.ApprovalSchemes) > 0 {
		schemeModels := make([]ApprovalSchemeModel, len(config.ApprovalSchemes))
		for i, s := range config.ApprovalSchemes {
			schemeModels[i] = ApprovalSchemeModel{
				ApproverType: types.StringValue(s.ApproverType),
			}
			if s.ApproverId != "" {
				schemeModels[i].ApproverID = types.StringValue(s.ApproverId)
			} else {
				schemeModels[i].ApproverID = types.StringNull()
			}
		}
		sl, d := types.ListValueFrom(ctx, approvalSchemeObjType, schemeModels)
		diags.Append(d...)
		model.ApprovalSchemes = sl
	} else {
		model.ApprovalSchemes, _ = types.ListValue(approvalSchemeObjType, []attr.Value{})
	}

	dimSchemaObjType := roleDimensionSchemaObjectType()
	if config.DimensionSchema != nil {
		ds := config.DimensionSchema
		dsModel := RoleDimensionSchemaModel{}
		dimAttrObjType := dimensionAttributeRefObjectType()
		if len(ds.DimensionAttributes) > 0 {
			daModels := make([]DimensionAttributeRefModel, len(ds.DimensionAttributes))
			for j, da := range ds.DimensionAttributes {
				daModel := DimensionAttributeRefModel{
					Name:        types.StringValue(da.Name),
					DisplayName: types.StringValue(da.DisplayName),
				}
				if da.Derived != nil {
					daModel.Derived = types.BoolValue(*da.Derived)
				} else {
					daModel.Derived = types.BoolValue(false)
				}
				daModels[j] = daModel
			}
			dal, d := types.ListValueFrom(ctx, dimAttrObjType, daModels)
			diags.Append(d...)
			dsModel.DimensionAttributes = dal
		} else {
			dsModel.DimensionAttributes, _ = types.ListValue(dimAttrObjType, []attr.Value{})
		}
		dsl, d := types.ListValueFrom(ctx, dimSchemaObjType, []RoleDimensionSchemaModel{dsModel})
		diags.Append(d...)
		model.DimensionSchema = dsl
	} else {
		model.DimensionSchema, _ = types.ListValue(dimSchemaObjType, []attr.Value{})
	}

	list, d := types.ListValueFrom(ctx, arcObjType, []RoleAccessRequestConfigModel{model})
	diags.Append(d...)
	return list
}

func roleAccessRequestConfigObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"comments_required":        types.BoolType,
		"denial_comments_required": types.BoolType,
		"approval_schemes":         types.ListType{ElemType: roleApprovalSchemeObjectType()},
		"dimension_schema":         types.ListType{ElemType: roleDimensionSchemaObjectType()},
	}}
}

func roleApprovalSchemeObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"approver_type": types.StringType,
		"approver_id":   types.StringType,
	}}
}

func roleDimensionSchemaObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"dimension_attributes": types.ListType{ElemType: dimensionAttributeRefObjectType()},
	}}
}

func dimensionAttributeRefObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":         types.StringType,
		"display_name": types.StringType,
		"derived":      types.BoolType,
	}}
}
