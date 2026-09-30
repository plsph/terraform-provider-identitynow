package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerDataSource(NewAuthorizationRightSetsDataSource)
}

// authorizationRightSetsPageSize is the maximum page size of the right sets endpoint.
const authorizationRightSetsPageSize = 50

// AuthorizationRightSet is a UI-assignable right set as returned by the experimental
// /v2026/authorization/authorization-assignable-right-sets API. Children are nested.
type AuthorizationRightSet struct {
	ID           string                             `json:"id"`
	Name         string                             `json:"name"`
	Description  *string                            `json:"description,omitempty"`
	Category     string                             `json:"category"`
	NestedConfig *AuthorizationRightSetNestedConfig `json:"nestedConfig,omitempty"`
	Children     []AuthorizationRightSet            `json:"children,omitempty"`
}

// AuthorizationRightSetNestedConfig describes the position of a right set in the hierarchy.
type AuthorizationRightSetNestedConfig struct {
	AncestorID  string   `json:"ancestorId,omitempty"`
	Depth       int64    `json:"depth"`
	ParentID    *string  `json:"parentId,omitempty"`
	ChildrenIDs []string `json:"childrenIds,omitempty"`
}

// ListAuthorizationRightSets lists the assignable right sets, optionally filtered by category.
func (c *Client) ListAuthorizationRightSets(ctx context.Context, category string) ([]AuthorizationRightSet, error) {
	query := url.Values{}
	if category != "" {
		query.Set("filters", eqFilter("category", category))
	}
	return listAllPagesSized[AuthorizationRightSet](ctx, c, "/v2026/authorization/authorization-assignable-right-sets", query, authorizationRightSetsPageSize, withExperimental())
}

var _ datasource.DataSource = &AuthorizationRightSetsDataSource{}

func NewAuthorizationRightSetsDataSource() datasource.DataSource {
	return &AuthorizationRightSetsDataSource{}
}

type AuthorizationRightSetsDataSource struct {
	client *Config
}

type AuthorizationRightSetsModel struct {
	Category  types.String `tfsdk:"category"`
	RightSets types.List   `tfsdk:"right_sets"`
}

type AuthorizationRightSetModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
	ParentID    types.String `tfsdk:"parent_id"`
	AncestorID  types.String `tfsdk:"ancestor_id"`
	Depth       types.Int64  `tfsdk:"depth"`
	ChildrenIDs types.List   `tfsdk:"children_ids"`
}

var authorizationRightSetObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":           types.StringType,
	"name":         types.StringType,
	"description":  types.StringType,
	"category":     types.StringType,
	"parent_id":    types.StringType,
	"ancestor_id":  types.StringType,
	"depth":        types.Int64Type,
	"children_ids": types.ListType{ElemType: types.StringType},
}}

func (d *AuthorizationRightSetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_right_sets"
}

func (d *AuthorizationRightSetsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the right sets that can be assigned to custom user levels. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				MarkdownDescription: "Only return right sets of this category, e.g. `identity`.",
				Optional:            true,
			},
			"right_sets": schema.ListNestedAttribute{
				MarkdownDescription: "Right sets, with the children of each right set following their parent.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":           schema.StringAttribute{MarkdownDescription: "Right set ID, used in `right_sets` of `identitynow_custom_user_level`.", Computed: true},
					"name":         schema.StringAttribute{MarkdownDescription: "Right set name.", Computed: true},
					"description":  schema.StringAttribute{MarkdownDescription: "Right set description.", Computed: true},
					"category":     schema.StringAttribute{MarkdownDescription: "Right set category.", Computed: true},
					"parent_id":    schema.StringAttribute{MarkdownDescription: "ID of the parent right set, null for top-level right sets.", Computed: true},
					"ancestor_id":  schema.StringAttribute{MarkdownDescription: "ID of the top-level ancestor right set.", Computed: true},
					"depth":        schema.Int64Attribute{MarkdownDescription: "Depth in the hierarchy, 0 for top-level right sets.", Computed: true},
					"children_ids": schema.ListAttribute{MarkdownDescription: "IDs of the child right sets.", Computed: true, ElementType: types.StringType},
				}},
			},
		},
	}
}

func (d *AuthorizationRightSetsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// authorizationRightSetsFlatten returns the right sets in depth-first order, parents before children.
func authorizationRightSetsFlatten(rightSets []AuthorizationRightSet, parentID *string) []AuthorizationRightSet {
	var flat []AuthorizationRightSet
	for _, rightSet := range rightSets {
		if rightSet.NestedConfig == nil {
			rightSet.NestedConfig = &AuthorizationRightSetNestedConfig{ParentID: parentID}
		}
		children := rightSet.Children
		rightSet.Children = nil
		flat = append(flat, rightSet)
		id := rightSet.ID
		flat = append(flat, authorizationRightSetsFlatten(children, &id)...)
	}
	return flat
}

// authorizationRightSetsState converts the right set hierarchy to the flat list attribute.
func authorizationRightSetsState(ctx context.Context, rightSets []AuthorizationRightSet, diags *diag.Diagnostics) types.List {
	flat := authorizationRightSetsFlatten(rightSets, nil)
	models := make([]AuthorizationRightSetModel, 0, len(flat))
	for _, rightSet := range flat {
		config := rightSet.NestedConfig
		childrenIDs := config.ChildrenIDs
		if childrenIDs == nil {
			childrenIDs = []string{}
		}
		children, d := types.ListValueFrom(ctx, types.StringType, childrenIDs)
		diags.Append(d...)
		model := AuthorizationRightSetModel{
			ID:          types.StringValue(rightSet.ID),
			Name:        types.StringValue(rightSet.Name),
			Description: types.StringNull(),
			Category:    types.StringValue(rightSet.Category),
			ParentID:    types.StringNull(),
			AncestorID:  stringValueOrNull(config.AncestorID),
			Depth:       types.Int64Value(config.Depth),
			ChildrenIDs: children,
		}
		if rightSet.Description != nil {
			model.Description = types.StringValue(*rightSet.Description)
		}
		if config.ParentID != nil && *config.ParentID != "" {
			model.ParentID = types.StringValue(*config.ParentID)
		}
		models = append(models, model)
	}
	list, d := types.ListValueFrom(ctx, authorizationRightSetObjectType, models)
	diags.Append(d...)
	return list
}

func (d *AuthorizationRightSetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AuthorizationRightSetsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	rightSets, err := client.ListAuthorizationRightSets(ctx, data.Category.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list authorization right sets: %s", err))
		return
	}
	data.RightSets = authorizationRightSetsState(ctx, rightSets, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
