package session

import (
	"context"

	"github.com/runmedev/owl/pkg/owl"

	rcontext "github.com/runmedev/runme/v3/runner/context"
)

type envStoreOwl struct {
	// logger   *zap.Logger
	owlStore *owl.Store

	// mu sync.RWMutex
	// subscribers []owlEnvStorerSubscriber
}

func newOwlStore() (*envStoreOwl, error) {
	owlStore, err := owl.NewStore()
	if err != nil {
		return nil, err
	}

	return &envStoreOwl{
		owlStore: owlStore,
	}, nil
}

var _ EnvStore = new(envStoreOwl)

func (s *envStoreOwl) Load(source string, envs ...string) error {
	return s.owlStore.LoadEnvs(source, envs...)
}

func (s *envStoreOwl) Merge(context context.Context, envs ...string) error {
	return s.owlStore.Update(owlContext(context), envs, nil)
}

func (s *envStoreOwl) Get(k string) (string, bool) {
	// todo(sebastian): return error?
	if v, ok, err := s.owlStore.InsecureGet(k); err == nil {
		return v, ok
	}

	return "", false
}

func (s *envStoreOwl) Set(context context.Context, k, v string) error {
	if len(k)+len(v) > MaxEnvSizeInBytes {
		return ErrEnvTooLarge
	}

	return s.owlStore.Update(owlContext(context), []string{k + "=" + v}, nil)
}

func (s *envStoreOwl) Delete(context context.Context, k string) error {
	return s.owlStore.Update(owlContext(context), nil, []string{k})
}

func (s *envStoreOwl) Items() ([]string, error) {
	return s.owlStore.InsecureValues()
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
