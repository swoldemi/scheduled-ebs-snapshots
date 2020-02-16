// Package auth defines authN/authZ types and methods.
package auth

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrInvalidRole is returned when the request does not conatin a role.
	ErrInvalidRole = errors.New("must provide cross account role ARN and external ID")

	// ErrInvalidRoleARN is returned when the request contains a role with no role ARN.
	ErrInvalidRoleARN = errors.New("must provide cross account role ARN")

	// ErrInvalidRoleExternalID is returned when the request contains a role with no external ID.
	ErrInvalidRoleExternalID = errors.New("must provide cross account role's external ID")
)

// CrossAccountRole defines the ARN and external ID used to
// authenticate with a customer's account.
type CrossAccountRole struct {
	ARN        string `json:"arn"`
	ExternalID string `json:"external_id"`
}

// Validate validates a cross account role.
func (c *CrossAccountRole) Validate() error {
	if c.ARN == "" {
		return ErrInvalidRoleARN
	}
	if !arn.IsARN(c.ARN) {
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
	log.Infof("Retrieving credentials for role assumption: %s\n", role.ARN)

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
