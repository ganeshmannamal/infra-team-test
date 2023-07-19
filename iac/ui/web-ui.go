package ui

import (
	"air-tek-iac/utils"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-awsx/sdk/go/awsx/awsx"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/go/awsx/ecs"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type WebUi struct {
	pulumi.ResourceState

	Url pulumi.StringOutput `pulumi:"Url"`
}

type WebUiArgs struct {
	NetworkName                string
	VpcId                      pulumi.StringInput
	LoadBalancerSubnets        pulumi.StringArrayInput
	LoadBalancerSecurityGroups pulumi.StringArrayInput
	Ec2SecurityGroups          pulumi.StringArrayInput
	Ec2Subnets                 pulumi.StringArrayInput
	EcsClusterArn              pulumi.StringInput
	WebApiEndpoint             pulumi.StringInput
}

func NewWebUi(ctx *pulumi.Context, args *WebUiArgs, opts ...pulumi.ResourceOption) (*WebUi, error) {

	var resource WebUi

	err := ctx.RegisterComponentResource("air-tek:infra:application", args.NetworkName+"-web-ui", &resource, opts...)
	if err != nil {
		return nil, err
	}

	webUiLoadBalancer, err := utils.NewLoadBalancer(ctx, &utils.LoadBalancerArgs{
		LoadBalancerName: args.NetworkName + "-web-ui-lb",
		VpcId:            args.VpcId,
		Subnets:          args.LoadBalancerSubnets,
		SecurityGroups:   args.LoadBalancerSecurityGroups,
		ListenerPort:     80,
		TargetPort:       5000,
	}, pulumi.Parent(&resource))

	registry, err := utils.NewECRRepository(ctx, args.NetworkName+"-web-ui-ecr", pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	image, err := docker.NewImage(ctx, "web-ui", &docker.ImageArgs{
		Build: docker.DockerBuildArgs{
			Context:    pulumi.String(".."),
			Dockerfile: pulumi.String("../infra-web/Dockerfile"),
			Platform:   pulumi.String("linux/amd64"),
		},
		ImageName: registry.RepositoryUrl,
		Registry: docker.RegistryArgs{
			Server:   registry.RepositoryUrl,
			Username: registry.User,
			Password: registry.Pass,
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	taskExecRole, err := iam.NewRole(ctx, args.NetworkName+"-web-ui-task-exec-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(
			`{
"Version": "2008-10-17",
"Statement": [{
	"Sid": "",
	"Effect": "Allow",
	"Principal": {
		"Service": "ecs-tasks.amazonaws.com"
	},
	"Action": "sts:AssumeRole"
}]
}`,
		),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}
	_, err = iam.NewRolePolicyAttachment(ctx, args.NetworkName+"-web-ui-task-exec-policy", &iam.RolePolicyAttachmentArgs{
		Role:      taskExecRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webUiEcsTaskDefinition, err := ecsx.NewEC2TaskDefinition(ctx, args.NetworkName+"-web-ui-ecs-task-def", &ecsx.EC2TaskDefinitionArgs{
		Family:      pulumi.String("web-ui-ecs-task-definition"),
		Cpu:         pulumi.String("256"),
		Memory:      pulumi.String("512"),
		NetworkMode: pulumi.String("awsvpc"),
		ExecutionRole: &awsx.DefaultRoleWithPolicyArgs{
			RoleArn: taskExecRole.Arn,
		},
		Containers: map[string]ecsx.TaskDefinitionContainerDefinitionArgs{
			"web-ui": {
				Name:  pulumi.String("web-ui"),
				Image: image.ImageName,
				PortMappings: &ecsx.TaskDefinitionPortMappingArray{
					&ecsx.TaskDefinitionPortMappingArgs{
						ContainerPort: pulumi.Int(5000),
						HostPort:      pulumi.Int(5000),
						Protocol:      pulumi.String("tcp"),
					},
				},
				Environment: &ecsx.TaskDefinitionKeyValuePairArray{
					&ecsx.TaskDefinitionKeyValuePairArgs{
						Name:  pulumi.String("ApiAddress"),
						Value: args.WebApiEndpoint,
					},
				},
			},
		},
	}, pulumi.Parent(&resource))

	_, err = ecs.NewService(ctx, args.NetworkName+"-web-ui-ecs-service", &ecs.ServiceArgs{
		Cluster:        args.EcsClusterArn,
		DesiredCount:   pulumi.Int(1),
		LaunchType:     pulumi.String("FARGATE"),
		TaskDefinition: webUiEcsTaskDefinition.TaskDefinition.Arn(),
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.Bool(true),
			Subnets:        args.Ec2Subnets,
			SecurityGroups: args.Ec2SecurityGroups,
		},
		LoadBalancers: ecs.ServiceLoadBalancerArray{
			ecs.ServiceLoadBalancerArgs{
				TargetGroupArn: webUiLoadBalancer.TargetGroupArn,
				ContainerName:  pulumi.String("web-ui"),
				ContainerPort:  pulumi.Int(5000),
			},
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.Url = webUiLoadBalancer.Url
	return &resource, nil
}
