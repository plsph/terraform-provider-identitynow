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

var _ datasource.DataSource = &AccessProfileDataSource{}

func NewAccessProfileDataSource() datasource.DataSource {
	return &AccessProfileDataSource{}
}

type AccessProfileDataSource struct {
	client *Config
}

type AccessProfileDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Requestable             types.Bool   `tfsdk:"requestable"`
	Source                  types.List   `tfsdk:"source"`
	Owner                   types.List   `tfsdk:"owner"`
	AccessRequestConfig     types.List   `tfsdk:"access_request_config"`
	RevocationRequestConfig types.List   `tfsdk:"revocation_request_config"`
	Segments                types.List   `tfsdk:"segments"`
	AccessModelMetadata     types.List   `tfsdk:"access_model_metadata"`
	ProvisioningCriteria    types.List   `tfsdk:"provisioning_criteria"`
	AdditionalOwners        types.List   `tfsdk:"additional_owners"`
}

func (d *AccessProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_profile"
}

func (d *AccessProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	refAttributes := func(idDesc, typeDesc, nameDesc string) map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, MarkdownDescription: idDesc},
			"type": schema.StringAttribute{Computed: true, MarkdownDescription: typeDesc},
			"name": schema.StringAttribute{Computed: true, MarkdownDescription: nameDesc},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Access Profile data source - looks up an access profile by name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access profile ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the access profile.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access profile description.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the access profile is enabled.",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the access profile is requestable by access request.",
			},
			"source": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the source associated with the access profile.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: refAttributes("Source ID.", "Source type, `SOURCE`.", "Source name."),
				},
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the owner of the access profile.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: refAttributes("Owner identity ID.", "Owner type, `IDENTITY`.", "Owner name."),
				},
			},
			"access_request_config": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the access request configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"comments_required":        schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the requester must provide comments justifying the request."},
						"denial_comments_required": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether an approver must provide comments when denying the request."},
						"reauthorization_required": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether reauthorization is required."},
						"require_end_date":         schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the requester must provide an access end date."},
						"form_definition_id":       schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the form definition presented to the requester during the access request."},
					},
				},
			},
			"revocation_request_config": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the revocation request configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"approval_schemes": schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "List of the approver types of the revocation approval steps, e.g. `[\"MANAGER\"]`."},
					},
				},
			},
			"segments": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of segment IDs assigned to the access profile.",
				ElementType:         types.StringType,
			},
			"access_model_metadata": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access model metadata of the access profile.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attributes": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Metadata attributes.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"key":         schema.StringAttribute{Computed: true, MarkdownDescription: "Unique identifier of the metadata type, e.g. `iscPrivacy`."},
									"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Human readable name of the metadata attribute."},
									"multiselect": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether multiple values can be selected."},
									"status":      schema.StringAttribute{Computed: true, MarkdownDescription: "Status of the metadata attribute, e.g. `active`."},
									"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the metadata attribute, e.g. `governance` or `custom`."},
									"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Description of the metadata attribute."},
									"object_types": schema.ListNestedAttribute{
										Computed:            true,
										MarkdownDescription: "Object types the metadata attribute applies to.",
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"value": schema.StringAttribute{Computed: true, MarkdownDescription: "Object type, e.g. `entitlement`."},
											},
										},
									},
									"values": schema.ListNestedAttribute{
										Computed:            true,
										MarkdownDescription: "Values assigned to the metadata attribute.",
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"value":  schema.StringAttribute{Computed: true, MarkdownDescription: "The metadata value."},
												"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "Human readable name of the value."},
												"status": schema.StringAttribute{Computed: true, MarkdownDescription: "Status of the value, e.g. `active`."},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"provisioning_criteria": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Criteria used to choose the account to provision.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"operation": schema.StringAttribute{Computed: true, MarkdownDescription: "Operation, e.g. `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `HAS`, `AND` or `OR`."},
						"attribute": schema.StringAttribute{Computed: true, MarkdownDescription: "Account attribute to compare, for comparison operations."},
						"value":     schema.StringAttribute{Computed: true, MarkdownDescription: "Value to compare the attribute with, for comparison operations."},
						"children": schema.ListNestedAttribute{
							Computed:            true,
							MarkdownDescription: "Child criteria for `AND` and `OR` operations, with the same attributes (up to 3 levels).",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"operation": schema.StringAttribute{Computed: true, MarkdownDescription: "Operation, e.g. `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `HAS`, `AND` or `OR`."},
									"attribute": schema.StringAttribute{Computed: true, MarkdownDescription: "Account attribute to compare, for comparison operations."},
									"value":     schema.StringAttribute{Computed: true, MarkdownDescription: "Value to compare the attribute with, for comparison operations."},
									"children": schema.ListNestedAttribute{
										Computed:            true,
										MarkdownDescription: "Child criteria for `AND` and `OR` operations, the third and last level of nesting.",
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"operation": schema.StringAttribute{Computed: true, MarkdownDescription: "Operation, e.g. `EQUALS`, `NOT_EQUALS`, `CONTAINS` or `HAS`."},
												"attribute": schema.StringAttribute{Computed: true, MarkdownDescription: "Account attribute to compare."},
												"value":     schema.StringAttribute{Computed: true, MarkdownDescription: "Value to compare the attribute with."},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"additional_owners": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Additional identity or governance group owners.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: refAttributes("ID of the identity or governance group.", "Owner type, `IDENTITY` or `GOVERNANCE_GROUP`.", "Owner name."),
				},
			},
		},
	}
}

