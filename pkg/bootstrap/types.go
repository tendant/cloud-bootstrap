package bootstrap

// Config represents the AWS resources configuration
type Config struct {
	Region          string          `yaml:"region"`
	S3Buckets       []S3Bucket      `yaml:"s3_buckets"`
	ECRRepositories []ECRRepository `yaml:"ecr_repositories"`
	IAMUsers        []IAMUser       `yaml:"iam_users"`
}

// S3Bucket represents an S3 bucket configuration
type S3Bucket struct {
	Name       string      `yaml:"name"`
	Versioning string      `yaml:"versioning"`
	Encryption string      `yaml:"encryption"`
	CORS       *CORSConfig `yaml:"cors,omitempty"`
	Policy     string      `yaml:"policy,omitempty"`
}

// CORSConfig represents CORS configuration for an S3 bucket
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	ExposeHeaders  []string `yaml:"expose_headers,omitempty"`
	MaxAgeSeconds  int      `yaml:"max_age_seconds"`
}

// ECRRepository represents an ECR repository configuration
type ECRRepository struct {
	Name           string `yaml:"name"`
	LifecyclePolicy string `yaml:"lifecycle_policy,omitempty"`
}

// IAMUser represents an IAM user configuration
type IAMUser struct {
	Name     string      `yaml:"name"`
	Policies []IAMPolicy `yaml:"policies"`
}

// IAMPolicy represents an IAM policy configuration
type IAMPolicy struct {
	Name           string `yaml:"name"`
	Description    string `yaml:"description"`
	PolicyDocument string `yaml:"policy_document"`
}
