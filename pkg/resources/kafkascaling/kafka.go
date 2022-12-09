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

package kafkascaling

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/koperator/api/v1beta1"
	"github.com/banzaicloud/koperator/pkg/k8sutil"
	"github.com/banzaicloud/koperator/pkg/resources"
)

const (
	componentName = "kafkascaling"
)

type Reconciler struct {
	resources.Reconciler
}

type ExternalBrokers struct {
	Brokers []v1beta1.Broker `json:"brokers"`
}

// New creates a new reconciler for Kafka
func New(client client.Client, directClient client.Reader, cluster *v1beta1.KafkaCluster) *Reconciler {
	return &Reconciler{
		Reconciler: resources.Reconciler{
			Client:       client,
			DirectClient: directClient,
			KafkaCluster: cluster,
		},
	}
}

// Reconcile implements the reconcile logic for Kafka
//
//gocyclo:ignore
//nolint:funlen
func (r *Reconciler) Reconcile(log logr.Logger) error {
	log = log.WithValues("component", componentName, "clusterName", r.KafkaCluster.Name, "clusterNamespace", r.KafkaCluster.Namespace)

	log.Info("Reconciling")

	ctx := context.Background()

	// read configmap
	key := types.NamespacedName{
		Name:      "kafkacluster-extension",
		Namespace: r.KafkaCluster.Namespace,
	}
	instanceExtension := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, key, instanceExtension)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "extension configmap not found")
	}

	extBrokers := ExternalBrokers{}
	if err := yaml.Unmarshal([]byte(instanceExtension.Data["kafkaClusterExtention"]), &extBrokers); err != nil {
		log.Error(err, "Config map data not in yaml format")
	}
	log.Info("Configmap content:" + fmt.Sprint(extBrokers))
	// merge configmap with kafka cr
	log.Info("Looking for new brokers to add")
	for _, b := range extBrokers.Brokers {
		found := false
		for _, tmp := range r.KafkaCluster.Spec.Brokers {
			if b.Id == tmp.Id {
				found = true
				break
			}
		}
		if !found {
			log.Info("Adding new broker " + fmt.Sprint(b.Id))
			err := k8sutil.AddNewBrokerToCr(b, r.KafkaCluster.Name, r.KafkaCluster.Namespace, r.Client)
			if err != nil {
				log.Error(err, "Failed to add broker id to kafka cr: "+fmt.Sprint(b.Id))
			}
		}
	}
	log.Info("Looking for old brokers to remove")
	for _, b := range r.KafkaCluster.Spec.Brokers {
		// brokers with ids grater than 1000 are ephemeral
		if b.Id/1000 > 0 {
			found := false
			for _, tmp := range extBrokers.Brokers {
				if b.Id == tmp.Id {
					found = true
					break
				}
			}
			if !found {
				log.Info("Removing old broker " + fmt.Sprint(b.Id))
				err = k8sutil.RemoveBrokerFromCr(fmt.Sprint(b.Id), r.KafkaCluster.Name, r.KafkaCluster.Namespace, r.Client)
				if err != nil {
					log.Error(err, "Failed to remove ephemeral broker id from cr: "+fmt.Sprint(b.Id))
				}
			}
		}
	}
	log.Info("Kafka scaled")

	return nil
}
