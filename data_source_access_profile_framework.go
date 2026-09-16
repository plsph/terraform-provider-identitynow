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
	resp.Schema = schema.Schema{
		MarkdownDescription: "Access Profile data source - looks up an access profile by name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access Profile ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Access Profile name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access Profile description",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether enabled",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether requestable",
			},
			"source": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access Profile source",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access Profile owner",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"access_request_config": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access request configuration",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"comments_required":        schema.BoolAttribute{Computed: true},
						"denial_comments_required": schema.BoolAttribute{Computed: true},
						"reauthorization_required": schema.BoolAttribute{Computed: true},
						"require_end_date":         schema.BoolAttribute{Computed: true},
					},
				},
			},
			"revocation_request_config": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Revocation request configuration",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"approval_schemes": schema.ListAttribute{Computed: true, ElementType: types.StringType},
					},
				},
			},
			"segments": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "Segment IDs assigned to this access profile",
				ElementType:         types.StringType,
			},
			"access_model_metadata": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Access model metadata for this access profile",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"attributes": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"key":         schema.StringAttribute{Computed: true},
									"name":        schema.StringAttribute{Computed: true},
									"multiselect": schema.BoolAttribute{Computed: true},
									"status":      schema.StringAttribute{Computed: true},
									"type":        schema.StringAttribute{Computed: true},
									"description": schema.StringAttribute{Computed: true},
									"object_types": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"value": schema.StringAttribute{Computed: true},
											},
										},
									},
									"values": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"value":  schema.StringAttribute{Computed: true},
												"name":   schema.StringAttribute{Computed: true},
												"status": schema.StringAttribute{Computed: true},
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
				MarkdownDescription: "Provisioning criteria for this access profile",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"operation": schema.StringAttribute{Computed: true},
						"attribute": schema.StringAttribute{Computed: true},
						"value":     schema.StringAttribute{Computed: true},
						"children": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"operation": schema.StringAttribute{Computed: true},
									"attribute": schema.StringAttribute{Computed: true},
									"value":     schema.StringAttribute{Computed: true},
									"children": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"operation": schema.StringAttribute{Computed: true},
												"attribute": schema.StringAttribute{Computed: true},
												"value":     schema.StringAttribute{Computed: true},
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
				MarkdownDescription: "Additional owners for this access profile",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{Computed: true},
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
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
		if _, notFound := err.(*NotFoundError); notFound {
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
	if ap.AccessRequestConfig != nil {
		requestConfig := []map[string]interface{}{{
			"comments_required":        ap.AccessRequestConfig.CommentsRequired,
			"denial_comments_required": ap.AccessRequestConfig.DenialCommentsRequired,
			"reauthorization_required": ap.AccessRequestConfig.ReauthorizationRequired,
			"require_end_date":         ap.AccessRequestConfig.RequireEndDate,
		}}
		if value, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
			"comments_required":        types.BoolType,
			"denial_comments_required": types.BoolType,
			"reauthorization_required": types.BoolType,
			"require_end_date":         types.BoolType,
		}}, requestConfig); d.HasError() {
			resp.Diagnostics.Append(d...)
		} else {
			data.AccessRequestConfig = value
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
