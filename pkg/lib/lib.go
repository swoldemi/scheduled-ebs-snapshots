// Package lib contains library units for the scheduled-ebs-snapshot Lambda function.
package lib

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
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

// InvokeSnapshot sets up and creates a snapshot of the given EBS volume.
func InvokeSnapshot(ctx context.Context, description, volumeID string, svc ec2iface.EC2API) error {
	log.Infof("Invoking snapshot creation with description: %s\n", description)
	input := &ec2.CreateSnapshotInput{
		Description: aws.String(description),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("snapshot"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("timestamp"),
						Value: aws.String(time.Now().UTC().Format(time.RFC1123)),
					},
				},
			}},
		VolumeId: aws.String(volumeID),
	}
	if err := input.Validate(); err != nil {
		log.Errorf("Error constructing input for CreateSnapshot API call: %v\n", err)
		return err
	}

	snapshot, err := svc.CreateSnapshotWithContext(ctx, input)
	if err != nil {
		log.Errorf("Error calling CreateSnapshot API: %v\n", err)
		return err
	}
	log.Infof("Got Snapshot response: %s\n", snapshot.String())
	return nil
}

// FormatSnapshotDescription formats the description for an EBS volume snapshot,
// using the ID of the volume and the CloudWatch event that triggered the snapshot.
func FormatSnapshotDescription(volumeID string, event events.CloudWatchEvent) (string, error) {
	var dstAcc string

	role := os.Getenv("ROLE_ARN")
	switch role {
	case "":
		dstAcc = event.AccountID
	default:
		r, err := arn.Parse(role)
		if err != nil {
			return "", err
		}
		dstAcc = r.AccountID
	}

	return fmt.Sprintf(`Snapshot of volume %s at %s. Source account: %s. Destination Account: %s`,
		volumeID,
		event.Time.Format(time.RFC1123),
		event.AccountID,
		dstAcc,
	), nil
}

// Handler is a type alias for the signature of the Lambda handler.
type Handler func(context.Context, events.CloudWatchEvent) error

// FunctionContainer contains the business logic for the scheduled-ebs-snapshots Lambda function.
func FunctionContainer(svc ec2iface.EC2API) (Handler, error) {
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
		log.Infof("Received event at %s for volume %s\n", event.Time.Format(time.RFC1123), volumeID)

		description, err := FormatSnapshotDescription(volumeID, event)
		if err != nil {
			log.Errorf("Error constructing description for CreateSnapshotInput: %v\n", err)
			return err
		}
		if err := InvokeSnapshot(ctx, description, volumeID, svc); err != nil {
			log.Errorf("Error during snapshot creation: %v\n", err)
			return err
		}
		return nil
	}, nil
}
