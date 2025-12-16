package services

import (
	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/compression"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// Registry provides access to all contextd services.
// Use accessor methods to retrieve individual services.
type Registry interface {
	Checkpoint() checkpoint.Service
	Remediation() remediation.Service
	Memory() *reasoningbank.Service
	Repository() *repository.Service
	Troubleshoot() *troubleshoot.Service
	Hooks() *hooks.HookManager
	Distiller() *reasoningbank.Distiller
	Scrubber() secrets.Scrubber
	Compression() *compression.Service
	VectorStore() vectorstore.Store
}

// Options configures the registry with service instances.
type Options struct {
	Checkpoint   checkpoint.Service
	Remediation  remediation.Service
	Memory       *reasoningbank.Service
	Repository   *repository.Service
	Troubleshoot *troubleshoot.Service
	Hooks        *hooks.HookManager
	Distiller    *reasoningbank.Distiller
	Scrubber     secrets.Scrubber
	Compression  *compression.Service
	VectorStore  vectorstore.Store
}

// registry is the concrete implementation of Registry.
type registry struct {
	checkpoint   checkpoint.Service
	remediation  remediation.Service
	memory       *reasoningbank.Service
	repository   *repository.Service
	troubleshoot *troubleshoot.Service
	hooks        *hooks.HookManager
	distiller    *reasoningbank.Distiller
	scrubber     secrets.Scrubber
	compression  *compression.Service
	vectorStore  vectorstore.Store
}

// NewRegistry creates a new service registry.
func NewRegistry(opts Options) Registry {
	return &registry{
		checkpoint:   opts.Checkpoint,
		remediation:  opts.Remediation,
		memory:       opts.Memory,
		repository:   opts.Repository,
		troubleshoot: opts.Troubleshoot,
		hooks:        opts.Hooks,
		distiller:    opts.Distiller,
		scrubber:     opts.Scrubber,
		compression:  opts.Compression,
		vectorStore:  opts.VectorStore,
	}
}

func (r *registry) Checkpoint() checkpoint.Service       { return r.checkpoint }
func (r *registry) Remediation() remediation.Service     { return r.remediation }
func (r *registry) Memory() *reasoningbank.Service       { return r.memory }
func (r *registry) Repository() *repository.Service      { return r.repository }
func (r *registry) Troubleshoot() *troubleshoot.Service  { return r.troubleshoot }
func (r *registry) Hooks() *hooks.HookManager            { return r.hooks }
func (r *registry) Distiller() *reasoningbank.Distiller  { return r.distiller }
func (r *registry) Scrubber() secrets.Scrubber           { return r.scrubber }
func (r *registry) Compression() *compression.Service    { return r.compression }
func (r *registry) VectorStore() vectorstore.Store       { return r.vectorStore }
