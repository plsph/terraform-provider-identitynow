package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &PasswordPolicyResource{}

func NewPasswordPolicyResource() resource.Resource {
	return &PasswordPolicyResource{}
}

type PasswordPolicyResource struct {
	client *Config
}

type PasswordPolicyResourceModel struct {
	ID                                    types.String `tfsdk:"id"`
	Name                                  types.String `tfsdk:"name"`
	Description                           types.String `tfsdk:"description"`
	AccountIDMinWordLength                types.Int64  `tfsdk:"account_id_min_word_length"`
	AccountNameMinWordLength              types.Int64  `tfsdk:"account_name_min_word_length"`
	DefaultPolicy                         types.Bool   `tfsdk:"default_policy"`
	EnablePasswordExpiration              types.Bool   `tfsdk:"enable_password_expiration"`
	FirstExpirationReminder               types.Int64  `tfsdk:"first_expiration_reminder"`
	MaxLength                             types.Int64  `tfsdk:"max_length"`
	MaxRepeatedChars                      types.Int64  `tfsdk:"max_repeated_chars"`
	MinAlpha                              types.Int64  `tfsdk:"min_alpha"`
	MinCharacterTypes                     types.Int64  `tfsdk:"min_character_types"`
	MinLength                             types.Int64  `tfsdk:"min_length"`
	MinLower                              types.Int64  `tfsdk:"min_lower"`
	MinNumeric                            types.Int64  `tfsdk:"min_numeric"`
	MinSpecial                            types.Int64  `tfsdk:"min_special"`
	MinUpper                              types.Int64  `tfsdk:"min_upper"`
	PasswordExpiration                    types.Int64  `tfsdk:"password_expiration"`
	RequireStrongAuthOffNetwork           types.Bool   `tfsdk:"require_strong_auth_off_network"`
	RequireStrongAuthUntrustedGeographies types.Bool   `tfsdk:"require_strong_auth_untrusted_geographies"`
	RequireStrongAuthn                    types.Bool   `tfsdk:"require_strong_authn"`
	UseAccountAttributes                  types.Bool   `tfsdk:"use_account_attributes"`
	UseDictionary                         types.Bool   `tfsdk:"use_dictionary"`
	UseHistory                            types.Int64  `tfsdk:"use_history"`
	UseIdentityAttributes                 types.Bool   `tfsdk:"use_identity_attributes"`
	ValidateAgainstAccountID              types.Bool   `tfsdk:"validate_against_account_id"`
	ValidateAgainstAccountName            types.Bool   `tfsdk:"validate_against_account_name"`
	SourceIDs                             types.List   `tfsdk:"source_ids"`
	ConnectedServices                     types.List   `tfsdk:"connected_services"`
	DateCreated                           types.String `tfsdk:"date_created"`
	LastUpdated                           types.String `tfsdk:"last_updated"`
}

type ConnectedServiceModel struct {
	ID                      types.String `tfsdk:"id"`
	ExternalID              types.String `tfsdk:"external_id"`
	Name                    types.String `tfsdk:"name"`
	SupportsPasswordSetDate types.Bool   `tfsdk:"supports_password_set_date"`
	AppCount                types.Int64  `tfsdk:"app_count"`
}

func (r *PasswordPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_policy"
}

func (r *PasswordPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Password Policy resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Password Policy ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Password policy name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Password policy description",
			},
			"account_id_min_word_length": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Char length that disallow account ID fragments",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"account_name_min_word_length": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Char length that disallow display name fragments",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"default_policy": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Is the password policy default policy?",
			},
			"enable_password_expiration": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable password expiration",
			},
			"first_expiration_reminder": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "First expiration reminder",
			},
			"max_length": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Password max length",
			},
			"max_repeated_chars": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Max repeated characters",
			},
			"min_alpha": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum letters in password",
			},
			"min_character_types": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum character types",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_length": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum password length",
			},
			"min_lower": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum number of lowercase characters",
			},
			"min_numeric": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum number in password",
			},
			"min_special": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum special characters",
			},
			"min_upper": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Minimum uppercase characters",
			},
			"password_expiration": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Password expiration in days",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"require_strong_auth_off_network": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Require strong authentication off network",
			},
			"require_strong_auth_untrusted_geographies": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Require strong authentication for untrusted geographies",
			},
			"require_strong_authn": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Require strong authentication",
			},
			"use_account_attributes": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Prevent use of account attributes?",
			},
			"use_dictionary": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Prevent use of words in this site's password dictionary?",
			},
			"use_history": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Use history",
			},
			"use_identity_attributes": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Prevent use of identity attributes?",
			},
			"validate_against_account_id": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disallow account ID fragments?",
			},
			"validate_against_account_name": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disallow account name fragments?",
			},
			"source_ids": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "List of source IDs",
				ElementType:         types.StringType,
			},
			"connected_services": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Connected services",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Source ID",
						},
						"external_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Source external ID",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Source name",
						},
						"supports_password_set_date": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Supports password set date",
						},
						"app_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "App count",
						},
					},
				},
			},
			"date_created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Date created",
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last updated",
			},
		},
	}
}

