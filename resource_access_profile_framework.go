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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AccessProfileResource{}
var _ resource.ResourceWithImportState = &AccessProfileResource{}
var _ resource.ResourceWithModifyPlan = &AccessProfileResource{}

func NewAccessProfileResource() resource.Resource {
	return &AccessProfileResource{}
}

type AccessProfileResource struct {
	client *Config
}

type AccessProfileResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Owner                   types.List   `tfsdk:"owner"`
	Source                  types.List   `tfsdk:"source"`
	Entitlements            types.List   `tfsdk:"entitlements"`
	AccessRequestConfig     types.List   `tfsdk:"access_request_config"`
	RevocationRequestConfig types.List   `tfsdk:"revocation_request_config"`
	Segments                types.List   `tfsdk:"segments"`
	AccessModelMetadata     types.List   `tfsdk:"access_model_metadata"`
	ProvisioningCriteria    types.List   `tfsdk:"provisioning_criteria"`
	AdditionalOwners        types.List   `tfsdk:"additional_owners"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Requestable             types.Bool   `tfsdk:"requestable"`
}

type EntitlementRefModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type AccessRequestConfigModel struct {
	CommentsRequired           types.Bool `tfsdk:"comments_required"`
	DenialCommentsRequired     types.Bool `tfsdk:"denial_comments_required"`
	ReauthorizationRequired    types.Bool `tfsdk:"reauthorization_required"`
	RequireEndDate             types.Bool `tfsdk:"require_end_date"`
	ApprovalSchemes            types.List `tfsdk:"approval_schemes"`
	MaxPermittedAccessDuration types.List `tfsdk:"max_permitted_access_duration"`
}

type ApprovalSchemeModel struct {
	ApproverType types.String `tfsdk:"approver_type"`
	ApproverID   types.String `tfsdk:"approver_id"`
}

type MaxPermittedAccessDurationModel struct {
	Value    types.Int64  `tfsdk:"value"`
	TimeUnit types.String `tfsdk:"time_unit"`
}

type AccessProfileRevocationRequestConfigModel struct {
	ApprovalSchemes types.List `tfsdk:"approval_schemes"`
}

type AdditionalOwnerModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type ProvisioningCriteriaModel struct {
	Operation types.String `tfsdk:"operation"`
	Attribute types.String `tfsdk:"attribute"`
	Value     types.String `tfsdk:"value"`
	Children  types.List   `tfsdk:"children"`
}

// ProvisioningCriteriaLeafModel is the third criteria level, which has no children.
type ProvisioningCriteriaLeafModel struct {
	Operation types.String `tfsdk:"operation"`
	Attribute types.String `tfsdk:"attribute"`
	Value     types.String `tfsdk:"value"`
}

func provisioningCriteriaObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation": types.StringType,
		"attribute": types.StringType,
		"value":     types.StringType,
		"children":  types.ListType{ElemType: provisioningCriteriaChildObjectType()},
	}}
}

func provisioningCriteriaChildObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation": types.StringType,
		"attribute": types.StringType,
		"value":     types.StringType,
		"children":  types.ListType{ElemType: provisioningCriteriaLeafObjectType()},
	}}
}

func provisioningCriteriaLeafObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"operation": types.StringType,
		"attribute": types.StringType,
		"value":     types.StringType,
	}}
}

func (r *AccessProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_profile"
}

func (r *AccessProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Access Profile resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"requestable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"segments": schema.ListAttribute{
				MarkdownDescription: "List of segment IDs assigned to the access profile",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
			"source": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
			"entitlements": schema.ListNestedBlock{
				MarkdownDescription: "Entitlements assigned to this access profile",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Entitlement ID",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Entitlement name. Read keeps the configured casing when the API name differs only in case.",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("ENTITLEMENT"),
							MarkdownDescription: "Entitlement type",
						},
					},
				},
			},
			"access_request_config": schema.ListNestedBlock{
				MarkdownDescription: "Access request configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"comments_required": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "If comment is required",
						},
						"denial_comments_required": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "If denial comment is required",
						},
						"reauthorization_required": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Indicates whether reauthorization is required",
						},
						"require_end_date": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Indicates whether the requester must provide access end date",
						},
					},
					Blocks: map[string]schema.Block{
						"approval_schemes": schema.ListNestedBlock{
							MarkdownDescription: "Approval schemes",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"approver_type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Type of approver",
									},
									"approver_id": schema.StringAttribute{
										Optional:            true,
										Computed:            true,
										Default:             stringdefault.StaticString(""),
										MarkdownDescription: "Id of approver",
									},
								},
							},
						},
						"max_permitted_access_duration": schema.ListNestedBlock{
							MarkdownDescription: "Max permitted access duration",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.Int64Attribute{
										Required:            true,
										MarkdownDescription: "The numeric value representing the amount of time",
									},
									"time_unit": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The unit of time",
									},
								},
							},
						},
					},
				},
			},
			"revocation_request_config": schema.ListNestedBlock{
				MarkdownDescription: "Revocation request configuration",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"approval_schemes": schema.ListNestedBlock{
							MarkdownDescription: "Revocation approval schemes",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"approver_type": schema.StringAttribute{Required: true},
									"approver_id":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
								},
							},
						},
					},
				},
			},
			"access_model_metadata": schema.ListNestedBlock{
				MarkdownDescription: "Access model metadata for this access profile",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"attributes": schema.ListNestedBlock{
							MarkdownDescription: "Metadata attributes",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"key":         schema.StringAttribute{Required: true},
									"name":        schema.StringAttribute{Required: true},
									"multiselect": schema.BoolAttribute{Optional: true, Computed: true},
									"status":      schema.StringAttribute{Optional: true, Computed: true},
									"type":        schema.StringAttribute{Optional: true, Computed: true},
									"description": schema.StringAttribute{Optional: true, Computed: true},
								},
								Blocks: map[string]schema.Block{
									"object_types": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{"value": schema.StringAttribute{Required: true}}}},
									"values":       schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{"value": schema.StringAttribute{Required: true}, "name": schema.StringAttribute{Optional: true, Computed: true}, "status": schema.StringAttribute{Optional: true, Computed: true}}}},
								},
							},
						},
					},
				},
			},
			"provisioning_criteria": schema.ListNestedBlock{
				MarkdownDescription: "Provisioning criteria to determine which account gets the access profile",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"operation": schema.StringAttribute{Required: true},
						"attribute": schema.StringAttribute{Optional: true},
						"value":     schema.StringAttribute{Optional: true},
					},
					Blocks: map[string]schema.Block{
						"children": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"operation": schema.StringAttribute{Required: true},
									"attribute": schema.StringAttribute{Optional: true},
									"value":     schema.StringAttribute{Optional: true},
								},
								Blocks: map[string]schema.Block{
									"children": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{"operation": schema.StringAttribute{Required: true}, "attribute": schema.StringAttribute{Optional: true}, "value": schema.StringAttribute{Optional: true}}}},
								},
							},
						},
					},
				},
			},
			"additional_owners": schema.ListNestedBlock{
				MarkdownDescription: "Additional owners for this access profile",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{Required: true},
						"id":   schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}
}

// ModifyPlan warns when access_model_metadata changes on an existing access profile. The access
// profile API only accepts metadata on creation, so such changes are not applied.
func (r *AccessProfileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var planned, prior types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("access_model_metadata"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("access_model_metadata"), &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !equalIgnoringUnknown(planned, prior) {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("access_model_metadata"),
			"Access model metadata is not updated",
			"The access profile API does not support changing access model metadata of an existing access profile. "+
				"The change is not applied in IdentityNow, recreate the access profile to change its metadata.",
		)
	}
}

func (r *AccessProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ap := r.apiFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating Access Profile", map[string]interface{}{"name": ap.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	newAP, err := client.CreateAccessProfile(ctx, ap)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(newAP.ID)
	data.Enabled = computedBoolFromAPI(data.Enabled, newAP.Enabled)
	data.Requestable = computedBoolFromAPI(data.Requestable, newAP.Requestable)
	data.AccessModelMetadata = accessModelMetadataFillUnknown(ctx, data.AccessModelMetadata, newAP.AccessModelMetadata, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// apiFromModel converts the Terraform model to the API AccessProfile struct.
func (r *AccessProfileResource) apiFromModel(ctx context.Context, data AccessProfileResourceModel, diags *diag.Diagnostics) *AccessProfile {
	ap := &AccessProfile{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	var owners []OwnerModel
	diags.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if diags.HasError() {
		return nil
	}
	if len(owners) > 0 {
		ap.AccessProfileOwner = &ObjectInfo{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	var sources []OwnerModel
	diags.Append(data.Source.ElementsAs(ctx, &sources, false)...)
	if diags.HasError() {
		return nil
	}
	if len(sources) > 0 {
		ap.AccessProfileSource = &ObjectInfo{
			ID:   sources[0].ID.ValueString(),
			Type: sources[0].Type.ValueString(),
			Name: sources[0].Name.ValueString(),
		}
	}

	if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() {
		enabled := data.Enabled.ValueBool()
		ap.Enabled = &enabled
	}

	if !data.Requestable.IsNull() && !data.Requestable.IsUnknown() {
		requestable := data.Requestable.ValueBool()
		ap.Requestable = &requestable
	}

	// Entitlements
	if !data.Entitlements.IsNull() && len(data.Entitlements.Elements()) > 0 {
		var entModels []EntitlementRefModel
		diags.Append(data.Entitlements.ElementsAs(ctx, &entModels, false)...)
		if diags.HasError() {
			return nil
		}
		for _, em := range entModels {
			ent := &ObjectInfo{
				ID:   em.ID.ValueString(),
				Name: em.Name.ValueString(),
			}
			if !em.Type.IsNull() && em.Type.ValueString() != "" {
				ent.Type = em.Type.ValueString()
			} else {
				ent.Type = "ENTITLEMENT"
			}
			ap.Entitlements = append(ap.Entitlements, ent)
		}
	}

	// Access Request Config
	if !data.AccessRequestConfig.IsNull() && len(data.AccessRequestConfig.Elements()) > 0 {
		var arcModels []AccessRequestConfigModel
		diags.Append(data.AccessRequestConfig.ElementsAs(ctx, &arcModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(arcModels) > 0 {
			arc := arcModels[0]
			config := &AccessRequestConfigList{}
			if !arc.CommentsRequired.IsNull() {
				config.CommentsRequired = arc.CommentsRequired.ValueBool()
			}
			if !arc.DenialCommentsRequired.IsNull() {
				config.DenialCommentsRequired = arc.DenialCommentsRequired.ValueBool()
			}
			if !arc.ReauthorizationRequired.IsNull() {
				config.ReauthorizationRequired = arc.ReauthorizationRequired.ValueBool()
			}
			if !arc.RequireEndDate.IsNull() {
				config.RequireEndDate = arc.RequireEndDate.ValueBool()
			}
			if !arc.ApprovalSchemes.IsNull() && len(arc.ApprovalSchemes.Elements()) > 0 {
				var schemes []ApprovalSchemeModel
				diags.Append(arc.ApprovalSchemes.ElementsAs(ctx, &schemes, false)...)
				if diags.HasError() {
					return nil
				}
				for _, s := range schemes {
					config.ApprovalSchemes = append(config.ApprovalSchemes, &ApprovalSchemes{
						ApproverType: s.ApproverType.ValueString(),
						ApproverId:   s.ApproverID.ValueString(),
					})
				}
			}
			if !arc.MaxPermittedAccessDuration.IsNull() && len(arc.MaxPermittedAccessDuration.Elements()) > 0 {
				var durModels []MaxPermittedAccessDurationModel
				diags.Append(arc.MaxPermittedAccessDuration.ElementsAs(ctx, &durModels, false)...)
				if diags.HasError() {
					return nil
				}
				if len(durModels) > 0 {
					config.MaxPermittedAccessDuration = &MaxPermittedAccessDuration{
						Value:    int(durModels[0].Value.ValueInt64()),
						TimeUnit: durModels[0].TimeUnit.ValueString(),
					}
				}
			}
			ap.AccessRequestConfig = config
		}
	}

	if !data.Segments.IsNull() && !data.Segments.IsUnknown() {
		var segments []string
		diags.Append(data.Segments.ElementsAs(ctx, &segments, false)...)
		if diags.HasError() {
			return nil
		}
		ap.Segments = segments
	}

	if !data.AccessModelMetadata.IsNull() {
		ap.AccessModelMetadata = accessModelMetadataModelToAPI(ctx, data.AccessModelMetadata, diags)
		if diags.HasError() {
			return nil
		}
	}

	if !data.RevocationRequestConfig.IsNull() {
		var revocationModels []AccessProfileRevocationRequestConfigModel
		diags.Append(data.RevocationRequestConfig.ElementsAs(ctx, &revocationModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(revocationModels) > 0 {
			var schemes []*ApprovalSchemes
			if !revocationModels[0].ApprovalSchemes.IsNull() {
				var schemeModels []ApprovalSchemeModel
				diags.Append(revocationModels[0].ApprovalSchemes.ElementsAs(ctx, &schemeModels, false)...)
				if diags.HasError() {
					return nil
				}
				for _, s := range schemeModels {
					schemes = append(schemes, &ApprovalSchemes{ApproverType: s.ApproverType.ValueString(), ApproverId: s.ApproverID.ValueString()})
				}
			}
			ap.RevocationRequestConfig = &AccessProfileRevocationRequestConfig{ApprovalSchemes: schemes}
		}
	}

	ap.ProvisioningCriteria = provisioningCriteriaModelToAPI(ctx, data.ProvisioningCriteria, diags)

	if !data.AdditionalOwners.IsNull() {
		var owners []AdditionalOwnerModel
		diags.Append(data.AdditionalOwners.ElementsAs(ctx, &owners, false)...)
		if diags.HasError() {
			return nil
		}
		ap.AdditionalOwners = make([]*AdditionalOwnerRef, 0, len(owners))
		for _, owner := range owners {
			ap.AdditionalOwners = append(ap.AdditionalOwners, &AdditionalOwnerRef{
				Type: owner.Type.ValueString(),
				ID:   owner.ID.ValueString(),
				Name: owner.Name.ValueString(),
			})
		}
	}

	return ap
}

// provisioningCriteriaModelToAPI converts up to three levels of provisioning criteria.
func provisioningCriteriaModelToAPI(ctx context.Context, list types.List, diags *diag.Diagnostics) *ProvisioningCriteriaLevel1 {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var criteria []ProvisioningCriteriaModel
	diags.Append(list.ElementsAs(ctx, &criteria, false)...)
	if len(criteria) == 0 {
		return nil
	}
	level1 := &ProvisioningCriteriaLevel1{
		Operation: criteria[0].Operation.ValueString(),
		Attribute: criteria[0].Attribute.ValueString(),
		Value:     criteria[0].Value.ValueString(),
	}
	if criteria[0].Children.IsNull() || criteria[0].Children.IsUnknown() {
		return level1
	}
	var children []ProvisioningCriteriaModel
	diags.Append(criteria[0].Children.ElementsAs(ctx, &children, false)...)
	for _, child := range children {
		level2 := &ProvisioningCriteriaLevel2{
			Operation: child.Operation.ValueString(),
			Attribute: child.Attribute.ValueString(),
			Value:     child.Value.ValueString(),
		}
		if !child.Children.IsNull() && !child.Children.IsUnknown() {
			var leaves []ProvisioningCriteriaLeafModel
			diags.Append(child.Children.ElementsAs(ctx, &leaves, false)...)
			for _, leaf := range leaves {
				level2.Children = append(level2.Children, &ProvisioningCriteriaLevel3{
					Operation: leaf.Operation.ValueString(),
					Attribute: leaf.Attribute.ValueString(),
					Value:     leaf.Value.ValueString(),
				})
			}
		}
		level1.Children = append(level1.Children, level2)
	}
	return level1
}

func (r *AccessProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	ap, err := client.GetAccessProfile(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	r.setStateFromAPI(ctx, &data, ap, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	var state AccessProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	ap := r.apiFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updated *AccessProfile
	if updatePatches := accessProfilePatches(data, state, ap); len(updatePatches) > 0 {
		updated, err = client.UpdateAccessProfile(ctx, updatePatches, data.ID.ValueString())
	} else {
		updated, err = client.GetAccessProfile(ctx, data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update access profile: %s", err))
		return
	}
	data.Enabled = computedBoolFromAPI(data.Enabled, updated.Enabled)
	data.Requestable = computedBoolFromAPI(data.Requestable, updated.Requestable)
	data.AccessModelMetadata = accessModelMetadataFillUnknown(ctx, data.AccessModelMetadata, updated.AccessModelMetadata, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// accessProfilePatches builds replace operations for the fields that differ between plan and state.
// access_model_metadata is not patchable through the access profile API, see ModifyPlan.
func accessProfilePatches(plan, state AccessProfileResourceModel, ap *AccessProfile) []*UpdateAccessProfile {
	var patches []*UpdateAccessProfile
	add := func(changed bool, path string, value interface{}) {
		if changed {
			patches = append(patches, &UpdateAccessProfile{Op: "replace", Path: path, Value: value})
		}
	}
	entitlements := ap.Entitlements
	if entitlements == nil {
		entitlements = []*ObjectInfo{}
	}
	segments := ap.Segments
	if segments == nil {
		segments = []string{}
	}
	additionalOwners := ap.AdditionalOwners
	if additionalOwners == nil {
		additionalOwners = []*AdditionalOwnerRef{}
	}
	sourceChanged := !plan.Source.Equal(state.Source)

	add(!plan.Name.Equal(state.Name), "/name", ap.Name)
	add(!plan.Description.Equal(state.Description), "/description", ap.Description)
	add(!plan.Owner.Equal(state.Owner), "/owner", ap.AccessProfileOwner)
	// The source can only change together with entitlements of the new source.
	add(sourceChanged, "/source", ap.AccessProfileSource)
	add(sourceChanged || !plan.Entitlements.Equal(state.Entitlements), "/entitlements", entitlements)
	add(ap.Enabled != nil && !plan.Enabled.Equal(state.Enabled), "/enabled", ap.Enabled)
	add(ap.Requestable != nil && !plan.Requestable.Equal(state.Requestable), "/requestable", ap.Requestable)
	add(!equalIgnoringUnknown(plan.AccessRequestConfig, state.AccessRequestConfig), "/accessRequestConfig", ap.AccessRequestConfig)
	add(!plan.RevocationRequestConfig.Equal(state.RevocationRequestConfig), "/revocationRequestConfig", ap.RevocationRequestConfig)
	add(!plan.Segments.Equal(state.Segments), "/segments", segments)
	add(!plan.ProvisioningCriteria.Equal(state.ProvisioningCriteria), "/provisioningCriteria", ap.ProvisioningCriteria)
	add(!plan.AdditionalOwners.Equal(state.AdditionalOwners), "/additionalOwners", additionalOwners)
	return patches
}

// computedBoolFromAPI resolves an optional and computed boolean after apply: a known planned
// value is kept, an unknown one is taken from the API response or defaults to false.
func computedBoolFromAPI(planned types.Bool, api *bool) types.Bool {
	if !planned.IsUnknown() {
		return planned
	}
	if api != nil {
		return types.BoolValue(*api)
	}
	return types.BoolValue(false)
}

func (r *AccessProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	ap, err := client.GetAccessProfile(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Auto-detach from source apps before deletion
	if ap.AccessProfileSource != nil && ap.AccessProfileSource.ID != nil {
		sourceApps, err := client.GetSourceAppsAll(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Failed to query source apps: %s", err.Error()))
			return
		}

		apId := data.ID.ValueString()
		apSourceID := fmt.Sprintf("%v", ap.AccessProfileSource.ID)
		for _, sa := range sourceApps {
			// Source apps can only hold access profiles of their account source.
			if sa.SourceAppSource != nil && sa.SourceAppSource.ID != nil && fmt.Sprintf("%v", sa.SourceAppSource.ID) != apSourceID {
				continue
			}
			attachment, err := client.GetAccessProfileAttachment(ctx, sa.ID)
			if err != nil {
				resp.Diagnostics.AddError("Client Error",
					fmt.Sprintf("Failed to get access profile attachments for source app %s: %s", sa.ID, err.Error()))
				return
			}

			for _, attachedId := range attachment.AccessProfiles {
				if attachedId == apId {
					detach := &AccessProfileAttachment{
						SourceAppId:    sa.ID,
						AccessProfiles: []string{apId},
					}
					if err := client.DeleteAccessProfileAttachment(ctx, detach); err != nil {
						resp.Diagnostics.AddError("Client Error",
							fmt.Sprintf("Failed to detach access profile %s from source app %s: %s", apId, sa.ID, err.Error()))
						return
					}
					tflog.Info(ctx, "Auto-detached access profile from source app before deletion", map[string]interface{}{
						"access_profile_id": apId,
						"source_app_id":     sa.ID,
					})
					break
				}
			}
		}
	}

	err = client.DeleteAccessProfile(ctx, ap)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}

func (r *AccessProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AccessProfileResource) setStateFromAPI(ctx context.Context, data *AccessProfileResourceModel, ap *AccessProfile, diags *diag.Diagnostics) {
	data.Name = types.StringValue(ap.Name)
	data.Description = types.StringValue(ap.Description)
	if ap.Enabled != nil {
		data.Enabled = types.BoolValue(*ap.Enabled)
	}
	if ap.Requestable != nil {
		data.Requestable = types.BoolValue(*ap.Requestable)
	}

	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
		"name": types.StringType,
	}}

	// Owner
	if ap.AccessProfileOwner != nil {
		ownerModels := []OwnerModel{
			{
				ID:   types.StringValue(fmt.Sprintf("%v", ap.AccessProfileOwner.ID)),
				Type: types.StringValue(ap.AccessProfileOwner.Type),
				Name: types.StringValue(ap.AccessProfileOwner.Name),
			},
		}
		ownerList, d := types.ListValueFrom(ctx, objType, ownerModels)
		diags.Append(d...)
		data.Owner = ownerList
	} else {
		data.Owner, _ = types.ListValue(objType, []attr.Value{})
	}

	// Source
	if ap.AccessProfileSource != nil {
		sourceModels := []OwnerModel{
			{
				ID:   types.StringValue(fmt.Sprintf("%v", ap.AccessProfileSource.ID)),
				Type: types.StringValue(ap.AccessProfileSource.Type),
				Name: types.StringValue(ap.AccessProfileSource.Name),
			},
		}
		sourceList, d := types.ListValueFrom(ctx, objType, sourceModels)
		diags.Append(d...)
		data.Source = sourceList
	} else {
		data.Source, _ = types.ListValue(objType, []attr.Value{})
	}

	// Entitlements
	entObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"type": types.StringType,
	}}
	if ap.Entitlements != nil {
		var prior []EntitlementRefModel
		if !data.Entitlements.IsNull() && !data.Entitlements.IsUnknown() {
			diags.Append(data.Entitlements.ElementsAs(ctx, &prior, false)...)
		}
		priorByID := make(map[string]EntitlementRefModel, len(prior))
		for _, p := range prior {
			priorByID[p.ID.ValueString()] = p
		}

		apiModels := make([]EntitlementRefModel, 0, len(ap.Entitlements))
		for _, e := range ap.Entitlements {
			// Keep the configured casing when the API name differs only in case
			name := e.Name
			if existing, ok := priorByID[fmt.Sprintf("%v", e.ID)]; ok && strings.EqualFold(existing.Name.ValueString(), name) {
				name = existing.Name.ValueString()
			}

			entType := e.Type
			if entType == "" {
				entType = "ENTITLEMENT"
			}

			apiModels = append(apiModels, EntitlementRefModel{
				ID:   types.StringValue(fmt.Sprintf("%v", e.ID)),
				Name: types.StringValue(name),
				Type: types.StringValue(entType),
			})
		}
		apiModels = orderByPriorIDs(apiModels, prior, func(m EntitlementRefModel) string { return m.ID.ValueString() })

		entList, d := types.ListValueFrom(ctx, entObjType, apiModels)
		diags.Append(d...)
		data.Entitlements = entList
	} else {
		data.Entitlements, _ = types.ListValue(entObjType, []attr.Value{})
	}

	// Access Request Config
	approvalSchemeObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"approver_type": types.StringType,
		"approver_id":   types.StringType,
	}}
	maxDurationObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"value":     types.Int64Type,
		"time_unit": types.StringType,
	}}
	arcObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"comments_required":             types.BoolType,
		"denial_comments_required":      types.BoolType,
		"reauthorization_required":      types.BoolType,
		"require_end_date":              types.BoolType,
		"approval_schemes":              types.ListType{ElemType: approvalSchemeObjType},
		"max_permitted_access_duration": types.ListType{ElemType: maxDurationObjType},
	}}
	if ap.AccessRequestConfig != nil && !(accessRequestConfigIsDefault(ap.AccessRequestConfig) && len(data.AccessRequestConfig.Elements()) == 0) {
		arc := ap.AccessRequestConfig

		var approvalSchemesList types.List
		if arc.ApprovalSchemes != nil {
			schemeModels := make([]ApprovalSchemeModel, len(arc.ApprovalSchemes))
			for i, s := range arc.ApprovalSchemes {
				schemeModels[i] = ApprovalSchemeModel{
					ApproverType: types.StringValue(s.ApproverType),
					ApproverID:   types.StringValue(s.ApproverId),
				}
			}
			sl, d := types.ListValueFrom(ctx, approvalSchemeObjType, schemeModels)
			diags.Append(d...)
			approvalSchemesList = sl
		} else {
			approvalSchemesList, _ = types.ListValue(approvalSchemeObjType, []attr.Value{})
		}

		var maxDurationList types.List
		if arc.MaxPermittedAccessDuration != nil {
			durModels := []MaxPermittedAccessDurationModel{
				{
					Value:    types.Int64Value(int64(arc.MaxPermittedAccessDuration.Value)),
					TimeUnit: types.StringValue(arc.MaxPermittedAccessDuration.TimeUnit),
				},
			}
			dl, d := types.ListValueFrom(ctx, maxDurationObjType, durModels)
			diags.Append(d...)
			maxDurationList = dl
		} else {
			maxDurationList, _ = types.ListValue(maxDurationObjType, []attr.Value{})
		}

		arcModels := []AccessRequestConfigModel{
			{
				CommentsRequired:           types.BoolValue(arc.CommentsRequired),
				DenialCommentsRequired:     types.BoolValue(arc.DenialCommentsRequired),
				ReauthorizationRequired:    types.BoolValue(arc.ReauthorizationRequired),
				RequireEndDate:             types.BoolValue(arc.RequireEndDate),
				ApprovalSchemes:            approvalSchemesList,
				MaxPermittedAccessDuration: maxDurationList,
			},
		}
		arcList, d := types.ListValueFrom(ctx, arcObjType, arcModels)
		diags.Append(d...)
		data.AccessRequestConfig = arcList
	} else {
		data.AccessRequestConfig, _ = types.ListValue(arcObjType, []attr.Value{})
	}

	if len(ap.Segments) > 0 {
		segmentList, d := types.ListValueFrom(ctx, types.StringType, ap.Segments)
		diags.Append(d...)
		data.Segments = segmentList
	} else if data.Segments.IsNull() || data.Segments.IsUnknown() {
		// segments is optional, keep it null when it is not configured
		data.Segments = types.ListNull(types.StringType)
	} else {
		data.Segments, _ = types.ListValue(types.StringType, []attr.Value{})
	}

	revocationObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"approval_schemes": types.ListType{ElemType: approvalSchemeObjType},
	}}
	if ap.RevocationRequestConfig != nil && len(ap.RevocationRequestConfig.ApprovalSchemes) > 0 {
		revocationModels := []AccessProfileRevocationRequestConfigModel{{
			ApprovalSchemes: types.ListNull(approvalSchemeObjType),
		}}
		var schemeModels []ApprovalSchemeModel
		for _, s := range ap.RevocationRequestConfig.ApprovalSchemes {
			schemeModels = append(schemeModels, ApprovalSchemeModel{
				ApproverType: types.StringValue(s.ApproverType),
				ApproverID:   types.StringValue(s.ApproverId),
			})
		}
		sl, d := types.ListValueFrom(ctx, approvalSchemeObjType, schemeModels)
		diags.Append(d...)
		revocationModels[0].ApprovalSchemes = sl
		list, d := types.ListValueFrom(ctx, revocationObjType, revocationModels)
		diags.Append(d...)
		data.RevocationRequestConfig = list
	} else {
		data.RevocationRequestConfig, _ = types.ListValue(revocationObjType, []attr.Value{})
	}

	data.AccessModelMetadata = accessModelMetadataReconcile(ctx, data.AccessModelMetadata, ap.AccessModelMetadata, diags)

	data.ProvisioningCriteria = provisioningCriteriaAPIToState(ctx, ap.ProvisioningCriteria, diags)

	if ap.AdditionalOwners != nil {
		// name is optional, keep it null for owners configured without a name
		unnamed := map[string]bool{}
		var priorOwners []AdditionalOwnerModel
		if !data.AdditionalOwners.IsNull() && !data.AdditionalOwners.IsUnknown() {
			diags.Append(data.AdditionalOwners.ElementsAs(ctx, &priorOwners, false)...)
		}
		for _, owner := range priorOwners {
			if owner.Name.IsNull() {
				unnamed[owner.ID.ValueString()] = true
			}
		}
		ownerModels := make([]AdditionalOwnerModel, 0, len(ap.AdditionalOwners))
		for _, owner := range ap.AdditionalOwners {
			name := stringValueOrNull(owner.Name)
			if unnamed[owner.ID] {
				name = types.StringNull()
			}
			ownerModels = append(ownerModels, AdditionalOwnerModel{
				Type: types.StringValue(owner.Type),
				ID:   types.StringValue(owner.ID),
				Name: name,
			})
		}
		additionalOwnersObjType := types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "id": types.StringType, "name": types.StringType}}
		list, d := types.ListValueFrom(ctx, additionalOwnersObjType, ownerModels)
		diags.Append(d...)
		data.AdditionalOwners = list
	} else {
		data.AdditionalOwners, _ = types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "id": types.StringType, "name": types.StringType}}, []attr.Value{})
	}
}

// accessRequestConfigIsDefault reports whether the API returned only default access request settings.
func accessRequestConfigIsDefault(arc *AccessRequestConfigList) bool {
	return !arc.CommentsRequired && !arc.DenialCommentsRequired && !arc.ReauthorizationRequired && !arc.RequireEndDate &&
		len(arc.ApprovalSchemes) == 0 && arc.MaxPermittedAccessDuration == nil
}

// provisioningCriteriaAPIToState maps up to three levels of provisioning criteria to state.
func provisioningCriteriaAPIToState(ctx context.Context, criteria *ProvisioningCriteriaLevel1, diags *diag.Diagnostics) types.List {
	if criteria == nil {
		list, _ := types.ListValue(provisioningCriteriaObjectType(), []attr.Value{})
		return list
	}
	children := make([]ProvisioningCriteriaModel, 0, len(criteria.Children))
	for _, child := range criteria.Children {
		leaves := make([]ProvisioningCriteriaLeafModel, 0, len(child.Children))
		for _, leaf := range child.Children {
			leaves = append(leaves, ProvisioningCriteriaLeafModel{
				Operation: types.StringValue(leaf.Operation),
				Attribute: stringValueOrNull(leaf.Attribute),
				Value:     stringValueOrNull(leaf.Value),
			})
		}
		leafList, d := types.ListValueFrom(ctx, provisioningCriteriaLeafObjectType(), leaves)
		diags.Append(d...)
		children = append(children, ProvisioningCriteriaModel{
			Operation: types.StringValue(child.Operation),
			Attribute: stringValueOrNull(child.Attribute),
			Value:     stringValueOrNull(child.Value),
			Children:  leafList,
		})
	}
	childList, d := types.ListValueFrom(ctx, provisioningCriteriaChildObjectType(), children)
	diags.Append(d...)
	list, d := types.ListValueFrom(ctx, provisioningCriteriaObjectType(), []ProvisioningCriteriaModel{{
		Operation: types.StringValue(criteria.Operation),
		Attribute: stringValueOrNull(criteria.Attribute),
		Value:     stringValueOrNull(criteria.Value),
		Children:  childList,
	}})
	diags.Append(d...)
	return list
}
