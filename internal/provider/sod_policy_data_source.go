package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SodPolicyDataSource{}
var _ datasource.DataSourceWithValidateConfig = &SodPolicyDataSource{}

func NewSodPolicyDataSource() datasource.DataSource {
	return &SodPolicyDataSource{}
}

type SodPolicyDataSource struct {
	client *Config
}

func (d *SodPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sod_policy"
}

func sodPolicyRefDataSourceAttribute(description string) dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				"id":   dsschema.StringAttribute{MarkdownDescription: "Owner ID", Computed: true},
				"type": dsschema.StringAttribute{MarkdownDescription: "Owner type", Computed: true},
				"name": dsschema.StringAttribute{MarkdownDescription: "Owner name", Computed: true},
			},
		},
	}
}

func (d *SodPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a separation of duties (SOD) policy by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "SOD policy ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "SOD policy name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"description":               dsschema.StringAttribute{MarkdownDescription: "Policy description", Computed: true},
			"owner_ref":                 sodPolicyRefDataSourceAttribute("Owner of the policy"),
			"external_policy_reference": dsschema.StringAttribute{MarkdownDescription: "Reference to an external policy", Computed: true},
			"policy_query":              dsschema.StringAttribute{MarkdownDescription: "Search query of the policy", Computed: true},
			"compensating_controls":     dsschema.StringAttribute{MarkdownDescription: "Compensating (mitigating) controls", Computed: true},
			"correction_advice":         dsschema.StringAttribute{MarkdownDescription: "Advice on how to correct a violation", Computed: true},
			"state":                     dsschema.StringAttribute{MarkdownDescription: "Whether the policy is `ENFORCED` or `NOT_ENFORCED`", Computed: true},
			"tags":                      dsschema.ListAttribute{MarkdownDescription: "Tags of the policy", Computed: true, ElementType: types.StringType},
			"violation_owner_assignment_config": dsschema.ListNestedAttribute{
				MarkdownDescription: "Who owns the violations of the policy",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"assignment_rule": dsschema.StringAttribute{MarkdownDescription: "`MANAGER` or `STATIC`", Computed: true},
						"owner_ref":       sodPolicyRefDataSourceAttribute("Owner of the violations for the `STATIC` rule"),
					},
				},
			},
			"scheduled":                        dsschema.BoolAttribute{MarkdownDescription: "Whether the policy is scheduled", Computed: true},
			"type":                             dsschema.StringAttribute{MarkdownDescription: "`GENERAL` or `CONFLICTING_ACCESS_BASED`", Computed: true},
			"conflicting_access_criteria_json": dsschema.StringAttribute{MarkdownDescription: "Conflicting access criteria as a JSON object", Computed: true},
			"creator_id":                       dsschema.StringAttribute{MarkdownDescription: "ID of the identity that created the policy", Computed: true},
			"modifier_id":                      dsschema.StringAttribute{MarkdownDescription: "ID of the identity that last modified the policy", Computed: true},
			"created":                          dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified":                         dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *SodPolicyDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *SodPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SodPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SodPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var policy *SodPolicy
	if !data.ID.IsNull() {
		policy, err = client.GetSodPolicy(ctx, data.ID.ValueString())
	} else {
		policy, err = client.GetSodPolicyByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("SOD policy not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SOD policy: %s", err))
		return
	}
	setSodPolicyState(ctx, &data, policy, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
