// Package lib contains library units for the scheduled-ebs-snapshot Lambda function.
package lib

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrInvalidVolumeID is returned when a Lambda is created without providing an EBS volume ID.
	ErrInvalidVolumeID = errors.New("must provide volume ID")

	// ErrInvalidVolumePrefix is rturned when a Lambda is provided with an
	// EBS volume ID that does not begin with the 'vol-' prefix.
	ErrInvalidVolumePrefix = errors.New("volume ID must begin with 'vol-' prefix")
)

// Environment denotes different environments.
type Environment string

const (
	// Production denotes a production environment.
	Production Environment = "production"

	// Development denotes a development environment.
	Development Environment = "development"
)

// FunctionContainer contains the dependencies and business logic for the scheduled-ebs-snapshots Lambda function.
type FunctionContainer struct {
	Environment Environment
	EC2         ec2iface.EC2API
	CloudWatch  cloudwatchiface.CloudWatchAPI
}

// NewFunctionContainer creates a new FunctionContainer.
func NewFunctionContainer(ec2Svc ec2iface.EC2API, cwSvc cloudwatchiface.CloudWatchAPI, env Environment) *FunctionContainer {
	log.Infof("Creating function container for environment: %v", env)
	return &FunctionContainer{
		Environment: env,
		EC2:         ec2Svc,
		CloudWatch:  cwSvc,
	}
}

// GetHandler returns the function handler for scheduled-ebs-snapshots.
func (f *FunctionContainer) GetHandler() (func(context.Context, events.CloudWatchEvent) error, error) {
	volumeID := os.Getenv("VOLUME_ID")
	if volumeID == "" {
		log.Error(ErrInvalidVolumeID)
		return nil, ErrInvalidVolumeID
	}
	if !strings.HasPrefix(volumeID, "vol-") {
		log.Error(ErrInvalidVolumePrefix)
		return nil, ErrInvalidVolumePrefix
	}

	return func(ctx context.Context, event events.CloudWatchEvent) error {
		log.Infof("Received event at %s for volume %s\n", event.Time.UTC(), volumeID)

		description, err := NewSnapshotDescription(volumeID, event)
		if err != nil {
			log.Errorf("Error constructing SnapshotDescription: %v\n", err)
			return err
		}
		if err := InvokeSnapshot(ctx, description.CreateSnapshotInput(), f.EC2); err != nil {
			log.Errorf("Error during snapshot creation: %v\n", err)
			return err
		}
		if err := PutSnapshotMetric(ctx, description.PutMetricDataInput(), f.CloudWatch); err != nil {
			log.Errorf("Error sending metrics: %v\n", err)
			return err
		}
		return nil
	}, nil
}
