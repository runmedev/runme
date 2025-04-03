package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgramResolverResolve(t *testing.T) {
	testInputs := map[string]string{
		"NoValue":     `export TEST_NO_VALUE`,
		"EmptyValue":  `export TEST_EMPTY_VALUE=`,
		"NakedValue":  `export TEST_STRING_VALUE=value`,
		"QuotedValue": `export TEST_STRING_VALUE="value"`,
		"ParamExpr":   `export TEST_PARAM_EXPR=${TEST:7:0}`,
	}

	type testOutput struct {
		expectedOutput string
		expectedResult *ProgramResolverResult
	}

	runTestCases := func(t *testing.T, suite string, mode ProgramResolverMode, strategy Retention, outputs map[string]testOutput) {
		t.Helper()
		for name, program := range testInputs {
			t.Run(suite+"_"+name, func(t *testing.T) {
				output := outputs[name]
				r := NewProgramResolver(mode, []string{})
				buf := bytes.NewBuffer(nil)
				got, err := r.Resolve(strings.NewReader(program), buf, strategy)
				require.NoError(t, err)
				assert.EqualValues(t, output.expectedOutput, buf.String())
				assert.EqualValues(t, output.expectedResult, got)
			})
		}
	}

	runTestCases(t, "Auto_FirstRun",
		ProgramResolverModeAuto,
		RetentionFirstRun,
		map[string]testOutput{
			"NoValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_NO_VALUE set in managed env store\n# \"export TEST_NO_VALUE\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusUnresolved,
							Name:   "TEST_NO_VALUE",
						},
					},
				},
			},
			"EmptyValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_EMPTY_VALUE set in managed env store\n# \"export TEST_EMPTY_VALUE=\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusUnresolved,
							Name:   "TEST_EMPTY_VALUE",
						},
					},
				},
			},
			"NakedValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=value\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusUnresolvedWithMessage,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
						},
					},
				},
			},
			"QuotedValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=\\\"value\\\"\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusUnresolvedWithPlaceholder,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
						},
					},
				},
			},
			"ParamExpr": {
				expectedOutput: "# Managed env store retention strategy: first\n\nexport TEST_PARAM_EXPR=${TEST:7:0}\n",
				expectedResult: &ProgramResolverResult{},
			},
		},
	)

	runTestCases(t, "SkipAll_FirstRun",
		ProgramResolverModeSkipAll,
		RetentionFirstRun,
		map[string]testOutput{
			"NoValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_NO_VALUE set in managed env store\n# \"export TEST_NO_VALUE\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusResolved,
							Name:   "TEST_NO_VALUE",
						},
					},
				},
			},
			"EmptyValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_EMPTY_VALUE set in managed env store\n# \"export TEST_EMPTY_VALUE=\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusResolved,
							Name:   "TEST_EMPTY_VALUE",
						},
					},
				},
			},
			"NakedValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=value\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusResolved,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
							Value:         "value",
						},
					},
				},
			},
			"QuotedValue": {
				expectedOutput: "# Managed env store retention strategy: first\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=\\\"value\\\"\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusResolved,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
							Value:         "value",
						},
					},
				},
			},
			"ParamExpr": {
				expectedOutput: "# Managed env store retention strategy: first\n\nexport TEST_PARAM_EXPR=${TEST:7:0}\n",
				expectedResult: &ProgramResolverResult{},
			},
		},
	)

	runTestCases(t, "Auto_LastRun",
		ProgramResolverModeAuto,
		RetentionLastRun,
		map[string]testOutput{
			"NoValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_NO_VALUE\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: false,
				},
			},
			"EmptyValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_EMPTY_VALUE=\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: false,
				},
			},
			"NakedValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_STRING_VALUE=value\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: false,
				},
			},
			"QuotedValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_STRING_VALUE=\"value\"\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: false,
				},
			},
			"ParamExpr": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_PARAM_EXPR=${TEST:7:0}\n",
				expectedResult: &ProgramResolverResult{},
			},
		},
	)

	runTestCases(t, "PromptAll_LastRun",
		ProgramResolverModePromptAll,
		RetentionLastRun,
		map[string]testOutput{
			"NoValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\n#\n# TEST_NO_VALUE set in managed env store\n# \"export TEST_NO_VALUE\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusUnresolvedWithPlaceholder,
							Name:   "TEST_NO_VALUE",
						},
					},
				},
			},
			"EmptyValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\n#\n# TEST_EMPTY_VALUE set in managed env store\n# \"export TEST_EMPTY_VALUE=\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status: ProgramResolverStatusUnresolvedWithPlaceholder,
							Name:   "TEST_EMPTY_VALUE",
						},
					},
				},
			},
			"NakedValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=value\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusUnresolvedWithPlaceholder,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
							Value:         "value",
						},
					},
				},
			},
			"QuotedValue": {
				expectedOutput: "# Managed env store retention strategy: last\n\n#\n# TEST_STRING_VALUE set in managed env store\n# \"export TEST_STRING_VALUE=\\\"value\\\"\"\n\n",
				expectedResult: &ProgramResolverResult{
					ModifiedProgram: true,
					Variables: []ProgramResolverVarResult{
						{
							Status:        ProgramResolverStatusUnresolvedWithPlaceholder,
							Name:          "TEST_STRING_VALUE",
							OriginalValue: "value",
							Value:         "value",
						},
					},
				},
			},
			"ParamExpr": {
				expectedOutput: "# Managed env store retention strategy: last\n\nexport TEST_PARAM_EXPR=${TEST:7:0}\n",
				expectedResult: &ProgramResolverResult{},
			},
		},
	)
}

