package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &IdentityDataSource{}

func NewIdentityDataSource() datasource.DataSource {
	return &IdentityDataSource{}
}

type IdentityDataSource struct {
	client *Config
}

type IdentityDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Alias          types.String `tfsdk:"alias"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	EmailAddress   types.String `tfsdk:"email_address"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IsManager      types.Bool   `tfsdk:"is_manager"`
	IdentityStatus types.String `tfsdk:"identity_status"`
	Attributes     types.List   `tfsdk:"attributes"`
}

type IdentityAttributesModel struct {
	AdpID     types.String `tfsdk:"adp_id"`
	LastName  types.String `tfsdk:"lastname"`
	FirstName types.String `tfsdk:"firstname"`
	Phone     types.String `tfsdk:"phone"`
	UserType  types.String `tfsdk:"user_type"`
	UID       types.String `tfsdk:"uid"`
	Email     types.String `tfsdk:"email"`
	WorkdayID types.String `tfsdk:"workday_id"`
}

func (d *IdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *IdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Identity data source - looks up an identity by alias or email",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity ID",
			},
			"alias": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Identity alias",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity description",
			},
			"email_address": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Identity email address",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether enabled",
			},
			"is_manager": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the identity is a manager",
			},
			"identity_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity status",
			},
			"attributes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Identity attributes",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"adp_id":    schema.StringAttribute{Computed: true},
						"lastname":  schema.StringAttribute{Computed: true},
						"firstname": schema.StringAttribute{Computed: true},
						"phone":     schema.StringAttribute{Computed: true},
						"user_type": schema.StringAttribute{Computed: true},
						"uid":       schema.StringAttribute{Computed: true},
						"email":     schema.StringAttribute{Computed: true},
						"workday_id": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *IdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	alias := data.Alias.ValueString()
	email := data.EmailAddress.ValueString()

	if !data.Alias.IsNull() && !data.EmailAddress.IsNull() && alias != "" && email != "" {
		resp.Diagnostics.AddError("Configuration Error", "Only one of 'alias' or 'email_address' must be set")
		return
	}

	if (data.Alias.IsNull() || alias == "") && (data.EmailAddress.IsNull() || email == "") {
		resp.Diagnostics.AddError("Configuration Error", "One of 'alias' or 'email_address' must be set")
		return
	}

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	var identity *Identity

	if !data.Alias.IsNull() && alias != "" {
		tflog.Info(ctx, "Reading Identity data source by alias", map[string]interface{}{"alias": alias})
		identities, err := client.GetIdentityByAlias(ctx, alias)
		if err != nil {
			if _, notFound := err.(*NotFoundError); notFound {
				resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Identity with alias %s not found", alias))
				return
			}
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		if len(identities) > 0 {
			identity = identities[0]
		}
	} else if !data.EmailAddress.IsNull() && email != "" {
		tflog.Info(ctx, "Reading Identity data source by email", map[string]interface{}{"email": email})
		identities, err := client.GetIdentityByEmail(ctx, email)
		if err != nil {
			if _, notFound := err.(*NotFoundError); notFound {
				resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Identity with email %s not found", email))
				return
			}
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		if len(identities) > 0 {
			identity = identities[0]
		}
	}

	if identity == nil {
		resp.Diagnostics.AddError("Not Found", "Identity not found")
		return
	}

	data.ID = types.StringValue(identity.ID)
	data.Name = types.StringValue(identity.Name)
	data.Description = types.StringValue(identity.Description)
	data.Alias = types.StringValue(identity.Alias)
	data.EmailAddress = types.StringValue(identity.EmailAddress)
	data.Enabled = types.BoolValue(identity.Enabled)
	data.IsManager = types.BoolValue(identity.IsManager)
	data.IdentityStatus = types.StringValue(identity.IdentityStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
