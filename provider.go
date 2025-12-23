package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
			"max_client_pool_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_MAX_POOL_SIZE", 10),
				Description: descriptions["max_client_pool_size"],
			},
			"default_client_pool_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_DEF_POOL_SIZE", 5),
				Description: descriptions["default_client_pool_size"],
			},
			"client_request_rate_limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("IDENTITYNOW_CLI_RQ_RATE", 2),
				Description: descriptions["client_request_rate_limit"],
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
			"identitynow_governance_group_members":     resourceGovernanceGroupMembers(),
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
		"max_client_pool_size": "Max client pool size for communication with the IdentityNow API",
		"default_client_pool_size": "Defalut client pool size for communication with the IdentityNow API",
		"client_request_rate_limit": "Client request rate limit for communication with the IdentityNow API",
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	tflog.Info(ctx, "Configuring IdentityNow provider")

	apiURL := d.Get("api_url").(string)
	clientId := d.Get("client_id").(string)
	clientSecret := d.Get("client_secret").(string)
	maxClientPoolSize := d.Get("max_client_pool_size").(int)
	defaultClientPoolSize := d.Get("default_client_pool_size").(int)
	clientRequestRateLimit := d.Get("client_request_rate_limit").(int)

	tflog.Debug(ctx, "Provider configuration", map[string]interface{}{
		"api_url":   apiURL,
		"client_id": clientId,
		"max_client_pool_size": maxClientPoolSize,
		"default_client_pool_size": defaultClientPoolSize,
		"client_request_rate_limit": clientRequestRateLimit,
		// Note: client_secret is intentionally not logged for security
	})

	config := &Config{
		URL:                   apiURL,
		ClientId:              clientId,
		ClientSecret:          clientSecret,
		MaxClientPoolSize:     maxClientPoolSize,
		DefaultClientPoolSize: defaultClientPoolSize,
		ClientRequestRateLimit: clientRequestRateLimit,
	}

	tflog.Info(ctx, "Successfully configured IdentityNow provider")
	return config, nil
}
