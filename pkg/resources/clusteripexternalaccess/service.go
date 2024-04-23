// Copyright © 2020 Cisco Systems, Inc. and/or its affiliates
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

package clusteripexternalaccess

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	apiutil "github.com/banzaicloud/koperator/api/util"
	"github.com/banzaicloud/koperator/api/v1beta1"
	"github.com/banzaicloud/koperator/pkg/resources/templates"
	"github.com/banzaicloud/koperator/pkg/util"
	contourutils "github.com/banzaicloud/koperator/pkg/util/contour"
	"github.com/banzaicloud/koperator/pkg/util/kafka"
	contour "github.com/projectcontour/contour/apis/projectcontour/v1"
)

// TODO handle deletion gracefully from status
func (r *Reconciler) brokerService(_ logr.Logger, id int32,
	extListener v1beta1.ExternalListenerConfig) runtime.Object {

	service := &corev1.Service{
		ObjectMeta: templates.ObjectMetaWithAnnotations(
			fmt.Sprintf(kafka.NodePortServiceTemplate, r.KafkaCluster.GetName(), id, extListener.Name),
			apiutil.MergeLabels(apiutil.LabelsForKafka(r.KafkaCluster.Name), map[string]string{v1beta1.BrokerIdLabelKey: fmt.Sprintf("%d", id)}),
			extListener.GetServiceAnnotations(), r.KafkaCluster),
		Spec: corev1.ServiceSpec{
			Selector: apiutil.MergeLabels(apiutil.LabelsForKafka(r.KafkaCluster.Name),
				map[string]string{v1beta1.BrokerIdLabelKey: fmt.Sprintf("%d", id)}),
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       fmt.Sprintf("broker-%d", id),
				Port:       extListener.ContainerPort,
				TargetPort: intstr.FromInt(int(extListener.ContainerPort)),
				Protocol:   corev1.ProtocolTCP,
			},
			},
			ExternalTrafficPolicy: extListener.ExternalTrafficPolicy,
		},
	}

	return service
}

func (r *Reconciler) clusterService(log logr.Logger, extListener v1beta1.ExternalListenerConfig,
	ingressConfig v1beta1.IngressConfig, ingressConfigName, defaultIngressConfigName string) runtime.Object {

	var serviceName string = util.GenerateEnvoyResourceName(contourutils.ContourServiceName, contourutils.ContourServiceNameWithScope,
		extListener, ingressConfig, ingressConfigName, r.KafkaCluster.GetName())

	service := &corev1.Service{
		ObjectMeta: templates.ObjectMetaWithAnnotations(
			serviceName,
			apiutil.LabelsForKafka(r.KafkaCluster.Name),
			extListener.GetServiceAnnotations(), r.KafkaCluster),
		Spec: corev1.ServiceSpec{
			Selector: apiutil.MergeLabels(apiutil.LabelsForKafka(r.KafkaCluster.Name)),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       "tcp-all-broker",
				Port:       *extListener.AnyCastPort,
				TargetPort: intstr.FromInt(int(*extListener.AnyCastPort)),
				Protocol:   corev1.ProtocolTCP,
			},
			},
			ExternalTrafficPolicy: extListener.ExternalTrafficPolicy,
		},
	}

	return service
}

// generate ingressroute resource based on status and listener name
func (r *Reconciler) ingressRoute(log logr.Logger, status v1beta1.ListenerStatus, listenerName string, id int32) runtime.Object {

	address := status.Address
	fqdn := strings.Split(address, ":")[0]
	port := strings.Split(address, ":")[1]

	portInt, _ := strconv.Atoi(port)
	ingressRoute := &contour.HTTPProxy{
		ObjectMeta: templates.ObjectMeta(fqdn,
			apiutil.LabelsForKafka(r.KafkaCluster.Name), r.KafkaCluster),
		Spec: contour.HTTPProxySpec{
			VirtualHost: &contour.VirtualHost{
				Fqdn: fqdn,
			},
			TCPProxy: &contour.TCPProxy{
				Services: []contour.Service{{
					Name: fmt.Sprintf(kafka.NodePortServiceTemplate, r.KafkaCluster.GetName(), id, listenerName),
					Port: portInt,
				}},
			},
		},
	}

	return ingressRoute
}
