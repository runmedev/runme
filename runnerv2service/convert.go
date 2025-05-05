package runnerv2service

import (
	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/project"
	"github.com/runmedev/runme/v3/session"
)

func convertSessionToProtoSession(sess *session.Session) *runnerv2.Session {
	return &runnerv2.Session{
		Id:  sess.ID,
		Env: sess.GetAllEnv(),
		// Metadata: sess.Metadata,
	}
}

// TODO(adamb): this function should not return nil project and nil error at the same time.
func convertProtoProjectToProject(runnerProj *runnerv2.Project) (*project.Project, error) {
	if runnerProj == nil {
		return nil, nil
	}

	opts := project.DefaultProjectOptions[:]

	if runnerProj.EnvLoadOrder != nil {
		opts = append(opts, project.WithEnvFilesReadOrder(runnerProj.EnvLoadOrder))
	}

	return project.NewDirProject(runnerProj.Root, opts...)
}
