package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	tasqueue "github.com/kalbhor/tasqueue/v2"
	broker "github.com/kalbhor/tasqueue/v2/brokers/in-memory"
	results "github.com/kalbhor/tasqueue/v2/results/in-memory"
)

const conversionTask = "image.convert"

type Converter interface {
	Convert(context.Context, Request) (Artifact, error)
}

type Request struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	Architecture string `json:"architecture"`
	BaseProfile  string `json:"baseProfile"`
}

type Artifact struct {
	Digest   string   `json:"digest"`
	Manifest string   `json:"manifest"`
	Kernel   string   `json:"kernel"`
	Rootfs   string   `json:"rootfs"`
	Warnings []string `json:"warnings,omitempty"`
}

type State string

const (
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateSucceeded State = "succeeded"
	StateFailed    State = "failed"
)

type Operation struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	State     State     `json:"state"`
	Request   Request   `json:"request"`
	Artifact  *Artifact `json:"artifact,omitempty"`
	Error     string    `json:"error,omitempty"`
	Logs      []string  `json:"logs,omitempty"`
	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

type payload struct {
	OperationID string  `json:"operationId"`
	Request     Request `json:"request"`
}

type Manager struct {
	queue         *tasqueue.Server
	converter     Converter
	log           *slog.Logger
	workerTimeout time.Duration
	mu            sync.RWMutex
	operations    map[string]Operation
}

func NewManager(ctx context.Context, workers int, converter Converter, log *slog.Logger) (*Manager, error) {
	return NewManagerWithTimeout(ctx, workers, converter, log, 30*time.Minute)
}

func NewManagerWithTimeout(ctx context.Context, workers int, converter Converter, log *slog.Logger, workerTimeout time.Duration) (*Manager, error) {
	if converter == nil {
		return nil, fmt.Errorf("converter is required")
	}
	if workers < 1 {
		workers = 1
	}
	if log == nil {
		log = slog.Default()
	}
	queue, err := tasqueue.NewServer(tasqueue.ServerOpts{
		Broker:  broker.New(),
		Results: results.New(),
		Logger:  log.Handler(),
	})
	if err != nil {
		return nil, fmt.Errorf("create operation queue: %w", err)
	}
	if workerTimeout <= 0 {
		workerTimeout = 30 * time.Minute
	}
	manager := &Manager{
		queue:         queue,
		converter:     converter,
		log:           log,
		workerTimeout: workerTimeout,
		operations:    make(map[string]Operation),
	}
	if err := queue.RegisterTask(conversionTask, manager.handleConversion, tasqueue.TaskOpts{Concurrency: uint32(workers)}); err != nil {
		return nil, fmt.Errorf("register conversion task: %w", err)
	}
	go queue.Start(ctx)
	return manager, nil
}

func (m *Manager) Enqueue(ctx context.Context, req Request) (Operation, error) {
	if req.Source == "" {
		return Operation{}, fmt.Errorf("source is required")
	}
	if req.SourceType == "" {
		req.SourceType = "oci"
	}
	switch req.SourceType {
	case "oci", "docker", "dockerfile", "archive":
	default:
		return Operation{}, fmt.Errorf("unsupported source type %q", req.SourceType)
	}
	if req.Architecture == "" {
		req.Architecture = "native"
	}
	if req.BaseProfile == "" {
		req.BaseProfile = "alpine"
	}
	now := time.Now().UTC()
	op := Operation{ID: uuid.NewString(), Kind: conversionTask, State: StateQueued, Request: req, Logs: []string{fmt.Sprintf("queued %s", req.Source)}, CreatedAt: now, UpdatedAt: now}
	m.mu.Lock()
	m.operations[op.ID] = op
	m.mu.Unlock()
	body, err := json.Marshal(payload{OperationID: op.ID, Request: req})
	if err != nil {
		return Operation{}, fmt.Errorf("encode operation: %w", err)
	}
	job, err := tasqueue.NewJob(conversionTask, body, tasqueue.JobOpts{ID: op.ID, Timeout: m.workerTimeout})
	if err != nil {
		return Operation{}, fmt.Errorf("create conversion job: %w", err)
	}
	if _, err := m.queue.Enqueue(ctx, job); err != nil {
		m.update(op.ID, func(current *Operation) { current.State = StateFailed; current.Error = err.Error() })
		return Operation{}, fmt.Errorf("enqueue conversion: %w", err)
	}
	return op, nil
}

func (m *Manager) List() []Operation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]Operation, 0, len(m.operations))
	for _, op := range m.operations {
		items = append(items, op)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items
}

func (m *Manager) Get(id string) (Operation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	op, ok := m.operations[id]
	return op, ok
}

func (m *Manager) handleConversion(raw []byte, jobCtx tasqueue.JobCtx) error {
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	m.update(p.OperationID, func(op *Operation) {
		op.State = StateRunning
		op.Logs = append(op.Logs, fmt.Sprintf("running %s", p.Request.Source))
	})
	artifact, err := m.converter.Convert(jobCtx, p.Request)
	if err != nil {
		m.update(p.OperationID, func(op *Operation) {
			op.State = StateFailed
			op.Error = err.Error()
			op.Logs = append(op.Logs, "failed: "+err.Error())
		})
		return err
	}
	m.update(p.OperationID, func(op *Operation) {
		op.State = StateSucceeded
		op.Artifact = &artifact
		op.Logs = append(op.Logs, fmt.Sprintf("succeeded %s", artifact.Digest))
	})
	return nil
}

func (m *Manager) update(id string, fn func(*Operation)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[id]
	if !ok {
		m.log.Error("operation not found", "operation_id", id)
		return
	}
	fn(&op)
	op.UpdatedAt = time.Now().UTC()
	m.operations[id] = op
}
