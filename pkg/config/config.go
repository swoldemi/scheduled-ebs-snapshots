// Package config defines environment configuration types and methods.
package config

import (
	"errors"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	log "github.com/sirupsen/logrus"
	"github.com/swoldemi/lambda-ebs-snapshot/pkg/auth"
)

var (
	// ErrInvalidVolumeID is returned when the request contains a role with no external ID.
	ErrInvalidVolumeID = errors.New("must provide EBS volume ID")
)

// Environment defines the environment that the Lambda function is configured with.
type Environment struct {
	EC2Client ec2iface.EC2API
	VolumeID  string
}

// NewEnvironment creates a new Environment.
func NewEnvironment(svcProvider func(*session.Session) ec2iface.EC2API) (*Environment, error) {
	role := &auth.CrossAccountRole{
		ARN:        os.Getenv("ROLE_ARN"),
		ExternalID: os.Getenv("ROLE_EXTERNAL_ID"),
	}
	if err := role.Validate(); err != nil {
		log.Errorf("Error while validating role: %v\n", err)
		return nil, err
	}

	sess, err := session.NewSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable})
	if err != nil {
		log.Errorf("Error creating base session: %v\n", err)
		return nil, err
	}
	sess, err = auth.AssumeCrossAccountRole(sess, role)
	if err != nil {
		log.Errorf("Error while attempting to assume cross account role: %v\n", err)
		return nil, err
	}

	log.Infof("Creating session using cross account role: %+v\n", role)
	return &Environment{
		EC2Client: svcProvider(sess),
		VolumeID:  os.Getenv("VOLUME_ID"),
	}, nil

}

// Validate validates the Environment.
func (e *Environment) Validate() error {
	if e.VolumeID == "" {
		return ErrInvalidVolumeID
	}
	return nil
}
