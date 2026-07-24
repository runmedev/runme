package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/runmedev/owl/pkg/owl"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/internal/lru"
	"github.com/runmedev/runme/v3/internal/ulid"
	"github.com/runmedev/runme/v3/project"
)

var owlStoreDefault = false

type envStorer interface {
	getEnv(string) (string, error) // Get
	envs() ([]string, error)       // Items
	addEnvs(context context.Context, envs []string) error
	updateStore(context context.Context, envs []string, newOrUpdated []string, deleted []string) error
	setEnv(context context.Context, k string, v string) error // Set
	sensitiveEnvKeys() ([]string, error)
	subscribe(ctx context.Context, snapshotc chan<- []owl.SnapshotItem) error
	complete()
}

// Session is an abstract entity separate from
// an execution. Currently, its main role is to
// keep track of environment variables.
type Session struct {
	ID        string
	Metadata  map[string]string
	envStorer envStorer

	logger *zap.Logger
}

func NewSession(envs []string, proj *project.Project, logger *zap.Logger) (*Session, error) {
	return NewSessionWithStore(envs, proj, owlStoreDefault, logger)
}

func NewSessionWithStore(envs []string, proj *project.Project, owlStore bool, logger *zap.Logger) (*Session, error) {
	sessionEnvs := []string(envs)

	var storer envStorer
	if owlStore && proj != nil {
		logger.Info("using owl store")
		var err error
		storer, err = newOwlStorer(sessionEnvs, proj, logger)
		if err != nil {
			return nil, err
		}
	} else {
		if proj == nil {
			logger.Debug("owl store requires project in session")
		}
		logger.Debug("using simple env store")
		storer = newRunnerStorer(sessionEnvs...)
	}

	s := &Session{
		ID:        ulid.GenerateID(),
		envStorer: storer,

		logger: logger,
	}

	if proj != nil {
		msg, err := s.loadDirEnv(context.Background(), proj)
		if err != nil {
			logger.Info("failed to load direnv", zap.Error(err))
		} else {
			logger.Info("direnv returned", zap.String("msg", msg))
		}
	}

	return s, nil
}

func (s *Session) Identifier() string {
	return s.ID
}

func (s *Session) UpdateStore(context context.Context, envs []string, newOrUpdated []string, deleted []string) error {
	return s.envStorer.updateStore(context, envs, newOrUpdated, deleted)
}

func (s *Session) AddEnvs(context context.Context, envs []string) error {
	return s.envStorer.addEnvs(context, envs)
}

func (s *Session) SensitiveEnvKeys() ([]string, error) {
	return s.envStorer.sensitiveEnvKeys()
}

func (s *Session) SetEnv(context context.Context, k string, v string) error {
	return s.envStorer.setEnv(context, k, v)
}

