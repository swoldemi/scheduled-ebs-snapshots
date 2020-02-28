package lib

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
)

const maxTagsPerResource = 50

var ErrVolumeNotFound = errors.New("unable to find volume")

// InvokeSnapshot sets up and creates a snapshot of the given EBS volume.
func InvokeSnapshot(ctx context.Context, input *ec2.CreateSnapshotInput, svc ec2iface.EC2API) error {
	volumeInfo, err := svc.DescribeVolumesWithContext(
		ctx,
		&ec2.DescribeVolumesInput{
			Filters: []*ec2.Filter{
				{Name: aws.String("volume-id"), Values: []*string{input.VolumeId}},
			},
		},
	)
	if err != nil {
		log.Errorf("Error locating volume %s for snapshot invocation: %v\n", *input.VolumeId, err)
		return err
	}

	if len(volumeInfo.Volumes) == 0 {
		log.Errorf("Unable to find volume with ID %s\n", *input.VolumeId)
		return ErrVolumeNotFound
	}

	if len(volumeInfo.Volumes[0].Tags) != 0 {
		log.Infof("Merging existing tags from volume %s\n", *input.VolumeId)
		for i, tag := range volumeInfo.Volumes[0].Tags {
			if len(input.TagSpecifications[0].Tags) == maxTagsPerResource {
				log.Warnf("Reached per resource limit of 50 tags: skipping tags %+v\n", volumeInfo.Volumes[0].Tags[i:])
				break
			}
			log.Debugf("Adding tag %+v to snapshot\n", tag)
			input.TagSpecifications[0].Tags = append(input.TagSpecifications[0].Tags, tag)
		}
	}

	log.Infof("Invoking snapshot creation with description: %s\n", *input.Description)
	if err := input.Validate(); err != nil {
		log.Errorf("Error constructing input for CreateSnapshot API call: %v\n", err)
		return err
	}

	if _, err := svc.CreateSnapshotWithContext(ctx, input); err != nil {
		log.Errorf("Error calling CreateSnapshot API: %v\n", err)
		return err
	}
	log.Info("Completed CreateSnapshot API call")
	return nil
}

// PutSnapshotMetric sends metrics related to the current snapshot invocation to CloudWatch.
func PutSnapshotMetric(ctx context.Context, input *cloudwatch.PutMetricDataInput, svc cloudwatchiface.CloudWatchAPI) error {
	log.Infof("Submitting snapshot metric to CloudWatch namespace: %s\n", *input.Namespace)

	if err := input.Validate(); err != nil {
		log.Errorf("Error constructing input for PutMetricData API call: %v\n", err)
		return err
	}

	if _, err := svc.PutMetricDataWithContext(ctx, input); err != nil {
		log.Errorf("Error calling PutMetricData API: %v\n", err)
		return err
	}
	log.Info("Completed PutMetricData API call")
	return nil
}
