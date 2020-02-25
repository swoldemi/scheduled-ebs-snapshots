// Package auth defines authN/authZ types and methods.
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrInvalidRoleARN is returned when the request contains a role with no role ARN.
	ErrInvalidRoleARN = errors.New("must provide cross account IAM role ARN")

	// ErrInvalidRoleExternalID is returned when the request contains a role with no external ID.
	ErrInvalidRoleExternalID = errors.New("must provide cross account role's external ID")
)

// CrossAccountRole defines the ARN and external ID used to
// authenticate with a customer's account.
type CrossAccountRole struct {
	ARN        string
	ExternalID string
}

// Validate validates a cross account role.
// Assumes the provided role ARN is not an empty string.
func (c *CrossAccountRole) Validate() error {
	role, err := arn.Parse(c.ARN)
	if err != nil {
		return err
	}
	if role.Resource != "role" {
		return ErrInvalidRoleARN
	}
	if c.ExternalID == "" {
		return ErrInvalidRoleExternalID
	}
	return nil
}

// AssumeCrossAccountRole returns a session containing credentials from
// a cross-account that is assumed.
func AssumeCrossAccountRole(sess *session.Session, role *CrossAccountRole) (*session.Session, error) {
	log.Infof("Retrieving credentials for role assumption with role: %s\n", role.ARN)

	creds := stscreds.NewCredentials(sess, role.ARN, func(p *stscreds.AssumeRoleProvider) {
		p.Duration = 1 * time.Hour
		p.ExpiryWindow = 15 * time.Minute
		p.ExternalID = aws.String(role.ExternalID)
		p.RoleSessionName = "scheduled-ebs-snapshots"
	})

	log.Info("Returning session using credentials")
	return session.NewSessionWithOptions(
		session.Options{
			Config: aws.Config{
				Credentials:                   creds,
				CredentialsChainVerboseErrors: aws.Bool(true),
			},
			SharedConfigState: session.SharedConfigEnable,
		},
	)
}

// NewSessionFromEnvironment is a helper for creating a session using
// configuration stored within the environment.
func NewSessionFromEnvironment() (*session.Session, error) {
	region := os.Getenv("REGION")
	if region == "" {
		region = "us-east-1"
	}
	log.Infof("Creating session within region: %s\n", region)
	sess, err := session.NewSessionWithOptions(
		session.Options{
			Config: aws.Config{
				Region: aws.String(region),
			},
			SharedConfigState: session.SharedConfigEnable,
		},
	)
	if err != nil {
		log.Errorf("Error creating base session: %v\n", err)
		return nil, err
	}

	role := &CrossAccountRole{
		ARN:        os.Getenv("ROLE_ARN"),
		ExternalID: os.Getenv("ROLE_EXTERNAL_ID"),
	}
	if role.ARN != "" {
		if err := role.Validate(); err != nil {
			log.Errorf("Error while validating role: %v\n", err)
			return nil, err
		}
		return AssumeCrossAccountRole(sess, role)
	}
	return sess, nil
}
