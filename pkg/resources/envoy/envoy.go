// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"fmt"

	"github.com/banzaicloud/kafka-operator/api/v1beta1"
	"github.com/banzaicloud/kafka-operator/pkg/k8sutil"
	"github.com/banzaicloud/kafka-operator/pkg/resources"
	"github.com/banzaicloud/kafka-operator/pkg/util"
	envoyutils "github.com/banzaicloud/kafka-operator/pkg/util/envoy"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	componentName            = "envoy"
	envoyVolumeAndConfigName = "envoy-config"
	envoyDeploymentName      = "envoy"
	envoyGlobal              = "envoy-global"
)

func labelSelector(envoyConfig *v1beta1.EnvoyConfig) map[string]string {
	if envoyConfig.Id == envoyGlobal {
		return map[string]string{"app": componentName,}
	} else {
		return map[string]string{"app": fmt.Sprintf("%s-%s", componentName, envoyConfig.Id),}
	}
}

// Reconciler implements the Component Reconciler
type Reconciler struct {
	resources.Reconciler
}

// New creates a new reconciler for Envoy
func New(client client.Client, cluster *v1beta1.KafkaCluster) *Reconciler {
	return &Reconciler{
		Reconciler: resources.Reconciler{
			Client:       client,
			KafkaCluster: cluster,
		},
	}
}

func (r *Reconciler) getObjects(log logr.Logger, configId string, brokerConfig v1beta1.BrokerConfig) []runtime.Object {
	var objects []runtime.Object
	for _, res := range []resources.ResourceWithLogsAndEnvoyConfig{
		r.configMap,
		r.deployment,
	} {
		objects = append(objects, res(log, util.GetEnvoyConfig(configId, brokerConfig, r.KafkaCluster.Spec)))
	}
	return objects
}

func (r *Reconciler) getGlobalObjects(log logr.Logger) []runtime.Object {
	return r.getObjects(log, envoyGlobal, v1beta1.BrokerConfig{})
}

func (r *Reconciler) getBrokerConfigSpecificObjects(log logr.Logger) []runtime.Object {
	var objects []runtime.Object
	for configId, brokerConfig := range r.KafkaCluster.Spec.BrokerConfigGroups {
		objects = append(objects, r.getObjects(log, configId, brokerConfig)...)
	}
	return objects
}

// Reconcile implements the reconcile logic for Envoy
func (r *Reconciler) Reconcile(log logr.Logger) error {
	log = log.WithValues("component", componentName)

	log.V(1).Info("Reconciling")
	if r.KafkaCluster.Spec.ListenersConfig.ExternalListeners != nil && r.KafkaCluster.Spec.GetIngressController() == envoyutils.IngressControllerName {
		var objectsMarkedForDelete []runtime.Object
		var objectsMarkedForReconcile []runtime.Object
		if r.KafkaCluster.Spec.EnvoyConfig.EnvoyPerBrokerGroup {
			objectsMarkedForDelete = r.getGlobalObjects(log)
			objectsMarkedForReconcile = r.getBrokerConfigSpecificObjects(log)
		} else {
			objectsMarkedForDelete = r.getBrokerConfigSpecificObjects(log)
			objectsMarkedForReconcile = r.getGlobalObjects(log)
		}

		if !r.KafkaCluster.Spec.EnvoyConfig.BringYourOwnLB {
			objectsMarkedForReconcile = append(objectsMarkedForReconcile, r.loadBalancer(log))
		} else {
			objectsMarkedForDelete = append(objectsMarkedForDelete, r.loadBalancer(log))
		}

		for _, o := range objectsMarkedForReconcile {
			err := k8sutil.Reconcile(log, r.Client, o, r.KafkaCluster)
			if err != nil {
				return err
			}
		}
		for _, o := range objectsMarkedForDelete {
			err := k8sutil.Delete(log, r.Client, o)
			if err != nil {
				return err
			}
		}
	}

	log.V(1).Info("Reconciled")

	return nil
}
