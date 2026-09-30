package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewPublicIdentitiesConfigResource)
}

// publicIdentitiesConfigID is the fixed ID of the tenant-wide public identities configuration.
const publicIdentitiesConfigID = "public-identities-config"

// PublicIdentitiesConfig is the configuration returned by the /v2026/public-identities-config API.
type PublicIdentitiesConfig struct {
	Attributes []*PublicIdentityAttribute `json:"attributes"`
	Modified   string                     `json:"modified,omitempty"`
}

type PublicIdentityAttribute struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (c *Client) GetPublicIdentitiesConfig(ctx context.Context) (*PublicIdentitiesConfig, error) {
	var config PublicIdentitiesConfig
	if err := c.doJSON(ctx, http.MethodGet, "/v2026/public-identities-config", nil, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Client) UpdatePublicIdentitiesConfig(ctx context.Context, config *PublicIdentitiesConfig) (*PublicIdentitiesConfig, error) {
	var updated PublicIdentitiesConfig
	if err := c.doJSON(ctx, http.MethodPut, "/v2026/public-identities-config", config, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

var _ resource.Resource = &PublicIdentitiesConfigResource{}
var _ resource.ResourceWithImportState = &PublicIdentitiesConfigResource{}

func NewPublicIdentitiesConfigResource() resource.Resource {
	return &PublicIdentitiesConfigResource{}
}

type PublicIdentitiesConfigResource struct {
	client *Config
}

type PublicIdentitiesConfigModel struct {
	ID         types.String `tfsdk:"id"`
	Attributes types.List   `tfsdk:"attribute"`
	Modified   types.String `tfsdk:"modified"`
}

type PublicIdentityAttributeModel struct {
	Key  types.String `tfsdk:"key"`
	Name types.String `tfsdk:"name"`
}

var publicIdentityAttributeObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"key":  types.StringType,
	"name": types.StringType,
}}

func (r *PublicIdentitiesConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_identities_config"
}

func (r *PublicIdentitiesConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the tenant-wide identity attributes that are publicly visible to access request approvers and certification reviewers. " +
			"There is one configuration per tenant, destroying the resource only removes it from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Always `" + publicIdentitiesConfigID + "`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"attribute": schema.ListNestedBlock{
				MarkdownDescription: "Identity attribute that is publicly visible. Up to 5 attributes can be configured.",
				Validators:          []validator.List{listSizeBetween(0, 5)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "Identity attribute key",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Identity attribute display name",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *PublicIdentitiesConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func publicIdentitiesConfigFromModel(ctx context.Context, data PublicIdentitiesConfigModel, diags *diag.Diagnostics) *PublicIdentitiesConfig {
	config := &PublicIdentitiesConfig{Attributes: []*PublicIdentityAttribute{}}
	var attributes []PublicIdentityAttributeModel
	diags.Append(data.Attributes.ElementsAs(ctx, &attributes, false)...)
	for _, a := range attributes {
		config.Attributes = append(config.Attributes, &PublicIdentityAttribute{Key: a.Key.ValueString(), Name: a.Name.ValueString()})
	}
	return config
}

func setPublicIdentitiesConfigState(ctx context.Context, data *PublicIdentitiesConfigModel, config *PublicIdentitiesConfig, diags *diag.Diagnostics) {
	data.ID = types.StringValue(publicIdentitiesConfigID)
	data.Modified = types.StringValue(config.Modified)
	attributes := make([]PublicIdentityAttributeModel, 0, len(config.Attributes))
	for _, a := range config.Attributes {
		attributes = append(attributes, PublicIdentityAttributeModel{Key: types.StringValue(a.Key), Name: types.StringValue(a.Name)})
	}
	if len(attributes) == 0 {
		data.Attributes = types.ListNull(publicIdentityAttributeObjectType)
		return
	}
	list, d := types.ListValueFrom(ctx, publicIdentityAttributeObjectType, attributes)
	diags.Append(d...)
	data.Attributes = list
}

// apply saves the planned configuration. The configuration always exists, so Create and Update both replace it.
func (r *PublicIdentitiesConfigResource) apply(ctx context.Context, data *PublicIdentitiesConfigModel, diags *diag.Diagnostics) {
	config := publicIdentitiesConfigFromModel(ctx, *data, diags)
	if diags.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdatePublicIdentitiesConfig(ctx, config)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update public identities config: %s", err))
		return
	}
	data.ID = types.StringValue(publicIdentitiesConfigID)
	data.Modified = types.StringValue(updated.Modified)
}

func (r *PublicIdentitiesConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PublicIdentitiesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIdentitiesConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PublicIdentitiesConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	config, err := client.GetPublicIdentitiesConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read public identities config: %s", err))
		return
	}
	setPublicIdentitiesConfigState(ctx, &data, config, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIdentitiesConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PublicIdentitiesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PublicIdentitiesConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, "Public identities config")
}

func (r *PublicIdentitiesConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), publicIdentitiesConfigID)...)
}
