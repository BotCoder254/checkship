package app

import (
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

// Group for deployment to prevent multiple deployment at the same time on the same
// environment.
func DeploymentGroup(config domain.ConfigSnapshot) string {
	return EnvironmentGroup(config.ProjectName())
}

// Same group as DeploymentGroup from a compose project name. Volume backups and
// restores use it so they never run during a deployment of the same environment.
func EnvironmentGroup(projectName string) string {
	return "deployment.deployment.deploy." + projectName
}

// Group for target operation to prevent multiple target configuration at the same time.
func TargetConfigurationGroup(id domain.TargetID) string {
	return "deployment.target.configure." + string(id)
}
