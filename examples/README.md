# Examples

This directory contains examples that are mostly used for documentation, but can also be run and tested manually.

The document generation tool looks for files in the following locations by default. All other files are ignored by the tool.

* **provider/provider.tf** example file for the provider index page
* **data-sources/`full data source name`/data-source.tf** example file for the named data source page
* **resources/`full resource name`/resource.tf** example file for the named resource page
* **resources/`full resource name`/import.sh** example import command for the named resource page

Additional examples of a resource are kept next to `resource.tf` and referenced from its template in `templates`.

The [complete](complete) directory contains a single Terraform configuration that uses many of the resources together.