func TestProgramResolverResolve_ProgramResolverModeAuto_First(t *testing.T) {
	r := NewProgramResolver(
		ProgramResolverModeAuto,
		[]string{},
		ProgramResolverSourceFunc([]string{"MY_ENV=resolved"}),
	)
	buf := bytes.NewBuffer(nil)
	result, err := r.Resolve(strings.NewReader(`export MY_ENV=default`), buf, RetentionFirstRun)
	require.NoError(t, err)
	require.EqualValues(
		t,
		&ProgramResolverResult{
			ModifiedProgram: true,
			Variables: []ProgramResolverVarResult{
				{
					Status:        ProgramResolverStatusResolved,
					Name:          "MY_ENV",
					OriginalValue: "default",
					Value:         "resolved",
				},
			},
		},
		result,
	)
	require.EqualValues(t, "# Managed env store retention strategy: first\n\n#\n# MY_ENV set in managed env store\n# \"export MY_ENV=default\"\n\n", buf.String())
}

func TestProgramResolverResolve_ProgramResolverModeAuto_Last(t *testing.T) {
	r := NewProgramResolver(
		ProgramResolverModeAuto,
		[]string{},
		ProgramResolverSourceFunc([]string{"MY_ENV=resolved"}),
	)
	buf := bytes.NewBuffer(nil)
	result, err := r.Resolve(strings.NewReader(`export MY_ENV=default`), buf, RetentionLastRun)
	require.NoError(t, err)
	require.EqualValues(
		t,
		&ProgramResolverResult{
			ModifiedProgram: false,
		},
		result,
	)
	require.EqualValues(t, "# Managed env store retention strategy: last\n\nexport MY_ENV=default\n", buf.String())
}

