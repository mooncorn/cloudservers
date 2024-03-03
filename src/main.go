package main

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/instance", createEC2InstanceHandler)

	r.Run(":3000")
}

func createEC2InstanceHandler(context *gin.Context) {
	// Create a new AWS session using default credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create AWS session"})
		return
	}

	// Create an EC2 service client
	svc := ec2.New(sess)

	// Specify parameters for the instance
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String("ami-12345678"), // AMI ID of the instance
		InstanceType: aws.String("t2.micro"),     // Instance type
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
	}

	// Run the instance
	result, err := svc.RunInstances(runInput)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to launch instance"})
		return
	}

	// Extract instance ID
	instanceID := *result.Instances[0].InstanceId

	context.JSON(http.StatusOK, gin.H{"instanceID": instanceID})
}
