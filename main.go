package main

/***************************************************************************************************
ec2idtotag
機能: 指定したEC2インスタンスIDに割り当てられているtag keyよりtag valueを表示する(完全一致)

(1)IAM Role利用(自インスタンス稼働リージョン)
$ ./ec2idtotag --instance-id `curl -s http://169.254.169.254/latest/meta-data/instance-id`
vpc-XXXXXXXX
(2)credentials利用
$ ./ec2idtotag --instance-id i-XXXXXXXX -p <shared credentials> -r <region>
***************************************************************************************************/

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/codegangsta/cli"
)

func getTagValueFromInstanceID(svc *ec2.EC2, s string, tagKey string) {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []*string{aws.String(s)},
			},
		},
	}
	var tagValue string
	resp, err := svc.DescribeInstances(params)
	for idx := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			for _, tag := range inst.Tags {
				if *tag.Key == tagKey {
					tagValue = *tag.Value
				}
			}
		}
	}
	fmt.Println(tagValue)

	if err != nil {
		panic(err)
	}
	/*
		if resp.Reservations[0].Instances[0].VpcId != nil {
			fmt.Println(*resp.Reservations[0].Instances[0].VpcId)
		}
	*/
}

func getRegionFromInstanceMetaData() (region string) {
	metadata := ec2metadata.New(session.New())
	region, err := metadata.Region()
	if err != nil {
		panic(err)
	}
	return region
}

func main() {
	var instanceID, tagKey, region, profile string
	app := cli.NewApp()
	app.Name = "ec2idtotag"
	app.Usage = "EC2 Instance ID From Tag value"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "instance-id, i",
			Value:       "",
			Usage:       "--instance-id `http://169.254.169.254/latest/meta-data/instance-id`",
			Destination: &instanceID,
		},
		cli.StringFlag{
			Name:        "tagkey, t",
			Value:       "Name",
			Usage:       "--tagkey Name",
			Destination: &tagKey,
		},
		cli.StringFlag{
			Name:        "profile, p",
			Value:       "",
			Usage:       "--profile default",
			Destination: &profile,
		},
		cli.StringFlag{
			Name:        "region, r",
			Value:       "",
			Usage:       "--region ap-northeast-1",
			Destination: &region,
		},
	}
	app.Action = func(c *cli.Context) {
		var config *aws.Config
		if c.String("profile") == "" {
			if c.String("region") == "" {
				region = getRegionFromInstanceMetaData()
			}
			ec2m := ec2metadata.New(session.New(), &aws.Config{
				HTTPClient: &http.Client{Timeout: 10 * time.Second},
			})
			creds := credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
				Client: ec2m,
			})
			// IAM Role
			config = aws.NewConfig().WithCredentials(creds).WithRegion(region)
		} else {
			config = &aws.Config{
				Credentials: credentials.NewSharedCredentials("", profile),
				Region:      aws.String(region),
			}
		}

		if c.String("instance-id") != "" {
			getTagValueFromInstanceID(ec2.New(session.New(), config), instanceID, tagKey)
			os.Exit(0)
		}
	}
	app.Run(os.Args)
}
