package stack

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"os/user"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/internal/cmd/authenticated"
)

type createContextMutation struct {
	ContextCreate struct {
		ID   string `graphql:"id"`
		Name string `graphql:"name"`
	} `graphql:"contextCreate(name: $name, description: $desc, space: $space)"`
}

type deleteContextMutation struct {
	ContextDelete struct {
		Name string `graphql:"name"`
	} `graphql:"contextDelete(id: $id)"`
}

type attachContextMutation struct {
	ContextAttach struct {
		ID string `graphql:"id"`
	} `graphql:"contextAttach(id: $contextID, stack: $stackID, priority: 0)"`
}

type addContextConfigMutext struct {
	ContextConfigAdd struct {
		ID string `graphql:"id"`
	} `graphql:"contextConfigAdd(context: $context, config: $config)"`
}

type stackInfoQuery struct {
	Stack *struct {
		ID                      string `graphql:"id"`
		Name                    string `graphql:"name"`
		AttachedAWSIntegrations []struct {
			ID            string `graphql:"id"`
			IntegrationID string `graphql:"integrationId"`
			Name          string `graphql:"name"`
			Read          bool   `graphql:"read"`
			Write         bool   `graphql:"write"`
		} `graphql:"attachedAwsIntegrations"`
		Space string `graphql:"space"`
	} `graphql:"stack(id: $stackId)" json:"stacks,omitempty"`
}

func attachAwsSession(cliCtx *cli.Context) error {
	// Ensure we capture SIGINT, and cleanup properly
	ctx, cancel := signal.NotifyContext(cliCtx.Context, os.Interrupt)
	defer cancel()

	// Load the current AWS config or active profile
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "could not load AWS default configuration")
	}

	// Retrieve active credentials
	awsCreds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return errors.Wrap(err, "could not retrieve AWS credentials")
	}

	// Verify that the credentials haven't expired
	if awsCreds.Expired() {
		return errors.New("AWS credentials are expired")
	}

	// Ensure they have valid keys
	if !awsCreds.HasKeys() {
		return errors.New("AWS credentials do not have keys")
	}

	identityArn, err := getAwsIdentityArn(ctx, cfg)
	if err != nil {
		return errors.Wrap(err, "could not get AWS identity ARN")
	}

	// Get the stack ID from the arguments/flags or infer it
	stackID, err := getStackID(cliCtx)
	if err != nil {
		return err
	}

	// Lookup the stack details
	var stackQuery stackInfoQuery
	err = authenticated.Client.Query(ctx, &stackQuery, map[string]interface{}{
		"stackId": graphql.ID(stackID),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to query for stack ID %q", stackID)
	}

	// Extract the stack and ensure it exists
	stack := stackQuery.Stack
	if stack == nil {
		return fmt.Errorf("stack ID %q not found", stackID)
	}

	// Lock the stack
	err = lock(cliCtx)
	if err != nil {
		return err
	}

	// Ensure the stack is unlocked
	defer func() {
		err := unlock(cliCtx)
		if err != nil {
			fmt.Printf("failed to unlock stack: %v\n", err)
		}
	}()

	contextName := generateRandomContextName()
	fmt.Printf("Creating temporary context '%v' for '%v'\n", contextName, identityArn)

	var createMutation createContextMutation
	err = authenticated.Client.Mutate(ctx, &createMutation, map[string]interface{}{
		"name":  graphql.String(contextName),
		"desc":  graphql.String(fmt.Sprintf("Temporary AWS credential context for %v", identityArn)),
		"space": graphql.ID(stack.Space),
	})
	if err != nil {
		return errors.Wrap(err, "could not create temporary context")
	}

	defer func() {
		fmt.Printf("Deleting temporary context '%v'\n", createMutation.ContextCreate.Name)
		err := authenticated.Client.Mutate(cliCtx.Context, &deleteContextMutation{}, map[string]interface{}{
			"id": createMutation.ContextCreate.ID,
		})
		if err != nil {
			fmt.Printf("Could not delete temporary context '%v'\n", createMutation.ContextCreate.Name)
		}
	}()

	envMap := map[string]string{}
	if len(stack.AttachedAWSIntegrations) > 0 {
		// If there are AWS integrations, override the AWS integration envvars
		envMap = map[string]string{
			"ro_AWS_ACCESS_KEY_ID":     awsCreds.AccessKeyID,
			"ro_AWS_SECRET_ACCESS_KEY": awsCreds.SecretAccessKey,
			"ro_AWS_SESSION_TOKEN":     awsCreds.SessionToken,
			"ro_AWS_SECURITY_TOKEN":    "",
			"wo_AWS_ACCESS_KEY_ID":     awsCreds.AccessKeyID,
			"wo_AWS_SECRET_ACCESS_KEY": awsCreds.SecretAccessKey,
			"wo_AWS_SESSION_TOKEN":     awsCreds.SessionToken,
			"wo_AWS_SECURITY_TOKEN":    "",
		}
		fmt.Printf("Overriding Read/Write AWS Integration Credentials\n")
	} else {
		// No AWS integrations, so just override the regular AWS auth envvars
		envMap = map[string]string{
			"AWS_ACCESS_KEY_ID":     awsCreds.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": awsCreds.SecretAccessKey,
			"AWS_SESSION_TOKEN":     awsCreds.SessionToken,
			"AWS_SECURITY_TOKEN":    "",
		}
		fmt.Printf("Overriding AWS Credentials\n")
	}

	for key, value := range envMap {
		err := authenticated.Client.Mutate(ctx, &addContextConfigMutext{}, map[string]interface{}{
			"context": graphql.ID(createMutation.ContextCreate.ID),
			"config": ConfigInput{
				ID:        key,
				Type:      envVarTypeConfig,
				Value:     graphql.String(value),
				WriteOnly: true,
			},
		})
		if err != nil {
			return errors.Wrapf(err, "could not add config '%v' to temporary context", key)
		}
	}

	// Attach temporary credentials to the stack
	fmt.Printf("Attaching temporary context to '%v' stack '%v'\n", contextName, stack.Name)
	err = authenticated.Client.Mutate(ctx, &attachContextMutation{}, map[string]interface{}{
		"contextID": createMutation.ContextCreate.ID,
		"stackID":   stack.ID,
	})
	if err != nil {
		return errors.Wrap(err, "could not attach temporary credentials to stack")
	}

	// Wait for the context to close (probably a SIGINT)
	fmt.Printf("Temporary credentials attached to '%v'. Press CTRL+C to revert changes..\n", stack.Name)
	<-ctx.Done()

	// Show the reason for exit
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		fmt.Println("Context canceled; shutting down...")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		fmt.Println("Context deadline exceeded; shutting down...")
	default:
		fmt.Printf("Unknown context error: %v\n", ctx.Err())
	}

	return nil
}

var letters []rune = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func generateRandomContextName() string {
	contextName := ""

	info, err := user.Current()
	if err == nil {
		contextName += info.Username + "_"
	}

	hostname, err := os.Hostname()
	if err == nil {
		contextName += hostname + "_"
	}

	contextName += "tmp_creds_"

	for i := 0; i < 8; i++ {
		contextName += string(letters[rand.Intn(len(letters))])
	}

	return contextName
}

func getAwsIdentityArn(ctx context.Context, cfg aws.Config) (string, error) {
	client := sts.NewFromConfig(cfg)

	output, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *output.Arn, nil
}
