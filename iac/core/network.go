package core

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type Network struct {
	pulumi.ResourceState

	NetworkName                       string              `pulumi:"NetworkName"`
	VpcId                             pulumi.StringOutput `pulumi:"VpcId"`
	PublicSubnet1aId                  pulumi.StringOutput `pulumi:"PublicSubnet1aId"`
	PublicSubnet1bId                  pulumi.StringOutput `pulumi:"PublicSubnet1bId"`
	PrivateSubnet1aId                 pulumi.StringOutput `pulumi:"PrivateSubnet1aId"`
	PrivateSubnet1bId                 pulumi.StringOutput `pulumi:"PrivateSubnet1bId"`
	WebUiEc2InstanceSecurityGroupId   pulumi.StringOutput `pulumi:"WebUiEc2InstanceSecurityGroupId"`
	WebUiLoadBalancerSecurityGroupId  pulumi.StringOutput `pulumi:"WebUiLoadBalancerSecurityGroupId"`
	WebApiEc2InstanceSecurityGroupId  pulumi.StringOutput `pulumi:"WebAPiEc2InstanceSecurityGroupId"`
	WebApiLoadBalancerSecurityGroupId pulumi.StringOutput `pulumi:"WebApiLoadBalancerSecurityGroupId"`
}

func NewNetwork(ctx *pulumi.Context, opts ...pulumi.ResourceOption) (*Network, error) {

	var resource Network

	config := config.New(ctx, "network")

	networkName := config.Require("name")

	err := ctx.RegisterComponentResource("air-tek:infra:network", networkName, &resource, opts...)
	if err != nil {
		return nil, err
	}

	vpc, err := ec2.NewVpc(ctx, networkName+"-vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String(config.Require("vpcRange")),
		EnableDnsSupport:   pulumi.Bool(true),
		EnableDnsHostnames: pulumi.Bool(true),
		InstanceTenancy:    pulumi.String("default"),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))

	if err != nil {
		return nil, err
	}

	availabilityZones, err := aws.GetAvailabilityZones(ctx, nil)
	if err != nil {
		return nil, err
	}

	publicSubnet1a, err := ec2.NewSubnet(ctx, networkName+"-public-subnet-1a", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.1.1.0/24"),
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String(availabilityZones.Names[0]),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	publicSubnet1b, err := ec2.NewSubnet(ctx, networkName+"-public-subnet-1b", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.1.2.0/24"),
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String(availabilityZones.Names[1]),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	privateSubnet1a, err := ec2.NewSubnet(ctx, networkName+"-private-subnet-1a", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.1.11.0/24"),
		MapPublicIpOnLaunch: pulumi.Bool(false),
		AvailabilityZone:    pulumi.String(availabilityZones.Names[0]),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	privateSubnet1b, err := ec2.NewSubnet(ctx, networkName+"-private-subnet-1b", &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String("10.1.12.0/24"),
		MapPublicIpOnLaunch: pulumi.Bool(false),
		AvailabilityZone:    pulumi.String(availabilityZones.Names[1]),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	igw, err := ec2.NewInternetGateway(ctx, networkName+"-igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	publicRouteTable, err := ec2.NewRouteTable(ctx, networkName+"-public-route-table", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	_, err = ec2.NewRouteTableAssociation(ctx, networkName+"-public-subnet-1a-route-table-association", &ec2.RouteTableAssociationArgs{
		SubnetId:     publicSubnet1a.ID(),
		RouteTableId: publicRouteTable.ID(),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, networkName+"-public-subnet-1b-route-table-association", &ec2.RouteTableAssociationArgs{
		SubnetId:     publicSubnet1b.ID(),
		RouteTableId: publicRouteTable.ID(),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	natGatewayEip, err := ec2.NewEip(ctx, networkName+"-nat-gw-eip", &ec2.EipArgs{
		Vpc: pulumi.Bool(true),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	natGateway, err := ec2.NewNatGateway(ctx, networkName+"-nat-gw", &ec2.NatGatewayArgs{
		AllocationId: natGatewayEip.ID(),
		SubnetId:     publicSubnet1a.ID(),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	natGwRouteTable, err := ec2.NewRouteTable(ctx, networkName+"-nat-gateway-route-table", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock:    pulumi.String("0.0.0.0/0"),
				NatGatewayId: natGateway.ID(),
			},
		},
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, networkName+"-private-subnet-1a-route-table-association", &ec2.RouteTableAssociationArgs{
		SubnetId:     privateSubnet1a.ID(),
		RouteTableId: natGwRouteTable.ID(),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	_, err = ec2.NewRouteTableAssociation(ctx, networkName+"-private-subnet-1b-route-table-association", &ec2.RouteTableAssociationArgs{
		SubnetId:     privateSubnet1b.ID(),
		RouteTableId: natGwRouteTable.ID(),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webUiLoadBalancerSecurityGroup, err := ec2.NewSecurityGroup(ctx, networkName+"-web-ui-loadbalancer-security-group", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("HTTPS"),
				FromPort:    pulumi.Int(443),
				ToPort:      pulumi.Int(443),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("HTTP"),
				FromPort:    pulumi.Int(80),
				ToPort:      pulumi.Int(80),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Tags: &pulumi.StringMap{
			"Name":            pulumi.String("elb allow http,https,egress"),
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webUiEc2InstanceSecurityGroup, err := ec2.NewSecurityGroup(ctx, networkName+"-web-ui-ec2-instance-security-group", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("http"),
				FromPort:    pulumi.Int(5000),
				ToPort:      pulumi.Int(5000),
				Protocol:    pulumi.String("tcp"),
				SecurityGroups: pulumi.StringArray{
					webUiLoadBalancerSecurityGroup.ID(),
				},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				FromPort: pulumi.Int(0),
				ToPort:   pulumi.Int(0),
				Protocol: pulumi.String("-1"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Tags: pulumi.StringMap{
			"Name":            pulumi.String("allow web ui elb"),
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webApiLoadBalancerSecurityGroup, err := ec2.NewSecurityGroup(ctx, networkName+"-web-api-loadbalancer-security-group", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("HTTP"),
				FromPort:    pulumi.Int(5000),
				ToPort:      pulumi.Int(5000),
				Protocol:    pulumi.String("tcp"),
				SecurityGroups: pulumi.StringArray{
					webUiEc2InstanceSecurityGroup.ID(),
				},
			},
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("tcp"),
				FromPort:    pulumi.Int(5000),
				ToPort:      pulumi.Int(5000),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Tags: pulumi.StringMap{
			"Name":            pulumi.String("elb web ui ec2"),
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webApiEc2InstanceSecurityGroup, err := ec2.NewSecurityGroup(ctx, networkName+"-web-api-ec2-instance-security-group", &ec2.SecurityGroupArgs{
		VpcId: vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("http"),
				FromPort:    pulumi.Int(5000),
				ToPort:      pulumi.Int(5000),
				Protocol:    pulumi.String("tcp"),
				SecurityGroups: pulumi.StringArray{
					webApiLoadBalancerSecurityGroup.ID(),
				},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				FromPort: pulumi.Int(0),
				ToPort:   pulumi.Int(0),
				Protocol: pulumi.String("-1"),
				CidrBlocks: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
			},
		},
		Tags: pulumi.StringMap{
			"Name":            pulumi.String("allow web api elb"),
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.NetworkName = networkName
	resource.VpcId = vpc.ID().ToStringOutput()
	resource.PublicSubnet1aId = publicSubnet1a.ID().ToStringOutput()
	resource.PublicSubnet1bId = publicSubnet1b.ID().ToStringOutput()
	resource.PrivateSubnet1aId = privateSubnet1a.ID().ToStringOutput()
	resource.PrivateSubnet1bId = privateSubnet1b.ID().ToStringOutput()
	resource.WebUiLoadBalancerSecurityGroupId = webUiLoadBalancerSecurityGroup.ID().ToStringOutput()
	resource.WebUiEc2InstanceSecurityGroupId = webUiEc2InstanceSecurityGroup.ID().ToStringOutput()
	resource.WebApiLoadBalancerSecurityGroupId = webApiLoadBalancerSecurityGroup.ID().ToStringOutput()
	resource.WebApiEc2InstanceSecurityGroupId = webApiEc2InstanceSecurityGroup.ID().ToStringOutput()

	ctx.RegisterResourceOutputs(&resource, pulumi.Map{
		"VpcId":                             vpc.ID(),
		"PrivateSubnet1aId":                 privateSubnet1a.ID(),
		"PrivateSubnet1bId":                 privateSubnet1b.ID(),
		"PublicSubnet1aId":                  publicSubnet1a.ID(),
		"PublicSubnet1bId":                  publicSubnet1b.ID(),
		"WebUiLoadBalancerSecurityGroupId":  webUiLoadBalancerSecurityGroup.ID(),
		"WebUiEc2InstanceSecurityGroupId":   webUiEc2InstanceSecurityGroup.ID(),
		"WebApiLoadBalancerSecurityGroupId": webApiLoadBalancerSecurityGroup.ID(),
		"WebApiEc2InstanceSecurityGroupId":  webApiEc2InstanceSecurityGroup.ID(),
	})

	return &resource, nil
}
