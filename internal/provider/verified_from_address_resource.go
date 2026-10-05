package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewVerifiedFromAddressResource)
	registerDataSource(NewVerifiedFromAddressDataSource)
}

// VerifiedFromAddress is a sender email address as returned by the /v2026/verified-from-addresses API.
type VerifiedFromAddress struct {
	ID                 string `json:"id,omitempty"`
	Email              string `json:"email"`
	IsVerifiedByDomain bool   `json:"isVerifiedByDomain,omitempty"`
	VerificationStatus string `json:"verificationStatus,omitempty"`
	Region             string `json:"region,omitempty"`
}

// ListVerifiedFromAddresses lists sender addresses, filtered by email when email is not empty.
func (c *Client) ListVerifiedFromAddresses(ctx context.Context, email string) ([]VerifiedFromAddress, error) {
	query := url.Values{}
	if email != "" {
		query.Set("filters", eqFilter("email", email))
	}
	return listAllPages[VerifiedFromAddress](ctx, c, "/v2026/verified-from-addresses", query)
}

// verifiedFromAddressFind lists the sender addresses filtered by email and returns the first one
// that matches. The email filter may be case-sensitive while the state keeps the configured case,
// so when the filtered list has no match, all addresses are listed and matched again.
func (c *Client) verifiedFromAddressFind(ctx context.Context, email string, match func(*VerifiedFromAddress) bool) (*VerifiedFromAddress, error) {
	items, err := c.ListVerifiedFromAddresses(ctx, email)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if match(&items[i]) {
			return &items[i], nil
		}
	}
	if email == "" {
		return nil, nil
	}
	if items, err = c.ListVerifiedFromAddresses(ctx, ""); err != nil {
		return nil, err
	}
	for i := range items {
		if match(&items[i]) {
			return &items[i], nil
		}
	}
	return nil, nil
}

// GetVerifiedFromAddress finds a sender address by ID. The API has no GET by ID, so the list is
// filtered by email when known, and matched by ID; when the filtered list does not contain the ID,
// all addresses are listed.
func (c *Client) GetVerifiedFromAddress(ctx context.Context, id, email string) (*VerifiedFromAddress, error) {
	address, err := c.verifiedFromAddressFind(ctx, email, func(a *VerifiedFromAddress) bool { return id != "" && a.ID == id })
	if err != nil {
		return nil, err
	}
	if address == nil {
		return nil, &NotFoundError{fmt.Sprintf("verified from address %q not found", id)}
	}
	return address, nil
}

// GetVerifiedFromAddressByEmail returns the sender address with the given email, compared case-insensitively.
func (c *Client) GetVerifiedFromAddressByEmail(ctx context.Context, email string) (*VerifiedFromAddress, error) {
	address, err := c.verifiedFromAddressFind(ctx, email, func(a *VerifiedFromAddress) bool { return strings.EqualFold(a.Email, email) })
	if err != nil {
		return nil, err
	}
	if address == nil {
		return nil, &NotFoundError{fmt.Sprintf("verified from address %q not found", email)}
	}
	return address, nil
}

func (c *Client) CreateVerifiedFromAddress(ctx context.Context, address *VerifiedFromAddress) (*VerifiedFromAddress, error) {
	var created VerifiedFromAddress
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/verified-from-addresses", address, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteVerifiedFromAddress(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/verified-from-addresses/%s", id), nil, nil)
}

var _ resource.Resource = &VerifiedFromAddressResource{}
var _ resource.ResourceWithImportState = &VerifiedFromAddressResource{}

func NewVerifiedFromAddressResource() resource.Resource {
	return &VerifiedFromAddressResource{}
}

type VerifiedFromAddressResource struct {
	client *Config
}

type VerifiedFromAddressModel struct {
	ID                 types.String `tfsdk:"id"`
	Email              types.String `tfsdk:"email"`
	IsVerifiedByDomain types.Bool   `tfsdk:"is_verified_by_domain"`
	VerificationStatus types.String `tfsdk:"verification_status"`
	Region             types.String `tfsdk:"region"`
}

