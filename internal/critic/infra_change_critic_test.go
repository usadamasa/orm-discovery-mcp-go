package critic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// Compile-time check that InfraChangeCritic implements Critic.
var _ Critic = (*InfraChangeCritic)(nil)

func TestInfraChangeCritic_Name(t *testing.T) {
	c := NewInfraChangeCritic()
	assert.Equal(t, "InfraChangeCritic", c.Name())
}

func TestInfraChangeCritic_Review(t *testing.T) {
	tests := []struct {
		name          string
		classified    []git.ClassifiedFile
		wantFindings  int
		wantPaths     []string
		wantNoFinding bool
	}{
		{
			name: "Dockerfile added generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "Dockerfile",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"Dockerfile"},
		},
		{
			name: "docker-compose.yml modified generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "docker-compose.yml",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"docker-compose.yml"},
		},
		{
			name: "CI workflow modified generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   ".github/workflows/ci.yml",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{".github/workflows/ci.yml"},
		},
		{
			name: "Terraform file added generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "infra/main.tf",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"infra/main.tf"},
		},
		{
			name: "k8s yaml modified generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "k8s/deployment.yaml",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"k8s/deployment.yaml"},
		},
		{
			name: "infra file deleted generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "Dockerfile",
						Status: git.FileStatusDeleted,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"Dockerfile"},
		},
		{
			name: "code files only produce no findings",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/handler.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "test files only produce no findings",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/handler_test.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryTest,
				},
			},
			wantNoFinding: true,
		},
		{
			name:          "empty classified files produces no findings",
			classified:    nil,
			wantNoFinding: true,
		},
		{
			name: "multiple infra files generate multiple findings",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "Dockerfile",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   ".github/workflows/ci.yml",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryInfra,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "infra/main.tf",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantFindings: 3,
			wantPaths:    []string{"Dockerfile", ".github/workflows/ci.yml", "infra/main.tf"},
		},
		{
			name: "renamed infra file is skipped",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:    "Dockerfile.prod",
						OldPath: "Dockerfile",
						Status:  git.FileStatusRenamed,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantNoFinding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewInfraChangeCritic()
			input := ReviewInput{
				Diff:            &git.DiffResult{},
				ClassifiedFiles: tt.classified,
			}

			findings, err := c.Review(context.Background(), input)
			require.NoError(t, err)

			if tt.wantNoFinding {
				assert.Empty(t, findings, "expected no findings")
				return
			}

			assert.Len(t, findings, tt.wantFindings)
			for i, f := range findings {
				assert.Equal(t, model.SeverityWarning, f.Severity)
				assert.Equal(t, model.CategoryInfraChange, f.Category)
				assert.Equal(t, "InfraChangeCritic", f.CriticName)
				assert.NotEmpty(t, f.ID)
				assert.NotEmpty(t, f.Message)
				assert.NotZero(t, f.CreatedAt)

				if i < len(tt.wantPaths) {
					assert.Equal(t, tt.wantPaths[i], f.Location.FilePath)
				}
			}
		})
	}
}

func TestInfraChangeCritic_FindingFields(t *testing.T) {
	c := NewInfraChangeCritic()
	input := ReviewInput{
		Diff: &git.DiffResult{},
		ClassifiedFiles: []git.ClassifiedFile{
			{
				ChangedFile: git.ChangedFile{
					Path:   "Dockerfile",
					Status: git.FileStatusAdded,
				},
				Category: git.FileCategoryInfra,
			},
		},
	}

	findings, err := c.Review(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, findings, 1)

	f := findings[0]
	assert.Equal(t, model.SeverityWarning, f.Severity)
	assert.Equal(t, model.CategoryInfraChange, f.Category)
	assert.Equal(t, "InfraChangeCritic", f.CriticName)
	assert.Equal(t, "Dockerfile", f.Location.FilePath)
	assert.Contains(t, f.Message, "Dockerfile")
	assert.NotEmpty(t, f.Suggestion)
	assert.True(t, f.Confidence > 0)
	assert.True(t, f.Confidence <= 1.0)
}

func TestInfraChangeCritic_MessageContainsInfraType(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantContains string
	}{
		{"Dockerfile", "Dockerfile", "Dockerfile"},
		{"Docker Compose", "docker-compose.yml", "Docker Compose"},
		{"CI/CD", ".github/workflows/ci.yml", "CI/CD"},
		{"Terraform", "infra/main.tf", "Terraform"},
		{"Kubernetes", "k8s/deployment.yaml", "Kubernetes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewInfraChangeCritic()
			input := ReviewInput{
				Diff: &git.DiffResult{},
				ClassifiedFiles: []git.ClassifiedFile{
					{
						ChangedFile: git.ChangedFile{
							Path:   tt.path,
							Status: git.FileStatusModified,
						},
						Category: git.FileCategoryInfra,
					},
				},
			}

			findings, err := c.Review(context.Background(), input)
			require.NoError(t, err)
			require.Len(t, findings, 1)
			assert.Contains(t, findings[0].Message, tt.wantContains)
		})
	}
}
