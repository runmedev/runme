package session

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/runmedev/owl/pkg/owl"
	"github.com/runmedev/owl/pkg/owl/seed"

	rcontext "github.com/runmedev/runme/v3/runner/context"
)

type envStoreOwl struct {
	// logger   *zap.Logger
	owlStore *owl.Store

	// mu sync.RWMutex
	// subscribers []owlEnvStorerSubscriber
}

type owlSessionStore struct{}

func (owlSessionStore) New(storeSeed SessionStoreSeed) (EnvStore, error) {
	options := seed.Options{
		Observed: []seed.ObservedSource{{
			Source:  owl.Source{Name: "[process]", Kind: "process"},
			Environ: mergedProcessEnv(storeSeed),
		}},
		Direnv: seed.DirenvDisabled,
	}

	if storeSeed.Project != nil {
		options.EnvFiles = storeSeed.Project.EnvFilesReadOrder()
		options.WorkDir = storeSeed.Project.Root()
		if storeSeed.Project.EnvDirEnvEnabled() {
			options.Direnv = seed.DirenvEnabledWarn
		}
	} else {
		options.WorkDir = projectlessSeedWorkDir()
	}

	result, err := seed.NewStore(context.Background(), options)
	if err != nil {
		return nil, err
	}
	if err := result.Store.Update(context.Background(), storeSeed.RequestEnv, nil); err != nil {
		return nil, err
	}

	return &envStoreOwl{
		owlStore: result.Store,
	}, nil
}

func projectlessSeedWorkDir() string {
	return filepath.Join(os.TempDir(), "runme-owl-projectless-"+strconv.Itoa(os.Getpid()))
}

var _ EnvStore = new(envStoreOwl)

func (s *envStoreOwl) Load(source string, envs ...string) error {
	return s.owlStore.LoadDotenvLines(source, envs...)
}

func (s *envStoreOwl) Merge(ctx context.Context, envs ...string) error {
	return s.owlStore.Update(owlContext(ctx), envs, nil)
}

func (s *envStoreOwl) Get(k string) (string, bool) {
	// todo(sebastian): return error?
	if v, ok, err := s.owlStore.Get(k, owl.GetPolicy{Reveal: true}); err == nil {
		return v.Value, ok
	}

	return "", false
}

func (s *envStoreOwl) Set(ctx context.Context, k, v string) error {
	if len(k)+len(v) > MaxEnvSizeInBytes {
		return ErrEnvTooLarge
	}

	return s.owlStore.Update(owlContext(ctx), []string{k + "=" + v}, nil)
}

func (s *envStoreOwl) Delete(ctx context.Context, k string) error {
	return s.owlStore.Update(owlContext(ctx), nil, []string{k})
}

func (s *envStoreOwl) Items() ([]string, error) {
	return s.owlStore.Dotenv(owl.DotenvPolicy{Insecure: true})
}

func owlContext(ctx context.Context) context.Context {
	execInfo, ok := rcontext.ExecutionInfoFromContext(ctx)
	if !ok {
		return ctx
	}

	return owl.ContextWithExecutionInfo(ctx, owl.ExecutionInfo{
		KnownID:     execInfo.KnownID,
		KnownName:   execInfo.KnownName,
		ExecContext: execInfo.ExecContext,
	})
}
