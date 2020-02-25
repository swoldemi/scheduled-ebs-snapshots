package lib

import (
	"context"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
)

// InvokeSnapshot sets up and creates a snapshot of the given EBS volume.
func InvokeSnapshot(ctx context.Context, input *ec2.CreateSnapshotInput, svc ec2iface.EC2API) error {
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
