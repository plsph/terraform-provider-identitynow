package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &PasswordPolicyResource{}
var _ resource.ResourceWithImportState = &PasswordPolicyResource{}

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
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Password policy description",
			},
			"account_id_min_word_length": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Char length that disallow account ID fragments",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"account_name_min_word_length": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Char length that disallow display name fragments",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"default_policy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Is the password policy default policy?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enable_password_expiration": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enable password expiration",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"first_expiration_reminder": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "First expiration reminder",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_length": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Password max length",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_repeated_chars": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Max repeated characters",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_alpha": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum letters in password",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_character_types": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum character types",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_length": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum password length",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_lower": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum number of lowercase characters",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_numeric": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum number in password",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_special": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum special characters",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"min_upper": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum uppercase characters",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"password_expiration": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Password expiration in days",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"require_strong_auth_off_network": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Require strong authentication off network",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"require_strong_auth_untrusted_geographies": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Require strong authentication for untrusted geographies",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"require_strong_authn": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Require strong authentication",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_account_attributes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Prevent use of account attributes?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_dictionary": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Prevent use of words in this site's password dictionary?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_history": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Use history",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"use_identity_attributes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Prevent use of identity attributes?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"validate_against_account_id": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Disallow account ID fragments?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"validate_against_account_name": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Disallow account name fragments?",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	r.setStateFromAPI(ctx, &data, newPP, false, &resp.Diagnostics)
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
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read password policy: %s", err))
		return
	}

	r.setStateFromAPI(ctx, &data, pp, true, &resp.Diagnostics)
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

	updatedPP, err := client.UpdatePasswordPolicy(ctx, pp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update password policy: %s", err))
		return
	}

	r.setStateFromAPI(ctx, &data, updatedPP, false, &resp.Diagnostics)

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
		if isNotFound(err) {
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

	if !data.AccountIDMinWordLength.IsNull() && !data.AccountIDMinWordLength.IsUnknown() {
		v := int(data.AccountIDMinWordLength.ValueInt64())
		pp.AccountIDMinWordLength = &v
	}
	if !data.AccountNameMinWordLength.IsNull() && !data.AccountNameMinWordLength.IsUnknown() {
		v := int(data.AccountNameMinWordLength.ValueInt64())
		pp.AccountNameMinWordLength = &v
	}
	if !data.DefaultPolicy.IsNull() && !data.DefaultPolicy.IsUnknown() {
		v := data.DefaultPolicy.ValueBool()
		pp.DefaultPolicy = &v
	}
	if !data.EnablePasswordExpiration.IsNull() && !data.EnablePasswordExpiration.IsUnknown() {
		v := data.EnablePasswordExpiration.ValueBool()
		pp.EnablePasswordExpiration = &v
	}
	if !data.FirstExpirationReminder.IsNull() && !data.FirstExpirationReminder.IsUnknown() {
		v := int(data.FirstExpirationReminder.ValueInt64())
		pp.FirstExpirationReminder = &v
	}
	if !data.MaxLength.IsNull() && !data.MaxLength.IsUnknown() {
		v := int(data.MaxLength.ValueInt64())
		pp.MaxLength = &v
	}
	if !data.MaxRepeatedChars.IsNull() && !data.MaxRepeatedChars.IsUnknown() {
		v := int(data.MaxRepeatedChars.ValueInt64())
		pp.MaxRepeatedChars = &v
	}
	if !data.MinAlpha.IsNull() && !data.MinAlpha.IsUnknown() {
		v := int(data.MinAlpha.ValueInt64())
		pp.MinAlpha = &v
	}
	if !data.MinCharacterTypes.IsNull() && !data.MinCharacterTypes.IsUnknown() {
		v := int(data.MinCharacterTypes.ValueInt64())
		pp.MinCharacterTypes = &v
	}
	if !data.MinLength.IsNull() && !data.MinLength.IsUnknown() {
		v := int(data.MinLength.ValueInt64())
		pp.MinLength = &v
	}
	if !data.MinLower.IsNull() && !data.MinLower.IsUnknown() {
		v := int(data.MinLower.ValueInt64())
		pp.MinLower = &v
	}
	if !data.MinNumeric.IsNull() && !data.MinNumeric.IsUnknown() {
		v := int(data.MinNumeric.ValueInt64())
		pp.MinNumeric = &v
	}
	if !data.MinSpecial.IsNull() && !data.MinSpecial.IsUnknown() {
		v := int(data.MinSpecial.ValueInt64())
		pp.MinSpecial = &v
	}
	if !data.MinUpper.IsNull() && !data.MinUpper.IsUnknown() {
		v := int(data.MinUpper.ValueInt64())
		pp.MinUpper = &v
	}
	if !data.PasswordExpiration.IsNull() && !data.PasswordExpiration.IsUnknown() {
		v := int(data.PasswordExpiration.ValueInt64())
		pp.PasswordExpiration = &v
	}
	if !data.RequireStrongAuthOffNetwork.IsNull() && !data.RequireStrongAuthOffNetwork.IsUnknown() {
		v := data.RequireStrongAuthOffNetwork.ValueBool()
		pp.RequireStrongAuthOffNetwork = &v
	}
	if !data.RequireStrongAuthUntrustedGeographies.IsNull() && !data.RequireStrongAuthUntrustedGeographies.IsUnknown() {
		v := data.RequireStrongAuthUntrustedGeographies.ValueBool()
		pp.RequireStrongAuthUntrustedGeographies = &v
	}
	if !data.RequireStrongAuthn.IsNull() && !data.RequireStrongAuthn.IsUnknown() {
		v := data.RequireStrongAuthn.ValueBool()
		pp.RequireStrongAuthn = &v
	}
	if !data.UseAccountAttributes.IsNull() && !data.UseAccountAttributes.IsUnknown() {
		v := data.UseAccountAttributes.ValueBool()
		pp.UseAccountAttributes = &v
	}
	if !data.UseDictionary.IsNull() && !data.UseDictionary.IsUnknown() {
		v := data.UseDictionary.ValueBool()
		pp.UseDictionary = &v
	}
	if !data.UseHistory.IsNull() && !data.UseHistory.IsUnknown() {
		v := int(data.UseHistory.ValueInt64())
		pp.UseHistory = &v
	}
	if !data.UseIdentityAttributes.IsNull() && !data.UseIdentityAttributes.IsUnknown() {
		v := data.UseIdentityAttributes.ValueBool()
		pp.UseIdentityAttributes = &v
	}
	if !data.ValidateAgainstAccountID.IsNull() && !data.ValidateAgainstAccountID.IsUnknown() {
		v := data.ValidateAgainstAccountID.ValueBool()
		pp.ValidateAgainstAccountID = &v
	}
	if !data.ValidateAgainstAccountName.IsNull() && !data.ValidateAgainstAccountName.IsUnknown() {
		v := data.ValidateAgainstAccountName.ValueBool()
		pp.ValidateAgainstAccountName = &v
	}

	if !data.SourceIDs.IsNull() && !data.SourceIDs.IsUnknown() {
		var sourceIDs []string
		diags.Append(data.SourceIDs.ElementsAs(ctx, &sourceIDs, false)...)
		pp.SourceIDs = sourceIDs
	}

	return pp
}

