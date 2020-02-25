package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-xray-sdk-go/xray"
	log "github.com/sirupsen/logrus"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/auth"
	"github.com/swoldemi/scheduled-ebs-snapshots/pkg/lib"
)

func main() {
	log.Info("Starting Lambda in live environment")
	sess, err := auth.NewSessionFromEnvironment()
	if err != nil {
		log.Fatalf("Error while attempting to assume cross account role: %v\n", err)
		return
	}

	ec2Svc := ec2.New(sess)
	cwSvc := cloudwatch.New(sess)
	if err := xray.Configure(xray.Config{LogLevel: "trace"}); err != nil {
		log.Fatalf("Error configuring X-Ray: %v\n", err)
		return
	}

	xray.AWS(ec2Svc.Client)
	xray.AWS(cwSvc.Client)
	log.Info("Enabled request tracing on EC2 and CloudWatch API clients")

	f := lib.NewFunctionContainer(ec2Svc, cwSvc, lib.Production)
	h, err := f.GetHandler()
	if err != nil {
		log.Fatalf("Error initializing Lambda environment: %v\n", err)
		return
	}
	lambda.Start(h)
}
