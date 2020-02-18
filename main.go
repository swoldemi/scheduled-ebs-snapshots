package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-xray-sdk-go/xray"
	log "github.com/sirupsen/logrus"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/config"
)

var (
	env *config.Environment
)

func buildDescription(volumeID string, event events.CloudWatchEvent) (string, error) {
	dstAcc, err := arn.Parse(os.Getenv("ROLE_ARN"))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`
Snapshot of volume %s at %s. 
Source account: %s. Destination Account: %s`,
		volumeID,
		event.Time.Format(time.RFC1123),
		event.AccountID,
		dstAcc.AccountID,
	), nil
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
	log.Infof("Received event at %s for volume %s\n", env.VolumeID, event.Time.Format(time.RFC1123))

	description, err := buildDescription(env.VolumeID, event)
	if err != nil {
		log.Errorf("Error constructing description for CreateSnapshotInput: %v\n", err)
		return err
	}
	input := &ec2.CreateSnapshotInput{
		Description:       aws.String(description),
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
