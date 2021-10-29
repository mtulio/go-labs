#!/usr/bin/env python3
import os

from aws_cdk import core as cdk

# For consistency with TypeScript code, `cdk` is the preferred import name for
# the CDK's core module.  The following line also imports it as `core` for use
# with examples from the CDK Developer's Guide, which are in the process of
# being updated to use `cdk`.  You may delete this import if you don't need it.
#from aws_cdk import core

import os.path

from aws_cdk.aws_s3_assets import Asset

from aws_cdk import (
    aws_ec2 as ec2,
    aws_iam as iam,
    core as cdk,
    aws_elasticloadbalancingv2 as elbv2,
)
from aws_cdk.aws_ec2 import SubnetConfiguration, SubnetType


from deploy_stack.deploy_stack_stack import DeployStackStack

dirname = os.path.dirname(__file__)

class LabAppStack(cdk.Stack):
    def __init__(self, app: cdk.App, id: str, **kwargs):
        super().__init__(app, id, **kwargs)

        # Subnet configurations for a public and private tier
        # subnet1 = SubnetConfiguration(
        #         name="use1a-public",
        #         subnet_type=SubnetType.PUBLIC,
        #         cidr_mask=24)
        # subnet2 = SubnetConfiguration(
        #         name="use1b-public",
        #         subnet_type=SubnetType.PUBLIC,
        #         cidr_mask=24)

        vpc = ec2.Vpc(self, "VPC",
                cidr="10.0.0.0/16",
                  enable_dns_hostnames=True,
                  enable_dns_support=True,
                  #max_azs=2,
                  nat_gateways=0,
                  subnet_configuration=[
                    SubnetConfiguration(
                        name="public",
                        subnet_type=SubnetType.PUBLIC,
                        cidr_mask=24)
                ]
        )
        # ec2.Subnet(
        #     self, "use1a-public",
        #     availability_zone="us-east-1a",
        #     vpc_id=vpc.vpc_id,
        #     cidr_block="10.0.0.0/24",
        #     map_public_ip_on_launch=True,
        # )
        # ec2.Subnet(
        #     self, "use1b-public",
        #     availability_zone="us-east-1b",
        #     vpc_id=vpc.vpc_id,
        #     cidr_block="10.0.0.0/24",
        #     map_public_ip_on_launch=True,
        # )
        cdk.CfnOutput(self, "vpcid", value=vpc.vpc_id)

        subnets = ec2.SubnetSelection(subnet_type=SubnetType.PUBLIC, one_per_az=True)
        #ubnets = ec2.SubnetSelection(subnet_type=SubnetType.PUBLIC)
        #print(subnets)
        
        ##> IAM
        # Instance Role and SSM Managed Policy
        role = iam.Role(self, "lab-app-instance", assumed_by=iam.ServicePrincipal("ec2.amazonaws.com"))
        role.add_managed_policy(iam.ManagedPolicy.from_aws_managed_policy_name("AmazonEC2ReadOnlyAccess"))

        ##> EC2 SG
        ec2_sg = ec2.SecurityGroup(self, "sg-lab-app", vpc=vpc,
            allow_all_outbound=True,
            description='Lab App SG',
        )
        ec2_sg.add_ingress_rule(
            ec2.Peer.ipv4("0.0.0.0/0"),
            ec2.Port.tcp(22),
            'Allow SSH')
        ec2_sg.add_ingress_rule(
            ec2.Peer.ipv4("0.0.0.0/0"),
            ec2.Port.tcp(6443),
            'Allow Service Port')
        ec2_sg.add_ingress_rule(
            ec2.Peer.ipv4("0.0.0.0/0"),
            ec2.Port.tcp(6444),
            'Allow Service Port')

        ##> EC2
        with open("./user-data.sh") as f:
            user_data = f.read()

        amzn_linux = ec2.MachineImage.latest_amazon_linux(
            generation=ec2.AmazonLinuxGeneration.AMAZON_LINUX_2,
            edition=ec2.AmazonLinuxEdition.STANDARD,
            virtualization=ec2.AmazonLinuxVirt.HVM,
            storage=ec2.AmazonLinuxStorage.GENERAL_PURPOSE
            )

        # Instance
        ec2_server01 = ec2.Instance(self, "lab-app-01",
            instance_type=ec2.InstanceType("t3.micro"),
            machine_image=amzn_linux,
            vpc = vpc,
            role = role,
            #vpc_subnets=ec2.SubnetSelection(
            #        subnet_group_name="use1a-public"),
            user_data=ec2.UserData.custom(user_data),
            key_name="openshift-dev",
            security_group=ec2_sg,
        )

        ec2_server02 = ec2.Instance(self, "lab-app-02",
            instance_type=ec2.InstanceType("t3.micro"),
            machine_image=amzn_linux,
            vpc = vpc,
            role = role,
            #vpc_subnets=ec2.SubnetSelection(
            #        subnet_group_name="use1b-public"),
            user_data=ec2.UserData.custom(user_data),
            key_name="openshift-dev",
            security_group=ec2_sg,
        )

        lb = elbv2.NetworkLoadBalancer(
            self, "LB",
            vpc=vpc,
            internet_facing=True,
            cross_zone_enabled=True,
            vpc_subnets=subnets,
            )

        tg = elbv2.NetworkTargetGroup(self, "app-svc-tcp-hc-https",
            port=6443, protocol=elbv2.Protocol.TCP,
            targets=[
                elbv2.IpTarget(ec2_server01.instance_private_ip, port=6443),
                elbv2.IpTarget(ec2_server02.instance_private_ip, port=6443)
            ],
            health_check=elbv2.HealthCheck(
                healthy_threshold_count=2,
                interval=cdk.Duration.seconds(10),
                path="/readyz",
                port="6444",
                protocol=elbv2.Protocol.HTTPS,
                timeout=cdk.Duration.seconds(10),
                unhealthy_threshold_count=2,
            ),
            target_type=elbv2.TargetType.IP,
            vpc=vpc,
        )

        # create the default listener. ToDo fix it to support tg that is failing in register_listener()
        listener = lb.add_listener("Listener", port=6443)
        listener.add_targets("Target", port=6443, targets=[])
        
        # tg.register_listener(listener)


env = cdk.Environment(
    region="us-east-1",
)
app = cdk.App()
Stack = LabAppStack(app, "lab-app", env=env)

app.synth()
