package main

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/swoldemi/lambda-ebs-snapshot/pkg/config"
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

func mockEC2Provider(sess *session.Session) ec2iface.EC2API {
	_ = sess
	return &mockEC2Client{}
}

var defaultEvent = events.CloudWatchEvent{}

func TestHandler(t *testing.T) {
	env, err := config.NewEnvironment(mockEC2Provider)
	if err != nil {
		t.Fatalf("Error creating environment with mock EC2 client: %v", err)
	}
	_ = env

	if err := os.Setenv("VOLUME_ID", "TEST_VOLUME"); err != nil {
		t.Fatalf("Error setting VOLUME_ID environment variable: %v", err)
	}
	if err := os.Setenv("ROLE_ARN", "arn:aws:iam::123456789012:role/SampleRole"); err != nil {
		t.Fatalf("Error setting ROLE_ARN environment variable: %v", err)
	}
	if err := os.Setenv("ROLE_EXTERNAL_ID", ""); err != nil {
		t.Fatalf("Error setting ROLE_EXTERNAL_ID environment variable: %v", err)
	}
	tests := []struct {
		name    string
		event   events.CloudWatchEvent
		wantErr bool
	}{
		{"SuccessfulInvocation", defaultEvent, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Handler(context.Background(), tt.event); (err != nil) != tt.wantErr {
				t.Errorf("Handler() error = %v, wantErr %v", err, tt.wantErr)

			}
		})
	}
}
