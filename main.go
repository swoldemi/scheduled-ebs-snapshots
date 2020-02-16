package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-xray-sdk-go/xray"
	log "github.com/sirupsen/logrus"
	"github.com/swoldemi/lambda-ebs-snapshot/pkg/config"
)

var (
	env *config.Environment
)

func buildDescription(volumeID string, ts string) string {
	return fmt.Sprintf("Snapshot of volume %s at %s", volumeID, ts)
}

func ec2Provider(sess *session.Session) ec2iface.EC2API {
	svc := ec2.New(sess)
	xray.AWS(svc.Client)
	return svc
}

// Handler handles the CrossAccountEBSVolumeSnapshotEvent. The execution flow is as follows:
// 1. Using the role granted to the Lambda function, create a session
// 2. Using the session, assume a role into the account containing the EBS volume
// 3. Using the credentials in the assumed role, create an Amazon EBS service client
// 4. Using the service client, create a snapshot of the volume
// 5. In the event of any errors, retry 3 times
// 6. Respond with the status of the snapshot creation
func Handler(ctx context.Context, event events.CloudWatchEvent) error {
	ts := time.Now().Format(time.RFC1123)
	log.Infof("Received event at %s for volume %s\n", env.VolumeID, ts)

	input := &ec2.CreateSnapshotInput{
		Description:       aws.String(buildDescription(env.VolumeID, ts)),
		VolumeId:          aws.String(env.VolumeID),
		TagSpecifications: []*ec2.TagSpecification{},
	}
	if err := input.Validate(); err != nil {
		log.Errorf("Error constructing input for CreateSnapshot API call: %v\n", err)
		return err
	}

	snapshot, err := env.EC2Client.CreateSnapshotWithContext(ctx, input)
	if err != nil {
		log.Errorf("Error calling CreateSnapshot API: %v\n", err)
		return err
	}
	log.Infof("Got Snapshot response: %+v\n", snapshot)
	return nil
}

func main() {
	log.Info("Starting Lambda in live environment")
	env, err := config.NewEnvironment(ec2Provider)
	if err != nil {
		log.Fatalf("Error while creating new environment: %v\n", err)

		return
	}

	if err := env.Validate(); err != nil {
		log.Fatalf("Error while validating environment: %v\n", err)
		return
	}
	lambda.Start(Handler)
}
