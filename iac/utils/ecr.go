package utils

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ECRRepository struct {
	pulumi.ResourceState

	RepositoryUrl pulumi.StringOutput `pulumi:"user"`
	User          pulumi.StringOutput `pulumi:"user"`
	Pass          pulumi.StringOutput `pulumi:"pass"`
}

func NewECRRepository(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*ECRRepository, error) {
	var resource ECRRepository

	config := config.New(ctx, "network")

	err := ctx.RegisterComponentResource("air-tek:infra:ecr", name, &resource, opts...)
	if err != nil {
		return nil, err
	}

	repo, err := ecr.NewRepository(ctx, name, &ecr.RepositoryArgs{
		ForceDelete: pulumi.Bool(true),
		Tags: &pulumi.StringMap{
			"air-tek:project": pulumi.String(ctx.Project()),
			"air-tek:stack":   pulumi.String(ctx.Stack()),
			"air-tek:network": pulumi.String(config.Require("name")),
		},
	}, pulumi.Parent(&resource))
	if err != nil {
		return nil, err
	}

	creds := repo.RegistryId.ApplyT(func(registryId string) ([]string, error) {
		creds, err := ecr.GetCredentials(ctx, &ecr.GetCredentialsArgs{
			RegistryId: registryId,
		})
		if err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(creds.AuthorizationToken)
		if err != nil {
			fmt.Println("error:", err)
			return nil, err
		}

		return strings.Split(string(data), ":"), nil
	}).(pulumi.StringArrayOutput)

	resource.RepositoryUrl = repo.RepositoryUrl
	resource.User = creds.Index(pulumi.Int(0))
	resource.Pass = creds.Index(pulumi.Int(1))

	return &resource, nil
}
