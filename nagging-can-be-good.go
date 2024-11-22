package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/cdklabs/cdk-nag-go/cdknag/v2"
	"os"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type NaggingCanBeGoodStackProps struct {
	awscdk.StackProps
	SshPubKey *string
}

func NewNaggingCanBeGoodStack(scope constructs.Construct, id string, props *NaggingCanBeGoodStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// ####### START VPC #######
	vpc := awsec2.NewVpc(stack, jsii.String("workshop-vpc"), &awsec2.VpcProps{
		NatGateways: jsii.Number(0),
		MaxAzs:      jsii.Number(1),
		SubnetConfiguration: &[]*awsec2.SubnetConfiguration{
			{
				CidrMask:            jsii.Number(24),
				Name:                jsii.String("workshop-public"),
				SubnetType:          awsec2.SubnetType_PUBLIC,
				MapPublicIpOnLaunch: jsii.Bool(true),
			},
		},
		FlowLogs: &map[string]*awsec2.FlowLogOptions{
			"cw": {
				Destination: awsec2.FlowLogDestination_ToCloudWatchLogs(nil, nil),
			},
		},
	})
	// ####### END VPC #######

	// ####### START SSH SG #######
	sshSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("workshop-ssh-sg"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		Description:      jsii.String("Workshop ssh security group"),
		AllowAllOutbound: jsii.Bool(true),
	})

	sshSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(22)),
		jsii.String("SSH from everywhere"),
		nil,
	)

	cdknag.NagSuppressions_AddResourceSuppressions(sshSecurityGroup, &[]*cdknag.NagPackSuppression{{
		Id:     jsii.String("AwsSolutions-EC23"),
		Reason: jsii.String("Can be open as workshop will be done in 1h."),
	}}, jsii.Bool(false))
	// ####### END SSH SG #######

	// ####### START EC2 ROLE #######
	ec2Role := awsiam.NewRole(stack, jsii.String("workshop-ec2-role"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"RetentionPolicy": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Resources: &[]*string{jsii.String("*")},
						Actions:   &[]*string{jsii.String("logs:PutRetentionPolicy")},
					}),
				},
			}),
		},
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("CloudWatchAgentServerPolicy")),
		},
	})

	cdknag.NagSuppressions_AddResourceSuppressions(ec2Role, &[]*cdknag.NagPackSuppression{{
		Id:     jsii.String("AwsSolutions-IAM4"),
		Reason: jsii.String("It is ok for our workshop"),
	}, {
		Id:        jsii.String("AwsSolutions-IAM5"),
		Reason:    jsii.String("It is ok for our workshop"),
		AppliesTo: &[]interface{}{"Resource::*"},
	}}, jsii.Bool(false))
	// ####### END EC2 ROLE #######

	// ####### START APP SG #######
	appSecurityGroup := awsec2.NewSecurityGroup(stack, jsii.String("workshop-app-sg"), &awsec2.SecurityGroupProps{
		Vpc:              vpc,
		Description:      jsii.String("Workshop app security group"),
		AllowAllOutbound: jsii.Bool(true),
	})

	appSecurityGroup.AddIngressRule(
		awsec2.Peer_AnyIpv4(),
		awsec2.Port_Tcp(jsii.Number(8085)),
		jsii.String("Our APP will be running on this port"),
		nil,
	)
	// ####### END APP SG #######

	cdknag.NagSuppressions_AddResourceSuppressions(appSecurityGroup, &[]*cdknag.NagPackSuppression{{
		Id:     jsii.String("AwsSolutions-EC23"),
		Reason: jsii.String("We need this open to access the app."),
	}}, jsii.Bool(false))

	// ####### START EC2 INSTANCE ######
	userData := awsec2.UserData_ForLinux(nil)

	instance := awsec2.NewInstance(stack, jsii.String("workshop-ec2-instance"), &awsec2.InstanceProps{
		Vpc:          vpc,
		InstanceType: awsec2.InstanceType_Of(awsec2.InstanceClass_BURSTABLE3, awsec2.InstanceSize_MEDIUM),
		MachineImage: awsec2.NewGenericLinuxImage(&map[string]*string{
			"eu-central-1": jsii.String("ami-0cee4a3eca5195216"),
		}, nil),
		SecurityGroup: appSecurityGroup,
		Role:          ec2Role,
		UserData:      userData,
		BlockDevices: &[]*awsec2.BlockDevice{
			{
				DeviceName: jsii.String("/dev/xvdh"),
				Volume: awsec2.BlockDeviceVolume_Ebs(jsii.Number(8), &awsec2.EbsDeviceOptions{
					Encrypted: jsii.Bool(true),
				}),
			},
		},
	})

	cdknag.NagSuppressions_AddResourceSuppressions(instance, &[]*cdknag.NagPackSuppression{{
		Id:     jsii.String("AwsSolutions-EC28"),
		Reason: jsii.String("Basic monitoring is enough for this workshop"),
	}, {
		Id:     jsii.String("AwsSolutions-EC29"),
		Reason: jsii.String("INo termination protection needed for this workshop"),
	}}, jsii.Bool(false))

	userData.AddCommands(
		jsii.String("echo \"" + *props.SshPubKey + "\" >> /home/ubuntu/.ssh/authorized_keys"),
	)

	instance.AddSecurityGroup(sshSecurityGroup)

	awscdk.NewCfnOutput(stack, jsii.String("ssh-command"), &awscdk.CfnOutputProps{
		Value: jsii.String("ssh -i live ubuntu@" + *instance.InstancePublicDnsName()),
	})
	// ####### END EC2 INSTANCE ######

	return stack
}

func main() {
	defer jsii.Close()
	sshPubKey := os.Getenv("SSH_PUB_KEY")

	app := awscdk.NewApp(nil)
	awscdk.Aspects_Of(app).Add(cdknag.NewAwsSolutionsChecks(nil))
	NewNaggingCanBeGoodStack(app, "NaggingCanBeGoodStack", &NaggingCanBeGoodStackProps{
		awscdk.StackProps{
			Env: env(),
		},
		jsii.String(sshPubKey),
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