func (d *AccessProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AccessProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccessProfileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Access Profile data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	accessProfiles, err := client.GetAccessProfileByName(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Access Profile with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(accessProfiles) == 0 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Access Profile with name %s not found", data.Name.ValueString()))
		return
	}

	ap := accessProfiles[0]
	data.ID = types.StringValue(ap.ID)
	data.Name = types.StringValue(ap.Name)
	data.Description = types.StringValue(ap.Description)

	if ap.Enabled != nil {
		data.Enabled = types.BoolValue(*ap.Enabled)
	} else {
		data.Enabled = types.BoolNull()
	}

	if ap.Requestable != nil {
		data.Requestable = types.BoolValue(*ap.Requestable)
	} else {
		data.Requestable = types.BoolNull()
	}
	if len(ap.Segments) > 0 {
		segmentList, d := types.ListValueFrom(ctx, types.StringType, ap.Segments)
		resp.Diagnostics.Append(d...)
		data.Segments = segmentList
	} else {
		data.Segments = types.ListNull(types.StringType)
	}
	data.Owner = objectInfoListState(ctx, ap.AccessProfileOwner, &resp.Diagnostics)
	data.Source = objectInfoListState(ctx, ap.AccessProfileSource, &resp.Diagnostics)
	data.AccessModelMetadata = accessModelMetadataAPIToState(ctx, ap.AccessModelMetadata, &resp.Diagnostics)
	if ap.ProvisioningCriteria != nil {
		data.ProvisioningCriteria = provisioningCriteriaAPIToState(ctx, ap.ProvisioningCriteria, &resp.Diagnostics)
	} else {
		data.ProvisioningCriteria = types.ListNull(provisioningCriteriaObjectType())
	}

	additionalOwnerType := types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "id": types.StringType, "name": types.StringType}}
	additionalOwners := make([]AdditionalOwnerModel, 0, len(ap.AdditionalOwners))
	for _, owner := range ap.AdditionalOwners {
		additionalOwners = append(additionalOwners, AdditionalOwnerModel{
			Type: types.StringValue(owner.Type),
			ID:   types.StringValue(owner.ID),
			Name: types.StringValue(owner.Name),
		})
	}
	data.AdditionalOwners, _ = types.ListValueFrom(ctx, additionalOwnerType, additionalOwners)

	revocationType := types.ObjectType{AttrTypes: map[string]attr.Type{"approval_schemes": types.ListType{ElemType: types.StringType}}}
	data.RevocationRequestConfig = types.ListNull(revocationType)
	if ap.RevocationRequestConfig != nil {
		// approval_schemes lists the approver types, e.g. MANAGER or GOVERNANCE_GROUP
		approvers := make([]string, 0, len(ap.RevocationRequestConfig.ApprovalSchemes))
		for _, scheme := range ap.RevocationRequestConfig.ApprovalSchemes {
			approvers = append(approvers, scheme.ApproverType)
		}
		approverList, d := types.ListValueFrom(ctx, types.StringType, approvers)
		resp.Diagnostics.Append(d...)
		revocation, d := types.ObjectValue(revocationType.AttrTypes, map[string]attr.Value{"approval_schemes": approverList})
		resp.Diagnostics.Append(d...)
		data.RevocationRequestConfig, d = types.ListValue(revocationType, []attr.Value{revocation})
		resp.Diagnostics.Append(d...)
	}

	requestConfigType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"comments_required":        types.BoolType,
		"denial_comments_required": types.BoolType,
		"reauthorization_required": types.BoolType,
		"require_end_date":         types.BoolType,
		"form_definition_id":       types.StringType,
	}}
	data.AccessRequestConfig = types.ListNull(requestConfigType)
	if ap.AccessRequestConfig != nil {
		requestConfigValue, d := types.ObjectValue(requestConfigType.AttrTypes, map[string]attr.Value{
			"comments_required":        types.BoolValue(ap.AccessRequestConfig.CommentsRequired),
			"denial_comments_required": types.BoolValue(ap.AccessRequestConfig.DenialCommentsRequired),
			"reauthorization_required": types.BoolValue(ap.AccessRequestConfig.ReauthorizationRequired),
			"require_end_date":         types.BoolValue(ap.AccessRequestConfig.RequireEndDate),
			"form_definition_id":       stringValueOrNull(ap.AccessRequestConfig.FormDefinitionId),
		})
		if d.HasError() {
			resp.Diagnostics.Append(d...)
		} else {
			data.AccessRequestConfig, d = types.ListValue(requestConfigType, []attr.Value{requestConfigValue})
			resp.Diagnostics.Append(d...)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
