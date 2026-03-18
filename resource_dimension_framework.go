package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DimensionResource{}
var _ resource.ResourceWithImportState = &DimensionResource{}

func NewDimensionResource() resource.Resource {
	return &DimensionResource{}
}

type DimensionResource struct {
	client *Config
}

type DimensionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleID         types.String `tfsdk:"role_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Owner          types.List   `tfsdk:"owner"`
	AccessProfiles types.List   `tfsdk:"access_profiles"`
	Entitlements   types.List   `tfsdk:"entitlements"`
	Membership     types.List   `tfsdk:"membership"`
}

func (r *DimensionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dimension"
}

func (r *DimensionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Dimension resource. A dimension is a sub-division of a role that allows fine-grained access grouping.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Dimension ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the role this dimension belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Dimension name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Dimension description",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Dimension owner",
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
				MarkdownDescription: "Access profiles assigned to this dimension",
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
				MarkdownDescription: "Entitlements assigned to this dimension",
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
			"membership": schema.ListNestedBlock{
				MarkdownDescription: "Dimension membership definition. Defines how identities are assigned to this dimension.",
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

func (r *DimensionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DimensionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DimensionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dimension := &Dimension{
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
		dimension.Owner = &ObjectInfo{
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
		dimension.AccessProfiles = make([]*ObjectInfo, len(aps))
		for i, ap := range aps {
			dimension.AccessProfiles[i] = &ObjectInfo{
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
		dimension.Entitlements = make([]*ObjectInfo, len(ents))
		for i, e := range ents {
			dimension.Entitlements[i] = &ObjectInfo{
				ID:   e.ID.ValueString(),
				Type: e.Type.ValueString(),
				Name: e.Name.ValueString(),
			}
		}
	}

	// Parse membership
	if !data.Membership.IsNull() {
		var memberships []MembershipModel
		resp.Diagnostics.Append(data.Membership.ElementsAs(ctx, &memberships, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(memberships) > 0 {
			dimension.Membership = membershipModelToAPI(ctx, memberships[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	tflog.Info(ctx, "Creating Dimension", map[string]interface{}{
		"name":    dimension.Name,
		"role_id": data.RoleID.ValueString(),
	})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newDimension, err := client.CreateDimension(ctx, data.RoleID.ValueString(), dimension)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create dimension: %s", err))
		return
	}

	data.ID = types.StringValue(newDimension.ID)

	tflog.Trace(ctx, "created a dimension resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DimensionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DimensionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Dimension", map[string]interface{}{
		"id":      data.ID.ValueString(),
		"role_id": data.RoleID.ValueString(),
	})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	dimension, err := client.GetDimension(ctx, data.RoleID.ValueString(), data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read dimension: %s", err))
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

	// Map membership from API response
	data.Membership = membershipAPIToState(ctx, dimension.Membership, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DimensionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DimensionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Dimension", map[string]interface{}{
		"id":      data.ID.ValueString(),
		"role_id": data.RoleID.ValueString(),
	})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	updatePatches := []*UpdateDimension{
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
	}

	// Patch owner
	var owners []OwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		updatePatches = append(updatePatches, &UpdateDimension{
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
		updatePatches = append(updatePatches, &UpdateDimension{
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
		updatePatches = append(updatePatches, &UpdateDimension{
			Op:    "replace",
			Path:  "/entitlements",
			Value: entValues,
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
			updatePatches = append(updatePatches, &UpdateDimension{
				Op:    "replace",
				Path:  "/membership",
				Value: membership,
			})
		}
	}

	_, err = client.UpdateDimension(ctx, data.RoleID.ValueString(), data.ID.ValueString(), updatePatches)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dimension: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DimensionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DimensionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Dimension", map[string]interface{}{
		"id":      data.ID.ValueString(),
		"role_id": data.RoleID.ValueString(),
	})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	err = client.DeleteDimension(ctx, data.RoleID.ValueString(), data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete dimension: %s", err))
		return
	}
}

func (r *DimensionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'role_id/dimension_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
