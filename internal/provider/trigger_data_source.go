package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerDataSource(NewTriggerDataSource)
}

// TriggerDefinition is an event trigger as returned by the /v2026/triggers API.
type TriggerDefinition struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"inputSchema,omitempty"`
	ExampleInput interface{}     `json:"exampleInput,omitempty"`
}

// GetTriggerDefinition finds a trigger by ID with an id filter on the list endpoint.
func (c *Client) GetTriggerDefinition(ctx context.Context, id string) (*TriggerDefinition, error) {
	items, err := listAllPages[TriggerDefinition](ctx, c, "/v2026/triggers", url.Values{"filters": {eqFilter("id", id)}})
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("trigger %q not found", id)}
}

// GetTriggerDefinitionByName lists all triggers and returns the one with the given name. The list
// endpoint cannot filter by name.
func (c *Client) GetTriggerDefinitionByName(ctx context.Context, name string) (*TriggerDefinition, error) {
	items, err := listAllPages[TriggerDefinition](ctx, c, "/v2026/triggers", nil)
	if err != nil {
		return nil, err
	}
	var match *TriggerDefinition
	for i := range items {
		if items[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple triggers are named %q", name)
		}
		match = &items[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("trigger with name %q not found", name)}
	}
	return match, nil
}

var _ datasource.DataSource = &TriggerDataSource{}
var _ datasource.DataSourceWithValidateConfig = &TriggerDataSource{}

func NewTriggerDataSource() datasource.DataSource {
	return &TriggerDataSource{}
}

type TriggerDataSource struct {
	client *Config
}

type TriggerDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	Description      types.String `tfsdk:"description"`
	InputSchema      types.String `tfsdk:"input_schema"`
	ExampleInputJSON types.String `tfsdk:"example_input_json"`
}

func (d *TriggerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (d *TriggerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an event trigger available in the tenant by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Trigger ID, e.g. `idn:identity-created`. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Trigger name. All triggers are listed and matched by exact name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Trigger type: `REQUEST_RESPONSE` or `FIRE_AND_FORGET`.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Trigger description.",
				Computed:            true,
			},
			"input_schema": schema.StringAttribute{
				MarkdownDescription: "JSON schema of the payload sent by the trigger to subscribers.",
				Computed:            true,
			},
			"example_input_json": schema.StringAttribute{
				MarkdownDescription: "Example payload sent by the trigger, as a JSON document.",
				Computed:            true,
			},
		},
	}
}

func (d *TriggerDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *TriggerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// setTriggerDataSourceState maps an API trigger onto the model. The input schema is documented as a
// string, but an embedded JSON object is kept as its JSON text.
func setTriggerDataSourceState(data *TriggerDataSourceModel, trigger *TriggerDefinition) {
	data.ID = types.StringValue(trigger.ID)
	data.Name = types.StringValue(trigger.Name)
	data.Type = types.StringValue(trigger.Type)
	data.Description = types.StringValue(trigger.Description)
	data.InputSchema = types.StringNull()
	if len(trigger.InputSchema) > 0 && string(trigger.InputSchema) != "null" {
		var text string
		if err := json.Unmarshal(trigger.InputSchema, &text); err == nil {
			data.InputSchema = types.StringValue(text)
		} else {
			data.InputSchema = types.StringValue(string(trigger.InputSchema))
		}
	}
	data.ExampleInputJSON = types.StringNull()
	if trigger.ExampleInput != nil {
		if encoded, err := json.Marshal(trigger.ExampleInput); err == nil {
			data.ExampleInputJSON = types.StringValue(string(encoded))
		}
	}
}

func (d *TriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TriggerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var trigger *TriggerDefinition
	attribute := path.Root("id")
	if !data.ID.IsNull() {
		trigger, err = client.GetTriggerDefinition(ctx, data.ID.ValueString())
	} else {
		attribute = path.Root("name")
		trigger, err = client.GetTriggerDefinitionByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(attribute, "Trigger not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read trigger: %s", err))
		return
	}
	setTriggerDataSourceState(&data, trigger)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
