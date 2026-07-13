package harbor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type EvalTaskNewOptions struct {
	TasksDir    string
	Org         string
	Description string
	Authors     []string
	NoSolution  bool
	Force       bool
	GitConfig   GitConfigFunc
	Stdout      io.Writer
	Stderr      io.Writer
}

type GitConfigFunc func(key string) (string, error)

type EvalTaskNewer struct {
	opts EvalTaskNewOptions
}

type evalTaskNewPaths struct {
	tasksDir string
	baseDir  string
}

type taskAuthor struct {
	Name  string
	Email string
}

type taskTemplateData struct {
	FullName         string
	ShortName        string
	Description      string
	Authors          []taskAuthor
	AuthorSummary    string
	EvalDatasetPath  string
	ContainerWorkdir string
}

var authorPattern = regexp.MustCompile(`^(.+?)\s*<(.+?)>\s*$`)

func NewEvalTaskNewer(opts EvalTaskNewOptions) *EvalTaskNewer {
	if opts.TasksDir == "" {
		opts.TasksDir = DefaultEvalDatasetPath
	}
	if opts.GitConfig == nil {
		opts.GitConfig = gitConfig
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return &EvalTaskNewer{opts: opts}
}

func (r *EvalTaskNewer) Run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("accepts exactly 1 task name, received %d", len(args))
	}

	fullName, shortName, err := resolveEvalTaskName(args[0], r.opts.Org)
	if err != nil {
		return err
	}
	authors, err := r.resolveAuthors()
	if err != nil {
		return err
	}
	paths, err := resolveEvalTaskNewPaths(r.opts.TasksDir)
	if err != nil {
		return err
	}

	taskDir := filepath.Join(paths.tasksDir, shortName)
	if err := ensureEvalTaskTarget(taskDir, r.opts.Force); err != nil {
		return err
	}
	containerWorkdir, err := evalTaskContainerWorkdir(paths.baseDir, taskDir)
	if err != nil {
		return err
	}
	evalDatasetPath, err := evalTaskDatasetPath(paths.baseDir, paths.tasksDir)
	if err != nil {
		return err
	}

	data := taskTemplateData{
		FullName:         fullName,
		ShortName:        shortName,
		Description:      r.opts.Description,
		Authors:          authors,
		AuthorSummary:    formatAuthors(authors),
		EvalDatasetPath:  evalDatasetPath,
		ContainerWorkdir: containerWorkdir,
	}
	if err := writeEvalTaskScaffold(taskDir, data, r.opts.NoSolution); err != nil {
		return err
	}

	return writeEvalTaskSummary(r.opts.Stdout, taskDir, data.AuthorSummary, r.opts.NoSolution)
}

func resolveEvalTaskNewPaths(tasksDir string) (evalTaskNewPaths, error) {
	invocationCwd, err := os.Getwd()
	if err != nil {
		return evalTaskNewPaths{}, err
	}
	invocationCwd = cleanExistingPath(invocationCwd)
	baseDir := defaultEvalBaseDir(invocationCwd)

	if tasksDir == "" {
		tasksDir = DefaultEvalDatasetPath
	}
	if filepath.IsAbs(tasksDir) {
		return evalTaskNewPaths{tasksDir: cleanExistingPath(tasksDir), baseDir: baseDir}, nil
	}
	if tasksDir == DefaultEvalDatasetPath {
		return evalTaskNewPaths{tasksDir: filepath.Join(baseDir, tasksDir), baseDir: baseDir}, nil
	}
	return evalTaskNewPaths{tasksDir: filepath.Clean(filepath.Join(invocationCwd, tasksDir)), baseDir: baseDir}, nil
}

func evalTaskContainerWorkdir(baseDir, taskDir string) (string, error) {
	baseDir = cleanExistingPath(baseDir)
	taskDir = cleanExistingPath(taskDir)
	rel := relativePathUnder(baseDir, taskDir)
	if rel == "" || rel == "." {
		return "", fmt.Errorf("eval task directory must be under workspace root %s to map to /app: %s", baseDir, taskDir)
	}
	return "/app/" + filepath.ToSlash(filepath.Join(rel, "workdir")), nil
}

func evalTaskDatasetPath(baseDir, tasksDir string) (string, error) {
	baseDir = cleanExistingPath(baseDir)
	tasksDir = cleanExistingPath(tasksDir)
	rel := relativePathUnder(baseDir, tasksDir)
	if rel == "" {
		return "", fmt.Errorf("eval tasks directory must be under workspace root %s: %s", baseDir, tasksDir)
	}
	return filepath.ToSlash(rel), nil
}

func resolveEvalTaskName(name, org string) (string, string, error) {
	name = strings.TrimSpace(name)
	org = strings.TrimSpace(org)
	if name == "" {
		return "", "", errors.New("task name cannot be empty")
	}
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("task name must be in org/name format: %s", name)
		}
		if err := validateEvalTaskDirName(parts[1]); err != nil {
			return "", "", err
		}
		return name, parts[1], nil
	}
	if org == "" {
		return "", "", fmt.Errorf("task name %q is missing an organization; pass org/name or --org", name)
	}
	if strings.Contains(org, "/") {
		return "", "", fmt.Errorf("organization cannot contain /: %s", org)
	}
	if err := validateEvalTaskDirName(name); err != nil {
		return "", "", err
	}
	return org + "/" + name, name, nil
}

