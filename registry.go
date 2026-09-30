package main

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// registeredResources and registeredDataSources hold resources and data sources that register
// themselves in an init function of their own file, so adding one does not require editing the
// provider.
var (
	registeredResources   []func() resource.Resource
	registeredDataSources []func() datasource.DataSource
)

func registerResource(constructors ...func() resource.Resource) {
	registeredResources = append(registeredResources, constructors...)
}

func registerDataSource(constructors ...func() datasource.DataSource) {
	registeredDataSources = append(registeredDataSources, constructors...)
}
