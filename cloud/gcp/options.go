package gcp

type CloudOption func(options *Cloud)

// WithProjectID sets the GCP project id.
func WithProjectID(projectID string) CloudOption {
	return func(options *Cloud) {
		options.projectID = projectID
	}
}

// WithDefaultTableName sets the default bigQuery table name. The default value is "billing_export.gcp_billing_export_v1".
func WithDefaultTableName(defaultTableName string) CloudOption {
	return func(options *Cloud) {
		options.defaultTableName = defaultTableName
	}
}

// WithResourceTableName sets the bigQuery table name containing resource names. The default value is
// "billing_export.gcp_billing_export_resource_v1".
func WithResourceTableName(resourceTableName string) CloudOption {
	return func(options *Cloud) {
		options.resourceTableName = resourceTableName
	}
}
