package aws

import "github.com/aws/aws-sdk-go-v2/aws"

type CloudOption func(options *Cloud)

// WithCloudConfig sets the AWS config to use.
func WithCloudConfig(cfg aws.Config) CloudOption {
	return func(options *Cloud) {
		options.cfg = &cfg
	}
}
