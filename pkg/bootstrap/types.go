package bootstrap

// Config represents the AWS resources configuration
type Config struct {
	Region          string          `yaml:"region"`
	S3Buckets       []S3Bucket      `yaml:"s3_buckets"`
	ECRRepositories []ECRRepository `yaml:"ecr_repositories"`
	IAMUsers        []IAMUser       `yaml:"iam_users"`
	RDSInstances    []RDSInstance   `yaml:"rds_instances,omitempty"`
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

// RDSInstance represents an RDS database instance configuration
type RDSInstance struct {
	Identifier       string `yaml:"identifier"`
	Engine           string `yaml:"engine"`
	EngineVersion    string `yaml:"engine_version,omitempty"`
	InstanceClass    string `yaml:"instance_class"`
	StorageType      string `yaml:"storage_type,omitempty"`
	AllocatedStorage int    `yaml:"allocated_storage"`
	DBName           string `yaml:"db_name"`
	MasterUsername   string `yaml:"master_username,omitempty"`
	MasterPassword   string `yaml:"master_password,omitempty"`
	PubliclyAccessible bool  `yaml:"publicly_accessible,omitempty"`
	BackupRetentionPeriod int `yaml:"backup_retention_period,omitempty"`
	MultiAZ          bool   `yaml:"multi_az,omitempty"`
	SkipFinalSnapshot bool  `yaml:"skip_final_snapshot,omitempty"`
}
