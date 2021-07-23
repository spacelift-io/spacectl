package cmd

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