func (r *PasswordPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PasswordPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PasswordPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pp := r.buildPasswordPolicy(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating Password Policy", map[string]interface{}{"name": pp.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newPP, err := client.CreatePasswordPolicy(ctx, pp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create password policy: %s", err))
		return
	}

	r.setStateFromAPI(&data, newPP)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PasswordPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Password Policy", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	pp, err := client.GetPasswordPolicy(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read password policy: %s", err))
		return
	}

	r.setStateFromAPI(&data, pp)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PasswordPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Password Policy", map[string]interface{}{"id": data.ID.ValueString()})

	pp := r.buildPasswordPolicy(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	pp.ID = data.ID.ValueString()

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	_, err = client.UpdatePasswordPolicy(ctx, pp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update password policy: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PasswordPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Password Policy", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	pp, err := client.GetPasswordPolicy(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get password policy: %s", err))
		return
	}

	err = client.DeletePasswordPolicy(ctx, pp.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete password policy: %s", err))
		return
	}
}

func (r *PasswordPolicyResource) buildPasswordPolicy(ctx context.Context, data PasswordPolicyResourceModel, diags *diag.Diagnostics) *PasswordPolicy {
	pp := &PasswordPolicy{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	if !data.AccountIDMinWordLength.IsNull() {
		v := int(data.AccountIDMinWordLength.ValueInt64())
		pp.AccountIDMinWordLength = &v
	}
	if !data.AccountNameMinWordLength.IsNull() {
		v := int(data.AccountNameMinWordLength.ValueInt64())
		pp.AccountNameMinWordLength = &v
	}
	if !data.DefaultPolicy.IsNull() {
		v := data.DefaultPolicy.ValueBool()
		pp.DefaultPolicy = &v
	}
	if !data.EnablePasswordExpiration.IsNull() {
		v := data.EnablePasswordExpiration.ValueBool()
		pp.EnablePasswordExpiration = &v
	}
	if !data.FirstExpirationReminder.IsNull() {
		v := int(data.FirstExpirationReminder.ValueInt64())
		pp.FirstExpirationReminder = &v
	}
	if !data.MaxLength.IsNull() {
		v := int(data.MaxLength.ValueInt64())
		pp.MaxLength = &v
	}
	if !data.MaxRepeatedChars.IsNull() {
		v := int(data.MaxRepeatedChars.ValueInt64())
		pp.MaxRepeatedChars = &v
	}
	if !data.MinAlpha.IsNull() {
		v := int(data.MinAlpha.ValueInt64())
		pp.MinAlpha = &v
	}
	if !data.MinCharacterTypes.IsNull() {
		v := int(data.MinCharacterTypes.ValueInt64())
		pp.MinCharacterTypes = &v
	}
	if !data.MinLength.IsNull() {
		v := int(data.MinLength.ValueInt64())
		pp.MinLength = &v
	}
	if !data.MinLower.IsNull() {
		v := int(data.MinLower.ValueInt64())
		pp.MinLower = &v
	}
	if !data.MinNumeric.IsNull() {
		v := int(data.MinNumeric.ValueInt64())
		pp.MinNumeric = &v
	}
	if !data.MinSpecial.IsNull() {
		v := int(data.MinSpecial.ValueInt64())
		pp.MinSpecial = &v
	}
	if !data.MinUpper.IsNull() {
		v := int(data.MinUpper.ValueInt64())
		pp.MinUpper = &v
	}
	if !data.PasswordExpiration.IsNull() {
		v := int(data.PasswordExpiration.ValueInt64())
		pp.PasswordExpiration = &v
	}
	if !data.RequireStrongAuthOffNetwork.IsNull() {
		v := data.RequireStrongAuthOffNetwork.ValueBool()
		pp.RequireStrongAuthOffNetwork = &v
	}
	if !data.RequireStrongAuthUntrustedGeographies.IsNull() {
		v := data.RequireStrongAuthUntrustedGeographies.ValueBool()
		pp.RequireStrongAuthUntrustedGeographies = &v
	}
	if !data.RequireStrongAuthn.IsNull() {
		v := data.RequireStrongAuthn.ValueBool()
		pp.RequireStrongAuthn = &v
	}
	if !data.UseAccountAttributes.IsNull() {
		v := data.UseAccountAttributes.ValueBool()
		pp.UseAccountAttributes = &v
	}
	if !data.UseDictionary.IsNull() {
		v := data.UseDictionary.ValueBool()
		pp.UseDictionary = &v
	}
	if !data.UseHistory.IsNull() {
		v := int(data.UseHistory.ValueInt64())
		pp.UseHistory = &v
	}
	if !data.UseIdentityAttributes.IsNull() {
		v := data.UseIdentityAttributes.ValueBool()
		pp.UseIdentityAttributes = &v
	}
	if !data.ValidateAgainstAccountID.IsNull() {
		v := data.ValidateAgainstAccountID.ValueBool()
		pp.ValidateAgainstAccountID = &v
	}
	if !data.ValidateAgainstAccountName.IsNull() {
		v := data.ValidateAgainstAccountName.ValueBool()
		pp.ValidateAgainstAccountName = &v
	}

	if !data.SourceIDs.IsNull() {
		var sourceIDs []string
		diags.Append(data.SourceIDs.ElementsAs(ctx, &sourceIDs, false)...)
		pp.SourceIDs = sourceIDs
	}

	return pp
}

func (r *PasswordPolicyResource) setStateFromAPI(data *PasswordPolicyResourceModel, pp *PasswordPolicy) {
	data.ID = types.StringValue(pp.ID)
	data.Name = types.StringValue(pp.Name)
	data.Description = types.StringValue(pp.Description)

	if pp.AccountIDMinWordLength != nil {
		data.AccountIDMinWordLength = types.Int64Value(int64(*pp.AccountIDMinWordLength))
	}
	if pp.AccountNameMinWordLength != nil {
		data.AccountNameMinWordLength = types.Int64Value(int64(*pp.AccountNameMinWordLength))
	}
	if pp.DefaultPolicy != nil {
		data.DefaultPolicy = types.BoolValue(*pp.DefaultPolicy)
	}
	if pp.EnablePasswordExpiration != nil {
		data.EnablePasswordExpiration = types.BoolValue(*pp.EnablePasswordExpiration)
	}
	if pp.FirstExpirationReminder != nil {
		data.FirstExpirationReminder = types.Int64Value(int64(*pp.FirstExpirationReminder))
	}
	if pp.MaxLength != nil {
		data.MaxLength = types.Int64Value(int64(*pp.MaxLength))
	}
	if pp.MaxRepeatedChars != nil {
		data.MaxRepeatedChars = types.Int64Value(int64(*pp.MaxRepeatedChars))
	}
	if pp.MinAlpha != nil {
		data.MinAlpha = types.Int64Value(int64(*pp.MinAlpha))
	}
	if pp.MinCharacterTypes != nil {
		data.MinCharacterTypes = types.Int64Value(int64(*pp.MinCharacterTypes))
	}
	if pp.MinLength != nil {
		data.MinLength = types.Int64Value(int64(*pp.MinLength))
	}
	if pp.MinLower != nil {
		data.MinLower = types.Int64Value(int64(*pp.MinLower))
	}
	if pp.MinNumeric != nil {
		data.MinNumeric = types.Int64Value(int64(*pp.MinNumeric))
	}
	if pp.MinSpecial != nil {
		data.MinSpecial = types.Int64Value(int64(*pp.MinSpecial))
	}
	if pp.MinUpper != nil {
		data.MinUpper = types.Int64Value(int64(*pp.MinUpper))
	}
	if pp.PasswordExpiration != nil {
		data.PasswordExpiration = types.Int64Value(int64(*pp.PasswordExpiration))
	}
	if pp.RequireStrongAuthOffNetwork != nil {
		data.RequireStrongAuthOffNetwork = types.BoolValue(*pp.RequireStrongAuthOffNetwork)
	}
	if pp.RequireStrongAuthUntrustedGeographies != nil {
		data.RequireStrongAuthUntrustedGeographies = types.BoolValue(*pp.RequireStrongAuthUntrustedGeographies)
	}
	if pp.RequireStrongAuthn != nil {
		data.RequireStrongAuthn = types.BoolValue(*pp.RequireStrongAuthn)
	}
	if pp.UseAccountAttributes != nil {
		data.UseAccountAttributes = types.BoolValue(*pp.UseAccountAttributes)
	}
	if pp.UseDictionary != nil {
		data.UseDictionary = types.BoolValue(*pp.UseDictionary)
	}
	if pp.UseHistory != nil {
		data.UseHistory = types.Int64Value(int64(*pp.UseHistory))
	}
	if pp.UseIdentityAttributes != nil {
		data.UseIdentityAttributes = types.BoolValue(*pp.UseIdentityAttributes)
	}
	if pp.ValidateAgainstAccountID != nil {
		data.ValidateAgainstAccountID = types.BoolValue(*pp.ValidateAgainstAccountID)
	}
	if pp.ValidateAgainstAccountName != nil {
		data.ValidateAgainstAccountName = types.BoolValue(*pp.ValidateAgainstAccountName)
	}
}