func (r *VerifiedFromAddressResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_verified_from_address"
}

func (r *VerifiedFromAddressResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a sender (\"From:\") email address for notifications. Creating it starts the email verification.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Sender address ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Sender email address. The address cannot be changed, changing this forces a new address to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_verified_by_domain": schema.BoolAttribute{
				MarkdownDescription: "Whether the address is verified by its domain.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"verification_status": schema.StringAttribute{
				MarkdownDescription: "Verification status: `PENDING`, `SUCCESS`, `FAILED` or `NA`. Refreshed on every read.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS SES region the address is associated with.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *VerifiedFromAddressResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setVerifiedFromAddressState maps the API address onto the model. A configured email that differs
// only in case is kept.
func setVerifiedFromAddressState(data *VerifiedFromAddressModel, address *VerifiedFromAddress) {
	data.ID = types.StringValue(address.ID)
	if data.Email.IsNull() || data.Email.IsUnknown() || !strings.EqualFold(data.Email.ValueString(), address.Email) {
		data.Email = types.StringValue(address.Email)
	}
	data.IsVerifiedByDomain = types.BoolValue(address.IsVerifiedByDomain)
	data.VerificationStatus = types.StringValue(address.VerificationStatus)
	data.Region = types.StringValue(address.Region)
}

func (r *VerifiedFromAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VerifiedFromAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateVerifiedFromAddress(ctx, &VerifiedFromAddress{Email: data.Email.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create verified from address: %s", err))
		return
	}
	if created.ID == "" {
		// The ID is documented as nullable; look the new address up to get it.
		if created, err = client.GetVerifiedFromAddressByEmail(ctx, data.Email.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read created verified from address: %s", err))
			return
		}
	}
	setVerifiedFromAddressState(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VerifiedFromAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VerifiedFromAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// After import only the ID is known, then all addresses are listed.
	address, err := client.GetVerifiedFromAddress(ctx, data.ID.ValueString(), data.Email.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read verified from address: %s", err))
		return
	}
	setVerifiedFromAddressState(&data, address)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is never called with changes, since the only configurable attribute forces replacement.
func (r *VerifiedFromAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VerifiedFromAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VerifiedFromAddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VerifiedFromAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteVerifiedFromAddress(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete verified from address: %s", err))
	}
}

func (r *VerifiedFromAddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &VerifiedFromAddressDataSource{}

func NewVerifiedFromAddressDataSource() datasource.DataSource {
	return &VerifiedFromAddressDataSource{}
}

type VerifiedFromAddressDataSource struct {
	client *Config
}

func (d *VerifiedFromAddressDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_verified_from_address"
}

func (d *VerifiedFromAddressDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a sender (\"From:\") email address and its verification status by email.",
		Attributes: map[string]dsschema.Attribute{
			"email": dsschema.StringAttribute{
				MarkdownDescription: "Sender email address, compared case-insensitively.",
				Required:            true,
			},
			"id":                    dsschema.StringAttribute{MarkdownDescription: "Sender address ID.", Computed: true},
			"is_verified_by_domain": dsschema.BoolAttribute{MarkdownDescription: "Whether the address is verified by its domain.", Computed: true},
			"verification_status":   dsschema.StringAttribute{MarkdownDescription: "Verification status: `PENDING`, `SUCCESS`, `FAILED` or `NA`.", Computed: true},
			"region":                dsschema.StringAttribute{MarkdownDescription: "AWS SES region the address is associated with.", Computed: true},
		},
	}
}

func (d *VerifiedFromAddressDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VerifiedFromAddressDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VerifiedFromAddressModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	address, err := client.GetVerifiedFromAddressByEmail(ctx, data.Email.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("email"), "Verified from address not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read verified from address: %s", err))
		return
	}
	setVerifiedFromAddressState(&data, address)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
