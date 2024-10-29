// +kubebuilder:object:generate=true
// +groupName=warptail.exceptionerror.io
package v1

import (
	"warptail/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "warptail.exceptionerror.io", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

type ServiceConfig struct {
	Enabled bool                `yaml:"enabled" json:"enabled,omitempty"`
	Routes  []utils.RouteConfig `yaml:"routes" json:"routes"`
}

// WarpTailServiceStatus defines the observed state of WarpTailService.
type WarpTailServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WarpTailService is the Schema for the warptailservices API.
type WarpTailService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceConfig         `json:"spec,omitempty"`
	Status WarpTailServiceStatus `json:"status,omitempty"`
}

func (w *WarpTailService) ToServiceConfig() utils.ServiceConfig {
	return utils.ServiceConfig{
		Name:    w.Name,
		Enabled: w.Spec.Enabled,
		Routes:  w.Spec.Routes,
	}
}

//+kubebuilder:object:root=true

// DataList contains a list of Data
type WarpTailServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []WarpTailService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WarpTailService{}, &WarpTailServiceList{})
}
