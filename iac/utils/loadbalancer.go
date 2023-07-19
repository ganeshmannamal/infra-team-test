package utils

import (
	elb "github.com/pulumi/pulumi-aws/sdk/v5/go/aws/elasticloadbalancingv2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type LoadBalancer struct {
	pulumi.ResourceState

	Url            pulumi.StringOutput `pulumi:"Url"`
	TargetGroupArn pulumi.StringOutput `pulumi:"TargetGroupArn"`
}

type LoadBalancerArgs struct {
	LoadBalancerName string
	VpcId            pulumi.StringInput
	Subnets          pulumi.StringArrayInput
	SecurityGroups   pulumi.StringArrayInput
	ListenerPort     int
	TargetPort       int
	HealthCheckPath  string
	HealthCheckPort  string
	Internal         bool
}

func NewLoadBalancer(ctx *pulumi.Context, args *LoadBalancerArgs, opts ...pulumi.ResourceOption) (*LoadBalancer, error) {
	var resource LoadBalancer

	config := config.New(ctx, "network")

	err := ctx.RegisterComponentResource("air-tek:infra:loadbalancer", args.LoadBalancerName, &resource, opts...)
	if err != nil {
		return nil, err
	}

	alb, err := elb.NewLoadBalancer(ctx, args.LoadBalancerName, &elb.LoadBalancerArgs{
		// Subnets:        toPulumiStringArray(subnet.Ids),
		Subnets:        args.Subnets,
		SecurityGroups: args.SecurityGroups,
		Internal:       pulumi.Bool(args.Internal),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	targetGroup, err := elb.NewTargetGroup(ctx, args.LoadBalancerName+"-tg", &elb.TargetGroupArgs{
		Port:       pulumi.Int(args.TargetPort),
		Protocol:   pulumi.String("HTTP"),
		TargetType: pulumi.String("ip"),
		VpcId:      args.VpcId,
		HealthCheck: &elb.TargetGroupHealthCheckArgs{
			Path: pulumi.String(args.HealthCheckPath),
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

	_, err = elb.NewListener(ctx, args.LoadBalancerName+"-listener", &elb.ListenerArgs{
		LoadBalancerArn: alb.Arn,
		Port:            pulumi.Int(args.ListenerPort),
		DefaultActions: elb.ListenerDefaultActionArray{
			elb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: targetGroup.Arn,
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

	resource.Url = alb.DnsName
	resource.TargetGroupArn = targetGroup.Arn

	return &resource, nil
}
