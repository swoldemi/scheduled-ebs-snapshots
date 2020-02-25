package lib

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

const noRoleWarning = `No role found. Assuming volume source account and snapshot destination account are the same.`

// SnapshotDescription encapsulates the data used to describe a snapshot.
type SnapshotDescription struct {
	DestinationAccount string
	SourceAccount      string
	Timestamp          string
	VolumeID           string
}

// NewSnapshotDescription returns the description for an EBS volume snapshot,
// using the ID of the volume and the CloudWatch event that triggered the snapshot.
func NewSnapshotDescription(volumeID string, event events.CloudWatchEvent) (*SnapshotDescription, error) {
	var dstAcc string

	role := os.Getenv("ROLE_ARN")
	switch role {
	case "":
		log.Warn(noRoleWarning)
		dstAcc = event.AccountID
	default:
		r, err := arn.Parse(role)
		if err != nil {
			return nil, err
		}
		dstAcc = r.AccountID
	}

	return &SnapshotDescription{
		DestinationAccount: dstAcc,
		SourceAccount:      event.AccountID,
		Timestamp:          event.Time.UTC().Format(time.RFC1123),
		VolumeID:           volumeID,
	}, nil
}

// String returns a string formatting of the current SnapshotDescription.
func (s *SnapshotDescription) String() string {
	return fmt.Sprintf(`Snapshot of volume %s at %s. Source account: %s. Destination Account: %s`,
		s.VolumeID,
		s.Timestamp,
		s.SourceAccount,
		s.DestinationAccount,
	)
}

// CreateSnapshotInput returns the related ec2.CreateSnapshotInput from the data
// provided for formatting the snapshot description.
func (s *SnapshotDescription) CreateSnapshotInput() *ec2.CreateSnapshotInput {
	return &ec2.CreateSnapshotInput{
		Description: aws.String(s.String()),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("snapshot"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("timestamp"),
						Value: aws.String(s.Timestamp),
					},
					{
						Key:   aws.String("source-account"),
						Value: aws.String(s.SourceAccount),
					},
				},
			},
		},
		VolumeId: aws.String(s.VolumeID),
	}
}

// PutMetricDataInput returns the input for PutSnapshotMetric.
// Metric dimensions: SourceAccountID, DestinationAccountID, VolumeID.
func (s *SnapshotDescription) PutMetricDataInput() *cloudwatch.PutMetricDataInput {
	dimensions := []*cloudwatch.Dimension{
		{
			Name:  aws.String("SourceAccountID"),
			Value: aws.String(s.SourceAccount),
		},
		{
			Name:  aws.String("DestinationAccountID"),
			Value: aws.String(s.DestinationAccount),
		},
		{
			Name:  aws.String("VolumeID"),
			Value: aws.String(s.VolumeID),
		},
	}
	return &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("scheduled-ebs-snapshots"),
		MetricData: []*cloudwatch.MetricDatum{
			{
				Dimensions:        dimensions,
				MetricName:        aws.String("SnapshotCount"),
				StorageResolution: aws.Int64(1),
				Unit:              aws.String("Count"),
				Value:             aws.Float64(1),
			},
		},
	}
}
