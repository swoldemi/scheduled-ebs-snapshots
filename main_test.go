package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/lib"
)

type mockEC2Client struct {
	mock.Mock
	ec2iface.EC2API
}

// CreateSnapshot mocks the CreateSnapshot EC2 API endpoint.
func (_m *mockEC2Client) CreateSnapshotWithContext(ctx aws.Context, input *ec2.CreateSnapshotInput, opts ...request.Option) (*ec2.Snapshot, error) {
	log.Debugf("Mocking CreateSnapshot API for volume: %s\n", *input.VolumeId)
	args := _m.Called(ctx, input)
	return args.Get(0).(*ec2.Snapshot), args.Error(1)
}

func testHandlerWithEvent(svc ec2iface.EC2API, event events.CloudWatchEvent) error {
	h, err := lib.FunctionContainer(svc)
	if err != nil {
		return err
	}
	if err := h(context.Background(), event); err != nil {
		return err
	}
	return nil
}

func TestHandler(t *testing.T) {
	defaultEvent := events.CloudWatchEvent{
		Version:    "0",
		ID:         "89d1a02d-5ec7-412e-82f5-13505f849b41",
		DetailType: "Scheduled Event",
		Source:     "aws.events",
		AccountID:  "123456789012",
		Time:       time.Now(),
		Region:     "us-east-1",
		Resources:  []string{"arn:aws:events:us-east-1:123456789012:rule/SampleRule"},
		Detail:     json.RawMessage{},
	}

	if err := os.Setenv("VOLUME_ID", "vol-test"); err != nil {
		t.Fatalf("Error setting VOLUME_ID environment variable: %v\n", err)
	}
	if err := os.Setenv("ROLE_ARN", "arn:aws:iam::123456789012:role/SampleRole"); err != nil {
		t.Fatalf("Error setting ROLE_ARN environment variable: %v\n", err)
	}
	if err := os.Setenv("ROLE_EXTERNAL_ID", "SampleExternalID"); err != nil {
		t.Fatalf("Error setting ROLE_EXTERNAL_ID environment variable: %v\n", err)
	}

	volumeID := os.Getenv("VOLUME_ID")
	description, err := lib.FormatSnapshotDescription(volumeID, defaultEvent)
	if err != nil {
		t.Fatalf("Error formatting snapshot description: %v\n", err)

	}
	defaultInput := &ec2.CreateSnapshotInput{
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

	m := new(mockEC2Client)
	m.On("CreateSnapshotWithContext", context.Background(), defaultInput).
		Return(&ec2.Snapshot{}, nil)

	tests := []struct {
		name    string
		event   events.CloudWatchEvent
		wantErr bool
	}{
		{"SuccessfulInvocation", defaultEvent, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testHandlerWithEvent(m, tt.event); (err != nil) != tt.wantErr {
				t.Errorf("testHandlerWithEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			m.AssertExpectations(t)
		})
	}
}
