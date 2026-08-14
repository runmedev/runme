package session

import (
	"context"
	"slices"

	"github.com/runmedev/runme/v3/project"
)

type SessionStoreSeed struct {
	SeedEnv    []string
	RequestEnv []string
	Project    *project.Project
}

type SessionStore interface {
	New(seed SessionStoreSeed) (EnvStore, error)
}

type plainSessionStore struct{}

func (plainSessionStore) New(seed SessionStoreSeed) (EnvStore, error) {
	envStore := NewEnvStore()

	if err := envStore.Load("[system]", seed.SeedEnv...); err != nil {
		return nil, err
	}

	if err := loadProjectEnvStore(envStore, seed.Project); err != nil {
		return nil, err
	}

	if err := envStore.Merge(context.Background(), seed.RequestEnv...); err != nil {
		return nil, err
	}

	return envStore, nil
}

func mergedProcessEnv(seed SessionStoreSeed) []string {
	positions := make(map[string]int, len(seed.SeedEnv)+len(seed.RequestEnv))
	envs := make([]string, 0, len(seed.SeedEnv)+len(seed.RequestEnv))
	for _, env := range slices.Concat(seed.SeedEnv, seed.RequestEnv) {
		key, _ := SplitEnv(env)
		if key == "" {
			continue
		}
		if position, ok := positions[key]; ok {
			envs[position] = env
			continue
		}
		positions[key] = len(envs)
		envs = append(envs, env)
	}
	return envs
}