func validateEvalTaskDirName(name string) error {
	if name == "." || name == ".." || strings.Contains(name, "/") || strings.Contains(name, `\`) {
		return fmt.Errorf("invalid task directory name: %s", name)
	}
	cleaned := filepath.Clean(name)
	if cleaned != name {
		return fmt.Errorf("invalid task directory name: %s", name)
	}
	return nil
}

func (r *EvalTaskNewer) resolveAuthors() ([]taskAuthor, error) {
	if len(r.opts.Authors) > 0 {
		return parseTaskAuthors(r.opts.Authors)
	}

	name, nameErr := r.opts.GitConfig("user.name")
	email, emailErr := r.opts.GitConfig("user.email")
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if nameErr != nil || name == "" {
		return nil, nil
	}
	if emailErr != nil || email == "" {
		return []taskAuthor{{Name: name}}, nil
	}
	return []taskAuthor{{Name: name, Email: email}}, nil
}

func parseTaskAuthors(values []string) ([]taskAuthor, error) {
	authors := make([]taskAuthor, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("author cannot be empty")
		}
		match := authorPattern.FindStringSubmatch(value)
		if match != nil {
			name := strings.TrimSpace(match[1])
			email := strings.TrimSpace(match[2])
			if name == "" || email == "" {
				return nil, fmt.Errorf("invalid author: %s", value)
			}
			authors = append(authors, taskAuthor{Name: name, Email: email})
			continue
		}
		authors = append(authors, taskAuthor{Name: value})
	}
	return authors, nil
}

func formatAuthors(authors []taskAuthor) string {
	parts := make([]string, 0, len(authors))
	for _, author := range authors {
		if author.Email != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", author.Name, author.Email))
		} else {
			parts = append(parts, author.Name)
		}
	}
	return strings.Join(parts, ", ")
}

func ensureEvalTaskTarget(taskDir string, force bool) error {
	info, err := os.Stat(taskDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("eval task path exists and is not a directory: %s", taskDir)
		}
		if !force {
			return fmt.Errorf("eval task already exists: %s; pass --force to overwrite scaffold-owned files", taskDir)
		}
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeEvalTaskScaffold(taskDir string, data taskTemplateData, noSolution bool) error {
	files := []struct {
		template string
		target   string
		mode     os.FileMode
	}{
		{"README.md.tmpl", "README.md", 0o644},
		{"task.toml.tmpl", "task.toml", 0o644},
		{"instruction.md.tmpl", "instruction.md", 0o644},
		{"environment/Dockerfile.tmpl", "environment/Dockerfile", 0o644},
		{"workdir/.gitignore.tmpl", "workdir/.gitignore", 0o644},
		{"tests/test.sh.tmpl", "tests/test.sh", 0o755},
	}
	if !noSolution {
		files = append(files, struct {
			template string
			target   string
			mode     os.FileMode
		}{"solution/solve.sh.tmpl", "solution/solve.sh", 0o755})
	}

	for _, file := range files {
		if err := writeEvalTaskTemplate(taskDir, file.template, file.target, file.mode, data); err != nil {
			return err
		}
	}

	workdirKeep := filepath.Join(taskDir, "workdir", ".gitkeep")
	if err := os.MkdirAll(filepath.Dir(workdirKeep), 0o750); err != nil {
		return err
	}
	return os.WriteFile(workdirKeep, nil, 0o644) // #nosec G306 -- generated scaffold files should be user-readable.
}

func writeEvalTaskSummary(w io.Writer, taskDir, authorSummary string, noSolution bool) error {
	lines := []string{
		fmt.Sprintf("Task initialized in %s\n", taskDir),
	}
	if authorSummary != "" {
		lines = append(lines, fmt.Sprintf("Author: %s\n", authorSummary))
	}
	lines = append(
		lines,
		"Next steps:\n",
		fmt.Sprintf("- Add your instruction to: %s\n", filepath.Join(taskDir, "instruction.md")),
		fmt.Sprintf("- Optional Docker setup (--env docker): %s\n", filepath.Join(taskDir, "environment", "Dockerfile")),
		fmt.Sprintf("- Use the test script to generate a reward: %s\n", filepath.Join(taskDir, "tests", "test.sh")),
	)
	if !noSolution {
		lines = append(lines, fmt.Sprintf("- Fill out the solution: %s\n", filepath.Join(taskDir, "solution", "solve.sh")))
	}
	_, err := io.WriteString(w, strings.Join(lines, ""))
	return err
}

func writeEvalTaskTemplate(taskDir, templatePath, targetPath string, mode os.FileMode, data taskTemplateData) error {
	content, err := renderEvalTaskTemplate(templatePath, data)
	if err != nil {
		return err
	}
	target := filepath.Join(taskDir, targetPath)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	return os.WriteFile(target, content, mode) // #nosec G306 -- generated scaffold files need their requested modes.
}

func renderEvalTaskTemplate(path string, data taskTemplateData) ([]byte, error) {
	source, err := evalTaskNewTemplates.ReadFile(filepath.ToSlash(filepath.Join("eval_task_new_templates", path)))
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(path).Funcs(template.FuncMap{
		"tomlString": tomlString,
	}).Parse(string(source))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func tomlString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\b", "\\b",
		"\t", "\\t",
		"\n", "\\n",
		"\f", "\\f",
		"\r", "\\r",
	)
	return `"` + replacer.Replace(value) + `"`
}

func gitConfig(key string) (string, error) {
	output, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
