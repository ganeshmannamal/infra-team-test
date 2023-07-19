package api

import (
	"air-tek-iac/utils"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type WebApi struct {
	pulumi.ResourceState

	Url         pulumi.StringOutput `pulumi:"Url"`
	ApiEndpoint pulumi.StringOutput `pulumi:"ApiEndpoint"`
}

type WebApiArgs struct {
	NetworkName                string
	VpcId                      pulumi.StringInput
	LoadBalancerSubnets        pulumi.StringArrayInput
	LoadBalancerSecurityGroups pulumi.StringArrayInput
	Ec2SecurityGroups          pulumi.StringArrayInput
	Ec2Subnets                 pulumi.StringArrayInput
	EcsClusterArn              pulumi.StringInput
}

func NewWebApi(ctx *pulumi.Context, args *WebApiArgs, opts ...pulumi.ResourceOption) (*WebApi, error) {

	var resource WebApi

	config := config.New(ctx, "network")

	err := ctx.RegisterComponentResource("air-tek:infra:application", args.NetworkName+"-web-api", &resource, opts...)
	if err != nil {
		return nil, err
	}

	webApiLoadBalancer, err := utils.NewLoadBalancer(ctx, &utils.LoadBalancerArgs{
		LoadBalancerName: args.NetworkName + "-web-api-lb",
		VpcId:            args.VpcId,
		Subnets:          args.LoadBalancerSubnets,
		SecurityGroups:   args.LoadBalancerSecurityGroups,
		Internal:         true,
		HealthCheckPath:  "/WeatherForecast",
		HealthCheckPort:  "5000",
		ListenerPort:     5000,
		TargetPort:       5000,
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	resource.ApiEndpoint = webApiLoadBalancer.Url.ApplyT(func(url string) string {
		return "http://" + url + ":5000/WeatherForecast"
	}).(pulumi.StringOutput)

	registry, err := utils.NewECRRepository(ctx, args.NetworkName+"-web-api-ecr", pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	image, err := docker.NewImage(ctx, "web-api", &docker.ImageArgs{
		Build: docker.DockerBuildArgs{
			Context:    pulumi.String(".."),
			Dockerfile: pulumi.String("../infra-api/Dockerfile"),
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

	containerDef := image.ImageName.ApplyT(func(name string) (string, error) {
		fmtstr := `[{
			"name": "web-api",
			"image": %q,
			"portMappings": [{
				"containerPort": 5000,
				"hostPort": 5000,
				"protocol": "tcp"
			}]
		}]`
		return fmt.Sprintf(fmtstr, name), nil
	}).(pulumi.StringOutput)

	taskExecRole, err := iam.NewRole(ctx, args.NetworkName+"-web-api-task-exec-role", &iam.RoleArgs{
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
	_, err = iam.NewRolePolicyAttachment(ctx, args.NetworkName+"-web-api-task-exec-policy", &iam.RolePolicyAttachmentArgs{
		Role:      taskExecRole.Name,
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"),
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	webApiEcsTaskDefinition, err := ecs.NewTaskDefinition(ctx, args.NetworkName+"-web-api-ecs-task-def", &ecs.TaskDefinitionArgs{
		Family:                  pulumi.String("web-api-ecs-task-definition"),
		Cpu:                     pulumi.String("256"),
		Memory:                  pulumi.String("512"),
		NetworkMode:             pulumi.String("awsvpc"),
		RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
		ExecutionRoleArn:        taskExecRole.Arn,
		ContainerDefinitions:    containerDef,
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	_, err = ecs.NewService(ctx, args.NetworkName+"-web-api-ecs-service", &ecs.ServiceArgs{
		Cluster:        args.EcsClusterArn,
		DesiredCount:   pulumi.Int(1),
		LaunchType:     pulumi.String("FARGATE"),
		TaskDefinition: webApiEcsTaskDefinition.Arn,
		NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
			AssignPublicIp: pulumi.Bool(true),
			Subnets:        args.Ec2Subnets,
			SecurityGroups: args.Ec2SecurityGroups,
		},
		LoadBalancers: ecs.ServiceLoadBalancerArray{
			ecs.ServiceLoadBalancerArgs{
				TargetGroupArn: webApiLoadBalancer.TargetGroupArn,
				ContainerName:  pulumi.String("web-api"),
				ContainerPort:  pulumi.Int(5000),
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

	resource.Url = webApiLoadBalancer.Url

	return &resource, nil
}
