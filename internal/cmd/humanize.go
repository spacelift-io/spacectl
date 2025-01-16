package cmd

import (
	"time"
)

// HumanizeVCSProvider converts the GraphQL VCSProvider enum to a human readable string.
func HumanizeVCSProvider(provider string) string {
	switch provider {
	case "BITBUCKET_CLOUD":
		return "Bitbucket Cloud"
	case "BITBUCKET_DATACENTER":
		return "Bitbucket Datacenter"
	case "GITHUB":
		return "GitHub"
	case "GITLAB":
		return "GitLab"
	case "GITHUB_ENTERPRISE":
		return "GitHub Enterprise"
	case "SHOWCASE":
		return "Showcase"
	case "AZURE_DEVOPS":
		return "Azure DevOps"
	}

	return provider
}

// HumanizePolicyType converts the GraphQL PolicyType enum to a human readable string.
func HumanizePolicyType(policyType string) string {
	switch policyType {
	case "ACCESS":
		return "Access"
	case "LOGIN":
		return "Login"
	case "GIT_PUSH":
		return "Git push"
	case "INITIALIZATION":
		return "Initialization"
	case "PLAN":
		return "Plan"
	case "TASK":
		return "Task"
	case "TRIGGER":
		return "Trigger"
	}

	return policyType
}

// HumanizeGitHash shortens a Git hash to make it more readable.
func HumanizeGitHash(hash string) string {
	if len(hash) > 8 {
		return hash[0:8]
	}

	return hash
}

func HumanizeBlueprintState(state string) string {
	switch state {
	case "DRAFT":
		return "Draft"
	case "PUBLISHED":
		return "Published"
	}

	return state
}

func HumanizeAuditTrailResourceType(resourceType string) string {
	switch resourceType {
	case "ACCOUNT":
		return "Account"
	case "API_KEY":
		return "API Key"
	case "AWS_INTEGRATION":
		return "AWS Integration"
	case "AZURE_DEVOPS_REPO_INTEGRATION":
		return "Azure DevOps Repo Integration"
	case "AZURE_INTEGRATION":
		return "Azure Integration"
	case "BITBUCKET_CLOUD_INTEGRATION":
		return "Bitbucket Cloud Integration"
	case "BITBUCKET_DATACENTER_INTEGRATION":
		return "Bitbucket Datacenter Integration"
	case "BLUEPRINT":
		return "Blueprint"
	case "CONTEXT":
		return "Context"
	case "DRIFT_DETECTION_INTEGRATION":
		return "Drift Detection Integration"
	case "GITHUB_APP_INSTALLATION":
		return "GitHub App Installation"
	case "GITHUB_ENTERPRISE_INTEGRATION":
		return "GitHub Enterprise Integration"
	case "GITLAB_INTEGRATION":
		return "GitLab Integration"
	case "GPG_KEY":
		return "GPG Key"
	case "LOGIN":
		return "Login"
	case "MODULE":
		return "Module"
	case "NAMED_WEBHOOKS_INTEGRATION":
		return "Named Webhooks Integration"
	case "NOTIFICATION":
		return "Notification"
	case "POLICY":
		return "Policy"
	case "RUN":
		return "Run"
	case "SCHEDULED_DELETE":
		return "Scheduled Delete"
	case "SCHEDULED_RUN":
		return "Scheduled Run"
	case "SCHEDULED_TASK":
		return "Scheduled Task"
	case "SECURITY_KEY":
		return "Security Key"
	case "SESSION":
		return "Session"
	case "SPACE":
		return "Space"
	case "STACK":
		return "Stack"
	case "TASK":
		return "Task"
	case "TERRAFORM_PROVIDER":
		return "Terraform Provider"
	case "TERRAFORM_PROVIDER_VERSION":
		return "Terraform Provider Version"
	case "UNKNOWN":
		return "Unknown"
	case "USER":
		return "User"
	case "USER_GROUP":
		return "User Group"
	case "USER_GROUP_INTEGRATION":
		return "User Group Integration"
	case "VERSION":
		return "Version"
	case "WEBHOOK":
		return "Webhook"
	case "WORKER":
		return "Worker"
	case "WORKER_POOL":
		return "Worker Pool"
	case "GENERIC_FORM":
		return "Generic Form"
	default:
		return resourceType
	}
}

func HumanizeUnixSeconds(seconds int) string {
	return time.Unix(int64(seconds), 0).Format(time.RFC3339)
}
