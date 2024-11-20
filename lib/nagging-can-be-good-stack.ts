import * as cdk from 'aws-cdk-lib';
import {Construct} from 'constructs';
import {CfnOutput, StackProps} from "aws-cdk-lib";
import {
    BlockDeviceVolume, FlowLogDestination, Instance,
    InstanceClass,
    InstanceSize, InstanceType, MachineImage,
    Peer,
    Port,
    SecurityGroup,
    SubnetType, UserData,
    Vpc
} from "aws-cdk-lib/aws-ec2";
import {ManagedPolicy, PolicyDocument, PolicyStatement, Role, ServicePrincipal} from "aws-cdk-lib/aws-iam";
import {NagSuppressions} from "cdk-nag";

export interface WorkshopProps extends StackProps {
    sshPubKey: string;
}

export class NaggingCanBeGoodStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: WorkshopProps) {
        super(scope, id, props);

        // ####### START VPC #######
        const vpc = new Vpc(this, 'workshop-vpc', {
            natGateways: 0,
            maxAzs: 1,
            subnetConfiguration: [
                {
                    cidrMask: 24,
                    name: 'workshop-public',
                    subnetType: SubnetType.PUBLIC,
                    mapPublicIpOnLaunch: true,
                }
            ],
            flowLogs: {
                'cw': {
                    destination: FlowLogDestination.toCloudWatchLogs()
                }
            }
        });
        // ####### END VPC #######

        // ####### START SSH SG #######
        const sshSecurityGroup = new SecurityGroup(this, 'workshop-ssh-sg', {
            vpc: vpc,
            description: 'Workshop ssh security group',
            allowAllOutbound: true,
        })
        sshSecurityGroup.addIngressRule(
            Peer.anyIpv4(),
            Port.tcp(22),
            'SSH from everywhere'
        )

        NagSuppressions.addResourceSuppressions(sshSecurityGroup, [
            {id: 'AwsSolutions-EC23', reason: 'Can be open as workshop will be done in 1h.'}
        ]);
        // ####### END SSH SG #######

        // ####### START EC2 ROLE #######
        // This role will allow the instance to put log events to CloudWatch Logs
        const ec2Role = new Role(this, 'workshop-ec2-role', {
            assumedBy: new ServicePrincipal('ec2.amazonaws.com'),
            inlinePolicies: {
                ['RetentionPolicy']: new PolicyDocument({
                    statements: [
                        new PolicyStatement({
                            resources: ['*'],
                            actions: ['logs:PutRetentionPolicy'],
                        }),
                    ],
                }),
            },
            managedPolicies: [
                ManagedPolicy.fromAwsManagedPolicyName('CloudWatchAgentServerPolicy'),
            ],
        });

        NagSuppressions.addResourceSuppressions(ec2Role, [
            {
                id: 'AwsSolutions-IAM4',
                reason: 'It is ok for our workshop'
            },
            {
                id: 'AwsSolutions-IAM5',
                reason: 'It is ok for our workshop',
                appliesTo: ['Resource::*']
            }
        ])
        // ####### END EC2 ROLE #######

        // ####### START APP SG #######
        const appSecurityGroup = new SecurityGroup(this, 'workshop-app-sg', {
                vpc: vpc,
                description: "Workshop app security group",
                allowAllOutbound: true,
            }
        )
        appSecurityGroup.addIngressRule(
            Peer.anyIpv4(),
            Port.tcp(8085),
            "Our APP will be running on this port"
        );
        NagSuppressions.addResourceSuppressions(appSecurityGroup, [
            {id: 'AwsSolutions-EC23', reason: 'We need this open to access the app.'}
        ]);
        // ####### END APP SG #######

        // ####### START EC2 INSTANCE ######
        const ec2Instance = new Instance(this, 'workshop-ec2-instance', {
            vpc: vpc,
            instanceType: InstanceType.of(InstanceClass.BURSTABLE3, InstanceSize.MEDIUM),
            machineImage: MachineImage.genericLinux({
                "eu-central-1": 'ami-0cee4a3eca5195216',
            }, {}),
            securityGroup: appSecurityGroup,
            role: ec2Role,
            userData: UserData.forLinux(),
            blockDevices: [{
                deviceName: "/dev/xvdh",
                volume: BlockDeviceVolume.ebs(8, {
                    encrypted: true
                })
            }],
        })

        NagSuppressions.addResourceSuppressions(ec2Instance, [
            {id: 'AwsSolutions-EC28', reason: 'Basic monitoring is enough for this workshop'},
            {id: 'AwsSolutions-EC29', reason: 'No termination protection needed for this workshop'},
        ]);

        ec2Instance.userData.addCommands(
            `echo "${props.sshPubKey}" >> /home/ubuntu/.ssh/authorized_keys`,
        );

        ec2Instance.addSecurityGroup(sshSecurityGroup);

        new CfnOutput(this, 'ssh-command', {
            value: `ssh -i live ubuntu@${ec2Instance.instancePublicDnsName}`
        });
        // ####### END EC2 INSTANCE ######
    }
}