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
	// Create AWS config
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(p.config.Region),
	}

	// Add custom endpoint if provided
	if p.config.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               p.config.Endpoint,
				SigningRegion:     p.config.Region,
				HostnameImmutable: true,
			}, nil
		})
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

	// Upload to S3
	p.log.WithFields(logrus.Fields{
		"bucket": p.config.BucketName,
		"key":    p.config.Key,
	}).Info("Uploading networks to S3")

	if _, err = p.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	p.log.Info("S3 upload completed successfully")

	return nil
}
