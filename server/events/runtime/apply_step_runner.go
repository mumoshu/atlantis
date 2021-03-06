package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudposse/atlantis/server/events/models"
	"github.com/hashicorp/go-version"
)

// ApplyStepRunner runs `terraform apply`.
type ApplyStepRunner struct {
	TerraformExecutor TerraformExec
}

func (a *ApplyStepRunner) Run(ctx models.ProjectCommandContext, extraArgs []string, path string) (string, error) {
	planPath := filepath.Join(path, GetPlanFilename(ctx.Workspace, ctx.ProjectConfig))
	stat, err := os.Stat(planPath)
	if err != nil || stat.IsDir() {
		return "", fmt.Errorf("no plan found at path %q and workspace %q – did you run plan?", ctx.RepoRelDir, ctx.Workspace)
	}

	// NOTE: we need to quote the plan path because Bitbucket Server can
	// have spaces in its repo owner names which is part of the path.
	tfApplyCmd := append(append(append([]string{"apply", "-input=false", "-no-color"}, extraArgs...), ctx.CommentArgs...), fmt.Sprintf("%q", planPath))
	var tfVersion *version.Version
	if ctx.ProjectConfig != nil && ctx.ProjectConfig.TerraformVersion != nil {
		tfVersion = ctx.ProjectConfig.TerraformVersion
	}
	out, tfErr := a.TerraformExecutor.RunCommandWithVersion(ctx.Log, path, tfApplyCmd, tfVersion, ctx.Workspace)

	if tfErr == nil {
		ctx.Log.Info("apply successful")
	}
	return out, tfErr
}