// setStateFromAPI maps an API password policy onto the model. With refresh set (Read), API values
// replace the model values. Otherwise (Create, Update) only unknown values are resolved, so the
// state matches the plan.
func (r *PasswordPolicyResource) setStateFromAPI(ctx context.Context, data *PasswordPolicyResourceModel, pp *PasswordPolicy, refresh bool, diags *diag.Diagnostics) {
	data.ID = types.StringValue(pp.ID)
	if refresh {
		data.Name = types.StringValue(pp.Name)
		if pp.Description != "" || !data.Description.IsNull() {
			data.Description = types.StringValue(pp.Description)
		}
		if len(pp.SourceIDs) > 0 || !data.SourceIDs.IsNull() {
			sourceIDs, d := types.ListValueFrom(ctx, types.StringType, pp.SourceIDs)
			diags.Append(d...)
			if pp.SourceIDs == nil {
				sourceIDs, _ = types.ListValue(types.StringType, []attr.Value{})
			}
			data.SourceIDs = sourceIDs
		}
	}

	data.AccountIDMinWordLength = int64FromAPI(data.AccountIDMinWordLength, pp.AccountIDMinWordLength, refresh)
	data.AccountNameMinWordLength = int64FromAPI(data.AccountNameMinWordLength, pp.AccountNameMinWordLength, refresh)
	data.DefaultPolicy = boolFromAPI(data.DefaultPolicy, pp.DefaultPolicy, refresh)
	data.EnablePasswordExpiration = boolFromAPI(data.EnablePasswordExpiration, pp.EnablePasswordExpiration, refresh)
	data.FirstExpirationReminder = int64FromAPI(data.FirstExpirationReminder, pp.FirstExpirationReminder, refresh)
	data.MaxLength = int64FromAPI(data.MaxLength, pp.MaxLength, refresh)
	data.MaxRepeatedChars = int64FromAPI(data.MaxRepeatedChars, pp.MaxRepeatedChars, refresh)
	data.MinAlpha = int64FromAPI(data.MinAlpha, pp.MinAlpha, refresh)
	data.MinCharacterTypes = int64FromAPI(data.MinCharacterTypes, pp.MinCharacterTypes, refresh)
	data.MinLength = int64FromAPI(data.MinLength, pp.MinLength, refresh)
	data.MinLower = int64FromAPI(data.MinLower, pp.MinLower, refresh)
	data.MinNumeric = int64FromAPI(data.MinNumeric, pp.MinNumeric, refresh)
	data.MinSpecial = int64FromAPI(data.MinSpecial, pp.MinSpecial, refresh)
	data.MinUpper = int64FromAPI(data.MinUpper, pp.MinUpper, refresh)
	data.PasswordExpiration = int64FromAPI(data.PasswordExpiration, pp.PasswordExpiration, refresh)
	data.RequireStrongAuthOffNetwork = boolFromAPI(data.RequireStrongAuthOffNetwork, pp.RequireStrongAuthOffNetwork, refresh)
	data.RequireStrongAuthUntrustedGeographies = boolFromAPI(data.RequireStrongAuthUntrustedGeographies, pp.RequireStrongAuthUntrustedGeographies, refresh)
	data.RequireStrongAuthn = boolFromAPI(data.RequireStrongAuthn, pp.RequireStrongAuthn, refresh)
	data.UseAccountAttributes = boolFromAPI(data.UseAccountAttributes, pp.UseAccountAttributes, refresh)
	data.UseDictionary = boolFromAPI(data.UseDictionary, pp.UseDictionary, refresh)
	data.UseHistory = int64FromAPI(data.UseHistory, pp.UseHistory, refresh)
	data.UseIdentityAttributes = boolFromAPI(data.UseIdentityAttributes, pp.UseIdentityAttributes, refresh)
	data.ValidateAgainstAccountID = boolFromAPI(data.ValidateAgainstAccountID, pp.ValidateAgainstAccountID, refresh)
	data.ValidateAgainstAccountName = boolFromAPI(data.ValidateAgainstAccountName, pp.ValidateAgainstAccountName, refresh)

	// Connected Services
	connSvcObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":                         types.StringType,
		"external_id":                types.StringType,
		"name":                       types.StringType,
		"supports_password_set_date": types.BoolType,
		"app_count":                  types.Int64Type,
	}}
	csModels := make([]ConnectedServiceModel, len(pp.ConnectedServices))
	for i, cs := range pp.ConnectedServices {
		csModels[i] = ConnectedServiceModel{
			ID:                      types.StringValue(cs.ID),
			ExternalID:              types.StringValue(cs.ExternalID),
			Name:                    types.StringValue(cs.Name),
			SupportsPasswordSetDate: types.BoolValue(cs.SupportsPasswordSetDate),
			AppCount:                types.Int64Value(int64(cs.AppCount)),
		}
	}
	csList, d := types.ListValueFrom(ctx, connSvcObjType, csModels)
	diags.Append(d...)
	data.ConnectedServices = csList

	// Date Created and Last Updated (interface{} fields)
	if dateStr, ok := pp.DateCreated.(string); ok && (refresh || data.DateCreated.IsUnknown()) {
		data.DateCreated = types.StringValue(dateStr)
	} else if data.DateCreated.IsUnknown() {
		data.DateCreated = types.StringNull()
	}
	if dateStr, ok := pp.LastUpdated.(string); ok {
		data.LastUpdated = types.StringValue(dateStr)
	} else if refresh || data.LastUpdated.IsUnknown() {
		data.LastUpdated = types.StringNull()
	}
}

// int64FromAPI resolves an optional and computed integer from the API. When refresh is false,
// known values are kept.
func int64FromAPI(current types.Int64, api *int, refresh bool) types.Int64 {
	if !refresh && !current.IsUnknown() {
		return current
	}
	if api == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*api))
}

// boolFromAPI resolves an optional and computed boolean from the API. When refresh is false,
// known values are kept.
func boolFromAPI(current types.Bool, api *bool, refresh bool) types.Bool {
	if !refresh && !current.IsUnknown() {
		return current
	}
	if api == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*api)
}

func (r *PasswordPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
