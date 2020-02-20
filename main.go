package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-xray-sdk-go/xray"
	log "github.com/sirupsen/logrus"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/auth"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/lib"
)

func main() {
	sess, err := auth.AssumeCrossAccountRoleFromEnvironment()
	if err != nil {
		log.Fatalf("Error while attempting to assume cross account role: %v\n", err)
		return
	}
	svc := ec2.New(sess)
	xray.AWS(svc.Client)

	h, err := lib.FunctionContainer(svc)
	if err != nil {
		log.Fatalf("Error initializing Lambda environment: %v\n", err)
		return
	}
	lambda.Start(h)
}
