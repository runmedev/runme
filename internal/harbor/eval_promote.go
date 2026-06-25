package harbor

import (
	"fmt"
	"io"
	"os"
)

const defaultPromoteSubject = "Promote changes verified by task eval"

type EvalPromoteOptions struct {
	JobsDir       string
	Job           string
	Latest        bool
	DryRun        bool
	EvidenceOnly  bool
	IncludeOracle bool
	AllowErrors   bool
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

	jobsRoot, jobDir, selection, err := resolvePromoteJob(promoteJobOptions{
		jobsDir:       p.opts.JobsDir,
		job:           p.opts.Job,
		latest:        p.opts.Latest,
		includeOracle: p.opts.IncludeOracle,
		allowErrors:   p.opts.AllowErrors,
	})
	if err != nil {
		return err
	}

	result, err := readPromoteJobResult(jobDir)
	if err != nil {
		return err
	}
	config, err := readPromoteJobConfig(jobDir)
	if err != nil {
		return err
	}
	if err := validatePromoteJobPolicy(result, config, promoteJobPolicy{
		includeOracle: p.opts.IncludeOracle,
		allowErrors:   p.opts.AllowErrors,
	}); err != nil {
		return err
	}

	gitClient := p.opts.Git
	if gitClient == nil {
		gitClient, err = newGoGitPromoteClient()
		if err != nil {
			return err
		}
	}

	jobRel, err := gitClient.Rel(jobDir)
	if err != nil {
		return err
	}
	jobsRel, err := gitClient.Rel(jobsRoot)
	if err != nil {
		return err
	}
	if p.opts.Latest {
		selection = fmt.Sprintf("latest job under %s", jobsRel)
	}
	if warning := newerPromotableJobWarning(jobsRoot, jobDir, result, promoteJobPolicy{
		includeOracle: p.opts.IncludeOracle,
		allowErrors:   p.opts.AllowErrors,
	}); warning != "" {
		_, _ = fmt.Fprintf(p.opts.Stderr, "warning: %s under %s\n\n", warning, jobsRel)
	}
	resultRel, err := gitClient.Rel(result.Path)
	if err != nil {
		return err
	}
	message := renderPromoteCommitMessage(promoteMessageData{
		subject:      p.opts.Message,
		jobPath:      jobRel,
		resultPath:   resultRel,
		evidenceOnly: p.opts.EvidenceOnly,
		config:       config,
		result:       result,
	})

	if p.opts.DryRun {
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s %s\n", evalOutputLabel(p.opts.Stdout, "Selected eval job:"), jobRel)
		_, _ = fmt.Fprintf(p.opts.Stdout, "%s %s\n\n", evalOutputLabel(p.opts.Stdout, "Selection:"), selection)
		_, _ = fmt.Fprint(p.opts.Stdout, renderPromoteCommitMessageForWriter(p.opts.Stdout, promoteMessageData{
			subject:      p.opts.Message,
			jobPath:      jobRel,
			resultPath:   resultRel,
			evidenceOnly: p.opts.EvidenceOnly,
			config:       config,
			result:       result,
		}))
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

	if len(staged) > 0 {
		latestStagedMod, err := gitClient.LatestModTime(staged)
		if err != nil {
			return err
		}
		if warning := stalePromoteWarning(result, latestStagedMod); warning != "" {
			_, _ = fmt.Fprintf(p.opts.Stderr, "warning: %s\n", warning)
		}
	}
	if err := gitClient.AddJobDir(jobDir); err != nil {
		return err
	}
	hash, err := gitClient.Commit(message)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(p.opts.Stdout, "Promoted eval job %s in commit %s\n", jobRel, hash)
	return nil
}
