/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ComponentTemplateSpec defines the desired state of ComponentTemplate
// +k8s:openapi-gen=true
type # Role mapping

You can use the custom resource `RoleMapping` to manage the role mapping inside Elasticsearch.

## Properties

You can use the following properties:
- **elasticsearchRef** (object): The Elasticsearch cluster ref
  - **managed** (object): Use it if cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where operator connect on. Default is used all node groups.
  - **external** (object): Use it if cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on cluster
  - **secretRef** (object): The secret ref that store the credentials to connect on Elasticsearch. It need to contain the keys `username` and `password`. It only used for external Elasticsearch.
    - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on Elasticsearch cluster.
    - **name** (string / require): The secret name
- **name** (string): The role mapping name. Default it use the resource name.
- **enabled** (boolean): Set to true to enable the role mapping. Default to `true`.
- **roles** (slice of string / require): The list of role. Default to empty.
- **rules** (string / require): The rules on JSON format.
- **metadata** (string): The metadata on JSON format. Default to empty

## Sample With managed Elasticsearch

In this sample, we will create role mapping on managed Elasticseach.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: RoleMapping
metadata:
  name: admins
  namespace: cluster-dev
spec:
  enabled: true
  roles:
    - superuser
    - admin
  rules: |
    {
      "any": [
          {
              "field": {
                "groups": "CN=ADMINS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          },
          {
              "field": {
                "groups": "CN=SUPPORTS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          }
      ]
    }
  elasticsearchRef:
    managed:
      name: elasticsearch
```

## Sample With external Elasticsearch

In this sample, we will create role mapping on external Elasticsearch.

**role.yml**:
```yaml
apiVersion: elasticsearchapi.k8s.webcenter.fr/v1
kind: RoleMapping
metadata:
  name: admins
  namespace: cluster-dev
spec:
  enabled: true
  roles:
    - superuser
    - admin
  rules: |
    {
      "any": [
          {
              "field": {
                "groups": "CN=ADMINS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          },
          {
              "field": {
                "groups": "CN=SUPPORTS,OU=Elastic,DC=DOMAIN,DC=COM"
              }
          }
      ]
    }
  elasticsearchRef:
    external:
      addresses:
        - https://elasticsearch-cluster-dev.domain.local
    secretRef:
      name: elasticsearch-credentials
    elasticsearchCASecretRef:
      name: custom-ca-elasticsearch
```

**custom-ca-elasticsearch-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca-elasticsearch
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**elasticsearch-credentials.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-credentials
  namespace: cluster-dev
type: Opaque
data:
  username: ++++++++
  password: ++++++++
```Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ElasticsearchRef is the Elasticsearch ref to connect on.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ElasticsearchRef shared.ElasticsearchRef `json:"elasticsearchRef"`

	// Name is the custom component template name
	// If empty, it use the ressource name
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Name string `json:"name,omitempty"`

	// Settings is the component setting
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Settings string `json:"settings,omitempty"`

	// Mappings is the component mapping
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Mappings string `json:"mappings,omitempty"`

	// Aliases is the component aliases
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Aliases string `json:"aliases,omitempty"`

	// Template is the raw template
	// You can use it instead to set settings, mappings or aliases
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +optional
	Template string `json:"rawTemplate,omitempty"`
}

// ComponentTemplateStatus defines the observed state of ComponentTemplate
type ComponentTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	apis.BasicRemoteObjectStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// ComponentTemplate is the Schema for the componenttemplates API
// +operator-sdk:csv:customresourcedefinitions:resources={{None,None,None}}
// +kubebuilder:printcolumn:name="Sync",type="boolean",JSONPath=".status.isSync"
// +kubebuilder:printcolumn:name="Error",type="boolean",JSONPath=".status.isOnError",description="Is on error"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="health"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ComponentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentTemplateSpec   `json:"spec,omitempty"`
	Status ComponentTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ComponentTemplateList contains a list of ComponentTemplate
type ComponentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ComponentTemplate{}, &ComponentTemplateList{})
}
