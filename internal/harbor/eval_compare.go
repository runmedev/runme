package harbor

import (
	"fmt"
	"io"
	"os"
)

type EvalCompareOptions struct {
	JobsDir       string
	Job           string
	Base          string
	Format        string
	IncludeOracle bool
	AllowErrors   bool
	Stdout        io.Writer
	Stderr        io.Writer
	Git           compareGit
}

type EvalComparer struct {
	opts EvalCompareOptions
}

func NewEvalComparer(opts EvalCompareOptions) *EvalComparer {
	if opts.Base == "" {
		opts.Base = "HEAD"
	}
	if opts.Format == "" {
		opts.Format = "text"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return &EvalComparer{opts: opts}
}

func (c *EvalComparer) Run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("accepts no arguments")
	}
	if c.opts.Format != "text" && c.opts.Format != "json" {
		return fmt.Errorf("unsupported format %q; expected text or json", c.opts.Format)
	}

	paths, err := resolveEvalViewPaths(c.opts.JobsDir)
	if err != nil {
		return err
	}

	gitClient := c.opts.Git
	if gitClient == nil {
		gitClient, err = newGoGitCompareClient()
		if err != nil {
			return err
		}
	}

	base, err := resolveCompareBase(compareBaseOptions{
		git:      gitClient,
		baseRef:  c.opts.Base,
		jobsRoot: paths.jobsDir,
	})
	if err != nil {
		return err
	}

	candidate, err := resolveCompareCandidate(compareCandidateOptions{
		jobsDir:       paths.jobsDir,
		job:           c.opts.Job,
		includeOracle: c.opts.IncludeOracle,
		allowErrors:   c.opts.AllowErrors,
		git:           gitClient,
	})
	if err != nil {
		return err
	}

	comparison := buildEvalComparison(base, candidate, c.opts.Base)
	if c.opts.Format == "json" {
		return renderEvalComparisonJSON(c.opts.Stdout, comparison)
	}
	return renderEvalComparisonText(c.opts.Stdout, comparison)
}
