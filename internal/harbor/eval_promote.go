package harbor

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultPromoteSubject = "Promote changes verified by task eval"

type EvalPromoteOptions struct {
	JobsDir       string
	Job           string
	Latest        bool
	DryRun        bool
	EvidenceOnly  bool
	Artifacts     bool
	IncludeOracle bool
	AllowErrors   bool
	PromoteAnyway bool
	Message       string
	Stdout        io.Writer
	Stderr        io.Writer
	Git           promoteGit
}

type EvalPromoter struct {
	opts EvalPromoteOptions
}

func NewEvalPromoter(opts EvalPromoteOptions) *EvalPromoter {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Message == "" {
		opts.Message = defaultPromoteSubject
	}
	return &EvalPromoter{opts: opts}
}

func (p *EvalPromoter) Run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("accepts no arguments")
	}
	if p.opts.Job == "" && !p.opts.Latest {
		return fmt.Errorf("one of --job or --latest is required")
	}
	if p.opts.Job != "" && p.opts.Latest {
		return fmt.Errorf("--job cannot be used together with --latest")
	}

	paths, err := resolveEvalViewPaths(p.opts.JobsDir)
	if err != nil {
		return err
	}
	jobsRoot := paths.jobsDir

	gitClient := p.opts.Git
	if gitClient == nil {
		gitClient, err = newGoGitPromoteClient()
		if err != nil {
			return err
		}
	}

	policy := promoteJobPolicy{
		includeOracle: p.opts.IncludeOracle,
		allowErrors:   p.opts.AllowErrors,
	}
	store := evalJobStore{
		jobsDir: jobsRoot,
		git:     gitClient,
		policy:  policy,
	}

	var candidate evalJobRef
	var jobDir string
	if p.opts.Latest {
		candidate, jobDir, err = store.LatestLocal()
	} else {
		candidate, jobDir, err = store.ExplicitLocal(p.opts.Job)
	}
	if err != nil {
		return err
	}

	jobsRel, err := gitClient.Rel(jobsRoot)
	if err != nil {
		return err
	}
	if p.opts.Latest {
		candidate.Selection = fmt.Sprintf("latest job under %s", jobsRel)
	}
	if warning := newerPromotableJobWarning(jobsRoot, jobDir, candidate.Result, policy); warning != "" {
		_, _ = fmt.Fprintf(p.opts.Stderr, "warning: %s under %s; newer eval jobs were skipped\n\n", warning, jobsRel)
	}
	message := renderPromoteCommitMessage(candidate.MessageData(p.opts.Message, p.opts.EvidenceOnly))

	comparison, hasComparison, err := p.comparison(jobsRoot, candidate, gitClient)
	if err != nil {
		return err
	}

	if p.opts.DryRun {
		files, err := gitClient.JobFiles(jobDir, p.opts.Artifacts)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s %s\n", evalOutputLabel(p.opts.Stdout, "Selected eval job:"), candidate.RelPath)
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s %s\n", evalOutputLabel(p.opts.Stdout, "Selection:"), candidate.Selection)
		if p.opts.Artifacts {
			_, _ = fmt.Fprintf(p.opts.Stdout, "%s artifacts\n", evalOutputLabel(p.opts.Stdout, "Evidence mode:"))
		} else {
			_, _ = fmt.Fprintf(p.opts.Stdout, "%s compact\n", evalOutputLabel(p.opts.Stdout, "Evidence mode:"))
		}
		_, _ = fmt.Fprintln(p.opts.Stdout, evalOutputLabel(p.opts.Stdout, "Files to add:"))
		for _, file := range files {
			_, _ = fmt.Fprintf(p.opts.Stdout, "  %s\n", file)
		}
		if p.opts.Artifacts {
			_, _ = fmt.Fprintf(p.opts.Stdout, "%s */workdir/*\n", evalOutputLabel(p.opts.Stdout, "Excluded:"))
		}
		_, _ = fmt.Fprintln(p.opts.Stdout)
		p.renderDryRunComparison(comparison, hasComparison)
		_, _ = fmt.Fprintln(p.opts.Stdout, evalOutputLabel(p.opts.Stdout, "Proposed commit message:"))
		_, _ = fmt.Fprintln(p.opts.Stdout)
		_, _ = fmt.Fprint(p.opts.Stdout, renderPromoteCommitMessage(candidate.MessageData(p.opts.Message, p.opts.EvidenceOnly)))
		return nil
	}

	staged, err := gitClient.StagedFiles()
	if err != nil {
		return err
	}
	if len(staged) == 0 {
		if !p.opts.EvidenceOnly {
			return fmt.Errorf("no staged changes to promote; use --evidence-only to commit only the selected eval job evidence")
		}
	} else if p.opts.EvidenceOnly {
		return fmt.Errorf("--evidence-only cannot be used with staged changes")
	}
	if len(staged) > 0 {
		if conflicts, err := gitClient.UnstagedFilesTouching(staged); err != nil {
			return err
		} else if len(conflicts) > 0 {
			return fmt.Errorf("unstaged changes touch staged files: %s", joinDisplay(conflicts))
		}
	}
	if hasComparison {
		gate := comparison.Gate()
		if gate.Blocking && !p.opts.PromoteAnyway {
			return fmt.Errorf("promotion blocked by eval comparison: %s", gate.Reason)
		}
	}

	if len(staged) > 0 {
		latestStagedMod, err := gitClient.LatestModTime(staged)
		if err != nil {
			return err
		}
		if warning := stalePromoteWarning(candidate.Result, latestStagedMod); warning != "" {
			_, _ = fmt.Fprintf(p.opts.Stderr, "warning: %s\n", warning)
		}
	}
	if err := gitClient.AddJobDir(jobDir, p.opts.Artifacts); err != nil {
		return err
	}
	hash, err := gitClient.Commit(message)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(p.opts.Stdout, "Promoted eval job %s in commit %s\n", candidate.RelPath, hash)
	return nil
}

func (p *EvalPromoter) comparison(jobsRoot string, candidate evalJobRef, gitClient compareGit) (evalComparison, bool, error) {
	base, err := (evalJobStore{
		jobsDir: jobsRoot,
		git:     gitClient,
	}).LatestTracked("HEAD")
	if err != nil {
		if strings.HasPrefix(err.Error(), "no complete Git-tracked eval job found") {
			return evalComparison{}, false, nil
		}
		return evalComparison{}, false, err
	}
	return newEvalComparison(base, candidate, "HEAD"), true, nil
}

func (p *EvalPromoter) renderDryRunComparison(comparison evalComparison, ok bool) {
	if !ok {
		_, _ = fmt.Fprintln(p.opts.Stdout, evalOutputLabel(p.opts.Stdout, "Comparison:")+" no tracked baseline found")
		_, _ = fmt.Fprintln(p.opts.Stdout)
		return
	}
	_, _ = fmt.Fprintln(p.opts.Stdout, evalOutputLabel(p.opts.Stdout, "Comparison:"))
	if err := comparison.RenderText(p.opts.Stdout); err != nil {
		return
	}
	gate := comparison.Gate()
	if gate.Blocking {
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s blocked\n", evalOutputLabel(p.opts.Stdout, "Promotion gate:"))
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s %s\n", evalOutputLabel(p.opts.Stdout, "Reason:"), gate.Reason)
	} else {
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s passed\n", evalOutputLabel(p.opts.Stdout, "Promotion gate:"))
	}
	_, _ = fmt.Fprintln(p.opts.Stdout)
}
