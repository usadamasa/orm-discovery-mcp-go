package cmd

import (
	"github.com/usadamasa/backlog-cli/internal/md"
)

func RunRegenerateMD(dir string) error {
	return md.Generate(dir)
}
