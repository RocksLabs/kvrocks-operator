package util

import (
	"fmt"
	"os/exec"
)

func HelmTool(args ...string) ([]byte, error) {
	if output, err := exec.Command("helm", "version").CombinedOutput(); err != nil {
		return output, fmt.Errorf("helm is not installed or not in the PATH: %w, output: %s", err, string(output))
	}

	cmd := exec.Command("helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("helm command %s failed: %w, output: %s", cmd.String(), err, string(output))
	}
	return output, nil
}

func KubectlTool(args ...string) ([]byte, error) {
	if output, err := exec.Command("kubectl", "version").CombinedOutput(); err != nil {
		return output, fmt.Errorf("kubectl is not installed or not in the PATH: %w, output: %s", err, string(output))
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("kubectl command %s failed: %w, output: %s", cmd.String(), err, string(output))
	}
	return output, nil
}
