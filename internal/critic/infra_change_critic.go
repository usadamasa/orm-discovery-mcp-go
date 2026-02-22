package critic

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// InfraChangeCritic detects changes to infrastructure files and generates warnings.
type InfraChangeCritic struct{}

// NewInfraChangeCritic creates a new InfraChangeCritic.
func NewInfraChangeCritic() *InfraChangeCritic {
	return &InfraChangeCritic{}
}

// Name returns the name of this critic.
func (c *InfraChangeCritic) Name() string {
	return "InfraChangeCritic"
}

// Review checks for infrastructure file changes and generates warning findings.
// Added, Modified, and Deleted infra files produce warnings. Renamed files are skipped.
func (c *InfraChangeCritic) Review(_ context.Context, input ReviewInput) ([]model.Finding, error) {
	if len(input.ClassifiedFiles) == 0 {
		return nil, nil
	}

	var findings []model.Finding
	for _, f := range input.ClassifiedFiles {
		if f.Category != git.FileCategoryInfra {
			continue
		}
		if f.Status == git.FileStatusRenamed {
			continue
		}

		infraType := classifyInfraType(f.Path)
		msg := fmt.Sprintf("%s file changed: %s (%s)", infraType, filepath.Base(f.Path), f.Status)

		finding := model.NewFinding(
			model.SeverityWarning,
			model.CategoryInfraChange,
			msg,
			model.Location{FilePath: f.Path},
		)
		finding.CriticName = c.Name()
		finding.Suggestion = fmt.Sprintf("review %s change carefully for security and operational impact", infraType)
		findings = append(findings, finding)
	}

	return findings, nil
}

// classifyInfraType returns a human-readable infrastructure type for a file path.
func classifyInfraType(path string) string {
	base := filepath.Base(path)

	if strings.HasPrefix(base, "docker-compose") {
		return "Docker Compose"
	}
	if strings.HasPrefix(base, "Dockerfile") {
		return "Dockerfile"
	}
	if filepath.Ext(path) == ".tf" {
		return "Terraform"
	}
	normalized := filepath.ToSlash(path)
	if strings.HasPrefix(normalized, ".github/workflows/") {
		return "CI/CD"
	}
	parts := strings.Split(normalized, "/")
	for _, p := range parts {
		if p == "k8s" || p == "kubernetes" {
			return "Kubernetes"
		}
	}
	return "infrastructure"
}
