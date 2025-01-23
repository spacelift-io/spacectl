package stack

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
	"github.com/shurcooL/graphql"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
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

type detachContextMutation struct {
	ContextAttach struct {
		ID string `graphql:"id"`
	} `graphql:"contextDetach(id: $contextStackAttachmentID)"`
}

type addContextConfigMutext struct {
	ContextConfigAdd struct {
		ID string `graphql:"id"`
	} `graphql:"contextConfigAdd(context: $context, config: $config)"`
}

type stackInfoQuery struct {
	Stack *struct {
		ID    string `graphql:"id"`
		Name  string `graphql:"name"`
		Space string `graphql:"space"`
	} `graphql:"stack(id: $stackId)" json:"stacks,omitempty"`
}

type userInfoQuery struct {
	Viewer *struct {
		ID   string `graphql:"id" json:"id"`
		Name string `graphql:"name" json:"name"`
	}
}

func attachAwsSession(cliCtx *cli.Context) error {
	// Ensure we capture SIGINT, and cleanup properly
	ctx, cancel := signal.NotifyContext(cliCtx.Context, os.Interrupt)
	defer cancel()

	manager, err := session.UserProfileManager()
	if err != nil {
		return fmt.Errorf("could not access profile manager: %w", err)
	}

	profile := manager.Current()
	if profile == nil {
		return fmt.Errorf("no active spacectl profile")
	}

	var userInfo userInfoQuery
	if err := authenticated.Client.Query(ctx, &userInfo, map[string]interface{}{}); err != nil {
		return errors.New("failed to query user information: unauthorized")
	}

	minimumCredLifetime := cliCtx.Duration(flagMinimumCredentialLifetime.Name)

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

	if awsCreds.CanExpire && time.Until(awsCreds.Expires) < minimumCredLifetime {
		return fmt.Errorf("AWS credentials expire in %v which is less than the minimum %v", time.Until(awsCreds.Expires), minimumCredLifetime)
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
	err = authenticated.Client.Mutate(ctx, &stackLockMutation{}, map[string]interface{}{
		"stack": graphql.ID(stackID),
		"note":  graphql.String(cliCtx.String(flagStackLockNote.Name)),
	})
	if err != nil {
		return err
	}

	// Ensure the stack is unlocked
	defer func() {
		err := authenticated.Client.Mutate(cliCtx.Context, &stackUnlockMutation{}, map[string]interface{}{
			"stack": graphql.ID(stackID),
		})
		if err != nil {
			fmt.Printf("failed to unlock stack: %v\n", err)
		}
	}()

	contextName := generateRandomContextName(userInfo.Viewer.Name)
	fmt.Printf("Creating temporary context '%v'\n", contextName)

	var createMutation createContextMutation
	err = authenticated.Client.Mutate(ctx, &createMutation, map[string]interface{}{
		"name":  graphql.String(contextName),
		"desc":  graphql.String(fmt.Sprintf("Temporary AWS credential context")),
		"space": graphql.ID(stack.Space),
	})
	if err != nil {
		return errors.Wrap(err, "could not create temporary context")
	}

	defer func() {
		var err error
		fmt.Printf("Deleting temporary context '%v'\n", createMutation.ContextCreate.Name)
		if err = authenticated.Client.Mutate(cliCtx.Context, &deleteContextMutation{}, map[string]interface{}{
			"id": createMutation.ContextCreate.ID,
		}); err != nil {
			fmt.Printf("Could not delete temporary context '%v'\n", createMutation.ContextCreate.Name)
		}
	}()

	envMap := map[string]string{
		"wo_AWS_ACCESS_KEY_ID":     awsCreds.AccessKeyID,
		"wo_AWS_SECRET_ACCESS_KEY": awsCreds.SecretAccessKey,
		"wo_AWS_SESSION_TOKEN":     awsCreds.SessionToken,
		"wo_AWS_SECURITY_TOKEN":    "",
	}
	if cliCtx.Bool(flagIncludeReadOnly.Name) {
		fmt.Println("Applying AWS credential to Read and Write environment variables")
		envMap["ro_AWS_ACCESS_KEY_ID"] = awsCreds.AccessKeyID
		envMap["ro_AWS_SECRET_ACCESS_KEY"] = awsCreds.SecretAccessKey
		envMap["ro_AWS_SESSION_TOKEN"] = awsCreds.SessionToken
		envMap["ro_AWS_SECURITY_TOKEN"] = ""
	} else {
		fmt.Println("Applying AWS credential to Write-Only environment variables")
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

	var attachMutation attachContextMutation
	err = authenticated.Client.Mutate(ctx, &attachMutation, map[string]interface{}{
		"contextID": createMutation.ContextCreate.ID,
		"stackID":   stack.ID,
	})
	if err != nil {
		return errors.Wrap(err, "could not attach temporary credentials to stack")
	}
	defer func() {
		fmt.Printf("Detaching temporary context '%v'\n", createMutation.ContextCreate.Name)
		if err := authenticated.Client.Mutate(cliCtx.Context, &detachContextMutation{}, map[string]interface{}{
			"contextStackAttachmentID": attachMutation.ContextAttach.ID,
		},
		); err != nil {
			fmt.Printf("Could not detach temporary context '%v'\n", createMutation.ContextCreate.Name)
		}
	}()

	fmt.Printf("Temporary credentials attached to '%v'\n", stack.Name)

	// If the credentials can expire, we should automatically exit after they do. Otherwise,
	// we could have stale credentials hanging around in the context.
	if awsCreds.CanExpire {
		fmt.Printf("Temporary credentials will expire in %v (at %v)\n", time.Until(awsCreds.Expires), awsCreds.Expires.Local())
		ctx, cancel = context.WithDeadlineCause(
			ctx,
			awsCreds.Expires,
			errors.New("AWS credentials expired"),
		)
		defer cancel()
	}

	// Wait for the context to close (probably a SIGINT)
	fmt.Println("Press CTRL+C to quit and revert changes")
	<-ctx.Done()

	if err := context.Cause(ctx); err != nil {
		fmt.Printf("%v; shutting down...\n", err)
	}

	return nil
}

var letters []rune = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func generateRandomContextName(userName string) string {
	contextName := "tmp_creds_" + userName + "_"

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
