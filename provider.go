package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	providerDefaultEmptyString = "nil"
)

var (
	descriptions map[string]string
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_URL", providerDefaultEmptyString),
				Description: descriptions["api_url"],
			},
			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_CLIENT_ID", providerDefaultEmptyString),
				Description: descriptions["client_id"],
			},
			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_CLIENT_SECRET", providerDefaultEmptyString),
				Description: descriptions["client_secret"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"identitynow_source":                       resourceSource(),
			"identitynow_access_profile":               resourceAccessProfile(),
			"identitynow_role":                         resourceRole(),
			"identitynow_account_aggregation_schedule": resourceScheduleAccountAggregation(),
			"identitynow_account_schema_attribute":     resourceAccountSchema(),
			"identitynow_password_policy":              resourcePasswordPolicy(),
			"identitynow_governance_group":             resourceGovernanceGroup(),
			"identitynow_source_app":                   resourceSourceApp(),
			"identitynow_access_profile_attachment":    resourceAccessProfileAttachment(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"identitynow_source":             dataSourceSource(),
			"identitynow_access_profile":     dataSourceAccessProfile(),
			"identitynow_source_entitlement": dataSourceSourceEntitlement(),
			"identitynow_identity":           dataSourceIdentity(),
			"identitynow_role":               dataSourceRole(),
			"identitynow_governance_group":   dataSourceGovernanceGroup(),
			"identitynow_source_app":         dataSourceApp(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func init() {
	descriptions = map[string]string{
		"api_url":       "The URL to the IdentityNow API",
		"client_id":     "API client used to authenticate with the IdentityNow API",
		"client_secret": "API client secret used to authenticate with the IdentityNow API",
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiURL := d.Get("api_url").(string)
	clientId := d.Get("client_id").(string)
	clientSecret := d.Get("client_secret").(string)

	config := &Config{
		URL:          apiURL,
		ClientId:     clientId,
		ClientSecret: clientSecret,
	}

	return config, nil
}
