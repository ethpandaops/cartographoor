package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cartographoor/pkg/discovery"
)

// Config represents the configuration for the S3 storage provider.
type Config struct {
	BucketName           string        `mapstructure:"bucketName"`
	Key                  string        `mapstructure:"key"`
	Region               string        `mapstructure:"region"`
	Endpoint             string        `mapstructure:"endpoint"`
	AccessKey            string        `mapstructure:"accessKey"`
	SecretKey            string        `mapstructure:"secretKey"`
	ForcePathStyle       bool          `mapstructure:"forcePathStyle"`
	DisableSSL           bool          `mapstructure:"disableSSL"`
	ContentType          string        `mapstructure:"contentType"`
	ACL                  string        `mapstructure:"acl"`
	RetryDuration        time.Duration `mapstructure:"retryDuration"`
	MaxRetries           int           `mapstructure:"maxRetries"`
	BackoffJitterPercent int           `mapstructure:"backoffJitterPercent"`
}

// Provider implements the storage provider interface for S3.
type Provider struct {
	log    *logrus.Logger
	config Config
	client *s3.Client
}

// NewProvider creates a new S3 storage provider.
func NewProvider(log *logrus.Logger, cfg Config) (*Provider, error) {
	log = log.WithField("module", "storage_s3").Logger

	// Set default values if not specified
	if cfg.Key == "" {
		cfg.Key = "networks.json"
	}

	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}

	if cfg.RetryDuration == 0 {
		cfg.RetryDuration = 5 * time.Second
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.BackoffJitterPercent == 0 {
		cfg.BackoffJitterPercent = 20
	}

	return &Provider{
		log:    log,
		config: cfg,
	}, nil
}

// Initialize sets up the S3 client.
func (p *Provider) Initialize(ctx context.Context) error {
	p.log.WithFields(logrus.Fields{
		"endpoint":       p.config.Endpoint,
		"bucket":         p.config.BucketName,
		"region":         p.config.Region,
		"forcePathStyle": p.config.ForcePathStyle,
	}).Debug("Initializing S3 client")

	// Create AWS config
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(p.config.Region),
	}

	// Add custom endpoint if provided
	if p.config.Endpoint != "" {
		p.log.WithField("endpoint", p.config.Endpoint).Debug("Using custom S3 endpoint")
		//nolint:staticcheck // fine.
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               p.config.Endpoint,
				SigningRegion:     p.config.Region,
				HostnameImmutable: true,
			}, nil
		})

		//nolint:staticcheck // fine.
		opts = append(opts, config.WithEndpointResolverWithOptions(customResolver))
	}

	// Add credentials if provided
	if p.config.AccessKey != "" && p.config.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     p.config.AccessKey,
				SecretAccessKey: p.config.SecretKey,
			}, nil
		})))
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Opts := []func(*s3.Options){}
	if p.config.ForcePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	p.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		for _, opt := range s3Opts {
			opt(o)
		}
	})

	return nil
}

// Upload uploads the discovery result to S3.
func (p *Provider) Upload(ctx context.Context, result discovery.Result) error {
	if p.client == nil {
		if err := p.Initialize(ctx); err != nil {
			return err
		}
	}

	// Marshal result to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal discovery result: %w", err)
	}

	// Create S3 put object input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(p.config.BucketName),
		Key:         aws.String(p.config.Key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(p.config.ContentType),
	}

	// Add ACL if configured
	if p.config.ACL != "" {
		input.ACL = types.ObjectCannedACL(p.config.ACL)
	}

	// Upload to S3
	p.log.WithFields(logrus.Fields{
		"bucket": p.config.BucketName,
		"key":    p.config.Key,
		"acl":    p.config.ACL,
	}).Info("Uploading networks to S3")

	if _, err = p.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	p.log.Info("S3 upload completed successfully")

	return nil
}

// Download downloads a file from S3.
func (p *Provider) Download(ctx context.Context, key string) ([]byte, error) {
	if p.client == nil {
		if err := p.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	p.log.WithFields(logrus.Fields{
		"bucket": p.config.BucketName,
		"key":    key,
	}).Debug("Downloading from S3")

	// Create S3 get object input
	input := &s3.GetObjectInput{
		Bucket: aws.String(p.config.BucketName),
		Key:    aws.String(key),
	}

	// Download from S3 with retry logic
	var (
		result *s3.GetObjectOutput
		err    error
	)

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		result, err = p.client.GetObject(ctx, input)
		if err == nil {
			break
		}

		if attempt < p.config.MaxRetries {
			p.log.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err,
			}).Warn("S3 download failed, retrying")
			time.Sleep(p.config.RetryDuration)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	defer result.Body.Close()

	// Read response body
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(result.Body); err != nil {
		return nil, fmt.Errorf("failed to read S3 response body: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"bucket": p.config.BucketName,
		"key":    key,
		"size":   buf.Len(),
	}).Debug("S3 download completed successfully")

	return buf.Bytes(), nil
}

// UploadRaw uploads raw data to S3 with a specific key.
func (p *Provider) UploadRaw(ctx context.Context, key string, data []byte, contentType string) error {
	if p.client == nil {
		if err := p.Initialize(ctx); err != nil {
			return err
		}
	}

	// Create S3 put object input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(p.config.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	// Add ACL if configured
	if p.config.ACL != "" {
		input.ACL = types.ObjectCannedACL(p.config.ACL)
	}

	// Upload to S3
	p.log.WithFields(logrus.Fields{
		"bucket": p.config.BucketName,
		"key":    key,
		"size":   len(data),
		"acl":    p.config.ACL,
	}).Debug("Uploading raw data to S3")

	if _, err := p.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	p.log.WithField("key", key).Debug("S3 raw upload completed successfully")

	return nil
}
