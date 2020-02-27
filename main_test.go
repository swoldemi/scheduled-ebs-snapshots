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
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
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

type mockCWClient struct {
	mock.Mock
	cloudwatchiface.CloudWatchAPI
}

// CreateSnapshot mocks the CreateSnapshot EC2 API endpoint.
func (_m *mockEC2Client) CreateSnapshotWithContext(ctx aws.Context, input *ec2.CreateSnapshotInput, opts ...request.Option) (*ec2.Snapshot, error) {
	log.Debugf("Mocking CreateSnapshot API for volume: %s\n", *input.VolumeId)
	args := _m.Called(ctx, input)
	return args.Get(0).(*ec2.Snapshot), args.Error(1)
}

// CreateSnapshot mocks the PutMetricData CloudWatch API endpoint.
func (_m *mockCWClient) PutMetricDataWithContext(ctx aws.Context, input *cloudwatch.PutMetricDataInput, opts ...request.Option) (*cloudwatch.PutMetricDataOutput, error) {
	log.Debugf("Mocking PutMetricData API with input: %s\n", input.String())
	args := _m.Called(ctx, input)
	return args.Get(0).(*cloudwatch.PutMetricDataOutput), args.Error(1)
}

func testHandlerWithEvent(f *lib.FunctionContainer, event events.CloudWatchEvent) error {
	h, err := f.GetHandler()
	if err != nil {
		return err
	}
	if err := h(context.Background(), event); err != nil {
		return err
	}
	return nil
}

var defaultEvent = events.CloudWatchEvent{
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

func TestHandler(t *testing.T) {
	if err := os.Setenv("VOLUME_ID", "vol-test"); err != nil {
		t.Fatalf("Error setting VOLUME_ID environment variable: %v\n", err)
	}
	if err := os.Setenv("VOLUME_REGION", "us-east-6"); err != nil {
		t.Fatalf("Error setting VOLUME_ID environment variable: %v\n", err)
	}
	if err := os.Setenv("ROLE_ARN", "arn:aws:iam::123456789012:role/SampleRole"); err != nil {
		t.Fatalf("Error setting ROLE_ARN environment variable: %v\n", err)
	}
	if err := os.Setenv("ROLE_EXTERNAL_ID", "SampleExternalID"); err != nil {
		t.Fatalf("Error setting ROLE_EXTERNAL_ID environment variable: %v\n", err)
	}

	volumeID := os.Getenv("VOLUME_ID")
	description, err := lib.NewSnapshotDescription(volumeID, os.Getenv("VOLUME_REGION"), defaultEvent)
	if err != nil {
		t.Fatalf("Error formatting snapshot description: %v\n", err)
	}

	ec2Svc := new(mockEC2Client)
	ec2Svc.On("CreateSnapshotWithContext", context.Background(), description.CreateSnapshotInput()).
		Return(&ec2.Snapshot{}, nil)

	cwSvc := new(mockCWClient)
	cwSvc.On("PutMetricDataWithContext", context.Background(), description.PutMetricDataInput()).
		Return(&cloudwatch.PutMetricDataOutput{}, nil)

	f := lib.NewFunctionContainer(ec2Svc, cwSvc, lib.Development)
	if err := testHandlerWithEvent(f, defaultEvent); err != nil {
		t.Fatalf("testHandlerWithEvent() error = %v\n", err)
	}
	ec2Svc.AssertExpectations(t)
	cwSvc.AssertExpectations(t)
}
