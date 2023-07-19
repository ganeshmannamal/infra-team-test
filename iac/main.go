package main

import (
	"air-tek-iac/api"
	"air-tek-iac/core"
	"air-tek-iac/ui"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		network, err := core.NewNetwork(ctx)

		if err != nil {
			return err
		}

		ecsCluster, err := ecs.NewCluster(ctx, network.NetworkName+"-ecs-cluster", nil)
		if err != nil {
			return err
		}

		webApi, err := api.NewWebApi(ctx, &api.WebApiArgs{
			NetworkName: network.NetworkName,
			VpcId:       network.VpcId,
			LoadBalancerSubnets: pulumi.StringArray{
				network.PrivateSubnet1aId,
				network.PrivateSubnet1bId,
			},
			LoadBalancerSecurityGroups: pulumi.StringArray{
				network.WebApiLoadBalancerSecurityGroupId,
			},
			Ec2Subnets: pulumi.StringArray{
				network.PrivateSubnet1aId,
				network.PrivateSubnet1bId,
			},
			Ec2SecurityGroups: pulumi.StringArray{
				network.WebApiEc2InstanceSecurityGroupId,
			},
			EcsClusterArn: ecsCluster.Arn,
		})

		webUi, err := ui.NewWebUi(ctx, &ui.WebUiArgs{
			NetworkName: network.NetworkName,
			VpcId:       network.VpcId,
			LoadBalancerSubnets: pulumi.StringArray{
				network.PublicSubnet1aId,
				network.PublicSubnet1bId,
			},
			LoadBalancerSecurityGroups: pulumi.StringArray{
				network.WebUiLoadBalancerSecurityGroupId,
			},
			Ec2Subnets: pulumi.StringArray{
				network.PrivateSubnet1aId,
				network.PrivateSubnet1bId,
			},
			Ec2SecurityGroups: pulumi.StringArray{
				network.WebUiEc2InstanceSecurityGroupId,
			},
			EcsClusterArn:  ecsCluster.Arn,
			WebApiEndpoint: webApi.ApiEndpoint,
		})

		ctx.Export("vpcId", network.VpcId)
		ctx.Export("web-api-url", webApi.Url)
		ctx.Export("web-ui-url", webUi.Url)

		return nil
	})
}
