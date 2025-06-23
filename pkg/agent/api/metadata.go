package api

const (
	Group   = "cloudassistant.io"
	Version = "v1alpha1"
)

// N.B. We need to redefine Metadata and not reuse the version in the K8s libraries
// because we want it to have yaml tags so we can serialize with the YAML library.

// Metadata holds an optional name of the project.
type Metadata struct {
	Name        string            `yaml:"name,omitempty" json:"name,omitempty"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	// ResourceVersion is used for optimistic concurrency.
	// Ref: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
	// This should be treated as an opaque value by clients.
	ResourceVersion string `yaml:"resourceVersion,omitempty" json:"resourceVersion,omitempty"`
}