func TestProgramResolverResolve_ProgramResolverModePrompt(t *testing.T) {
	t.Run("Prompt with message", func(t *testing.T) {
		r := NewProgramResolver(
			ProgramResolverModePromptAll,
			[]string{},
			ProgramResolverSourceFunc([]string{"MY_ENV=resolved"}),
		)
		buf := bytes.NewBuffer(nil)
		result, err := r.Resolve(strings.NewReader(`export MY_ENV=message value`), buf, RetentionFirstRun)
		require.NoError(t, err)
		require.EqualValues(
			t,
			&ProgramResolverResult{
				ModifiedProgram: true,
				Variables: []ProgramResolverVarResult{
					{
						Status:        ProgramResolverStatusUnresolvedWithPlaceholder,
						Name:          "MY_ENV",
						OriginalValue: "message value",
						Value:         "resolved",
					},
				},
			},
			result,
		)
		require.EqualValues(t, "# Managed env store retention strategy: first\n\n#\n# MY_ENV set in managed env store\n# \"export MY_ENV=message value\"\n\n", buf.String())
	})

	t.Run("Prompt with placeholder", func(t *testing.T) {
		r := NewProgramResolver(
			ProgramResolverModePromptAll,
			[]string{},
			ProgramResolverSourceFunc([]string{"MY_ENV=resolved"}),
		)
		buf := bytes.NewBuffer(nil)
		result, err := r.Resolve(strings.NewReader(`export MY_ENV="placeholder value"`), buf, RetentionFirstRun)
		require.NoError(t, err)
		require.EqualValues(
			t,
			&ProgramResolverResult{
				ModifiedProgram: true,
				Variables: []ProgramResolverVarResult{
					{
						Status:        ProgramResolverStatusUnresolvedWithPlaceholder,
						Name:          "MY_ENV",
						OriginalValue: "placeholder value",
						Value:         "resolved",
					},
				},
			},
			result,
		)
		require.EqualValues(t, "# Managed env store retention strategy: first\n\n#\n# MY_ENV set in managed env store\n# \"export MY_ENV=\\\"placeholder value\\\"\"\n\n", buf.String())
	})
}

func TestProgramResolverResolve_SensitiveEnvKeys(t *testing.T) {
	t.Run("Prompt with message", func(t *testing.T) {
		r := NewProgramResolver(
			ProgramResolverModePromptAll,
			[]string{"MY_PASSWORD", "MY_SECRET"},
		)
		buf := bytes.NewBuffer(nil)
		result, err := r.Resolve(strings.NewReader("export MY_PASSWORD=super-secret\nexport MY_SECRET=also-secret\nexport MY_PLAIN=text\n"), buf, RetentionFirstRun)
		require.NoError(t, err)
		require.EqualValues(
			t,
			&ProgramResolverResult{
				ModifiedProgram: true,
				Variables: []ProgramResolverVarResult{
					{
						Status:        ProgramResolverStatusUnresolvedWithSecret,
						Name:          "MY_PASSWORD",
						OriginalValue: "super-secret",
						Value:         "",
					},
					{
						Status:        ProgramResolverStatusUnresolvedWithSecret,
						Name:          "MY_SECRET",
						OriginalValue: "also-secret",
						Value:         "",
					},
					{
						Status:        ProgramResolverStatusUnresolvedWithMessage,
						Name:          "MY_PLAIN",
						OriginalValue: "text",
						Value:         "",
					},
				},
			},
			result,
		)
		require.EqualValues(t, "# Managed env store retention strategy: first\n\n#\n# MY_PASSWORD set in managed env store\n# \"export MY_PASSWORD=super-secret\"\n#\n# MY_SECRET set in managed env store\n# \"export MY_SECRET=also-secret\"\n#\n# MY_PLAIN set in managed env store\n# \"export MY_PLAIN=text\"\n\n", buf.String())
	})
}

func TestUnescapeShellLiteral(t *testing.T) {
	assert.Equal(t, `echo "Hello World!"`, unescapeShellLiteral(`echo "Hello World!"`))
	assert.Equal(t, `echo "Hello ${name}!"`, unescapeShellLiteral(`echo "Hello ${name}!"`))
	assert.Equal(t, `[Guest type (hyperv,proxmox,openstack)]`, unescapeShellLiteral(`[Guest type \(hyperv,proxmox,openstack\)]`))
	assert.Equal(t, `[IP of waiting server {foo}]`, unescapeShellLiteral(`[IP of waiting server \{foo\}]`))
	assert.Equal(t, `[Guest\ Type]`, unescapeShellLiteral(`[Guest\ Type]`))
	assert.Equal(t, `[Guest Type]`, unescapeShellLiteral(`\[Guest Type\]`))
}
