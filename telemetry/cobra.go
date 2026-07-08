package telemetry

import (
	"sync"

	"github.com/spf13/cobra"
)

var reportCLIInvocation = reportCLIInvocationDefault

func InstrumentCommandTree(root *cobra.Command) {
	var once sync.Once
	instrumentCommandTree(root, root, &once)
}

func instrumentCommandTree(root, cmd *cobra.Command, once *sync.Once) {
	if cmd == nil {
		return
	}

	if cmd == root || cmd.PersistentPreRun != nil || cmd.PersistentPreRunE != nil {
		wrapPersistentPreRun(cmd, once)
	}

	for _, child := range cmd.Commands() {
		instrumentCommandTree(root, child, once)
	}
}

func wrapPersistentPreRun(cmd *cobra.Command, once *sync.Once) {
	oldRun := cmd.PersistentPreRun
	oldRunE := cmd.PersistentPreRunE

	cmd.PersistentPreRun = nil
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		once.Do(func() {
			reportCLIInvocation(cmd)
		})

		if oldRunE != nil {
			return oldRunE(cmd, args)
		}
		if oldRun != nil {
			oldRun(cmd, args)
		}

		return nil
	}
}