func (s *Session) Envs() ([]string, error) {
	vals, err := s.envStorer.envs()
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func (s *Session) Subscribe(ctx context.Context, snapshotc chan<- []owl.SnapshotItem) error {
	return s.envStorer.subscribe(ctx, snapshotc)
}

func (s *Session) Complete() {
	s.envStorer.complete()
}

type runnerEnvStorer struct {
	// logger   *zap.Logger
	envStore *envStore
}

func newRunnerStorer(sessionEnvs ...string) *runnerEnvStorer {
	return &runnerEnvStorer{
		envStore: newEnvStore(sessionEnvs...),
	}
}

func (es *runnerEnvStorer) subscribe(_ context.Context, snapshotc chan<- []owl.SnapshotItem) error {
	defer close(snapshotc)
	return fmt.Errorf("not available for runner env store")
}

func (es *runnerEnvStorer) complete() {
	// noop
}

func (es *runnerEnvStorer) addEnvs(_ context.Context, envs []string) error {
	es.envStore.Add(envs...)
	return nil
}

func (es *runnerEnvStorer) sensitiveEnvKeys() ([]string, error) {
	// noop, not supported
	return []string{}, nil
}

func (es *runnerEnvStorer) getEnv(name string) (string, error) {
	return es.envStore.Get(name), nil
}

func (es *runnerEnvStorer) envs() ([]string, error) {
	envs, err := es.envStore.Values()
	if err != nil {
		return nil, err
	}
	return envs, nil
}

func (es *runnerEnvStorer) setEnv(_ context.Context, k string, v string) error {
	_, err := es.envStore.Set(k, v)
	return err
}

func (es *runnerEnvStorer) updateStore(_ context.Context, envs []string, newOrUpdated []string, deleted []string) error {
	es.envStore = newEnvStore(envs...).Add(newOrUpdated...).Delete(deleted...)
	return nil
}

type owlEnvStorerSubscriber chan<- []owl.SnapshotItem

type owlEnvStorer struct {
	logger   *zap.Logger
	owlStore *owl.Store

	mu          sync.RWMutex
	subscribers []owlEnvStorerSubscriber
}

func newOwlStorer(envs []string, proj *project.Project, logger *zap.Logger) (*owlEnvStorer, error) {
	// todo(sebastian): technically system should be session
	opts := []owl.StoreOption{
		owl.WithDotenv("[system]", strings.NewReader(dotenvLines(envs))),
	}

	envSpecFiles := []string{}
	// envFilesOrder := []string{}
	if proj != nil {
		// todo(sebastian): specs loading should be independent of project
		envSpecFiles = []string{".env.sample", ".env.example", ".env.spec"}
	}

	for _, specFile := range envSpecFiles {
		raw, _ := proj.LoadRawFile(specFile)
		if raw == nil {
			continue
		}
		opts = append(opts, owl.WithEnvSpec(specFile, strings.NewReader(string(raw))))
	}

	envWithSource, err := proj.LoadEnvWithSource()
	if err != nil {
		return nil, err
	}

	for envSource, envMap := range envWithSource {
		envs := []string{}
		for k, v := range envMap {
			env := fmt.Sprintf("%s=%s", k, v)
			envs = append(envs, env)
		}
		sort.Strings(envs)
		opts = append(opts, owl.WithDotenv(envSource, strings.NewReader(dotenvLines(envs))))
	}

	owlYAML, err := proj.LoadRawFile(".runme/owl.yaml")
	if err != nil {
		return nil, err
	} else if owlYAML != nil {
		logger.Warn("ignoring .runme/owl.yaml because Owl v2 resolver/CRD support is not part of this cutover")
	}

	owlStore, err := owl.NewStore(opts...)
	if err != nil {
		return nil, err
	}

	return &owlEnvStorer{
		logger:   logger,
		owlStore: owlStore,
	}, nil
}

func (es *owlEnvStorer) subscribe(context context.Context, snapshotc chan<- []owl.SnapshotItem) error {
	defer es.mu.Unlock()
	es.mu.Lock()
	es.logger.Debug("subscribed to owl store")

	es.subscribers = append(es.subscribers, snapshotc)

	go func() {
		<-context.Done()
		err := es.unsubscribe(snapshotc)
		if err != nil {
			es.logger.Error("unsubscribe from owl store failed", zap.Error(err))
		}
	}()

	// avoid deadlock
	go func() {
		es.notifySubscribers()
	}()

	return nil
}

func (es *owlEnvStorer) complete() {
	defer es.mu.Unlock()
	es.mu.Lock()

	for _, sub := range es.subscribers {
		err := es.unsubscribeUnsafe(sub)
		if err != nil {
			es.logger.Error("unsubscribe from owl store failed", zap.Error(err))
		}
	}
}

func (es *owlEnvStorer) unsubscribe(snapshotc chan<- []owl.SnapshotItem) error {
	defer es.mu.Unlock()
	es.mu.Lock()

	return es.unsubscribeUnsafe(snapshotc)
}

func (es *owlEnvStorer) unsubscribeUnsafe(snapshotc chan<- []owl.SnapshotItem) error {
	es.logger.Debug("unsubscribed from owl store")

	for i, sub := range es.subscribers {
		if sub == snapshotc {
			es.subscribers = append(es.subscribers[:i], es.subscribers[i+1:]...)
			close(sub)
			return nil
		}
	}

	return fmt.Errorf("unknown subscriber")
}

func (es *owlEnvStorer) notifySubscribers() {
	defer es.mu.RUnlock()
	es.mu.RLock()

	snapshot, err := es.owlStore.Snapshot(owl.SnapshotPolicy{})
	if err != nil {
		es.logger.Error("failed to get snapshot", zap.Error(err))
		return
	}

	for _, sub := range es.subscribers {
		sub <- snapshot
	}
}

func (es *owlEnvStorer) updateStore(context context.Context, envs []string, newOrUpdated []string, deleted []string) error {
	if err := es.owlStore.Update(context, newOrUpdated, deleted); err != nil {
		return err
	}
	es.notifySubscribers()
	return nil
}

func (es *owlEnvStorer) addEnvs(context context.Context, envs []string) error {
	if err := es.owlStore.Update(context, envs, nil); err != nil {
		return err
	}
	es.notifySubscribers()
	return nil
}

func (es *owlEnvStorer) getEnv(name string) (string, error) {
	v, _, err := es.owlStore.Get(name, owl.GetPolicy{Reveal: true})
	return v.Value, err
}

func (es *owlEnvStorer) sensitiveEnvKeys() ([]string, error) {
	vals, err := es.owlStore.SensitiveKeys()
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func (es *owlEnvStorer) setEnv(context context.Context, k string, v string) error {
	// todo(sebastian): add checking env length inside Update
	err := es.owlStore.Update(context, []string{fmt.Sprintf("%s=%s", k, v)}, nil)
	if err != nil {
		return err
	}
	es.notifySubscribers()
	return err
}

func (es *owlEnvStorer) envs() ([]string, error) {
	vals, err := es.owlStore.Dotenv(owl.DotenvPolicy{Insecure: true})
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func dotenvLines(envs []string) string {
	if len(envs) == 0 {
		return ""
	}
	return strings.Join(envs, "\n") + "\n"
}

type sessionList = lru.Cache[*Session]

// sessionListCapacity is a maximum number of entries
// stored in a single SessionList.
const sessionListCapacity = 1024

func newSessionList() *sessionList {
	return lru.NewCache[*Session](sessionListCapacity)
}
