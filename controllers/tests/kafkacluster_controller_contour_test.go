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

package tests

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/banzaicloud/koperator/api/v1beta1"
	"github.com/banzaicloud/koperator/pkg/util"
	contourutils "github.com/banzaicloud/koperator/pkg/util/contour"
	"github.com/banzaicloud/koperator/pkg/util/kafka"
)

var _ = Describe("KafkaClusterWithContourIngressController", func() {
	var (
		count        uint64 = 0
		namespace    string
		namespaceObj *corev1.Namespace
		kafkaCluster *v1beta1.KafkaCluster
	)

	BeforeEach(func() {
		atomic.AddUint64(&count, 1)
		namespace = fmt.Sprintf("kafkacontourtest-%v", count)
		namespaceObj = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}

		kafkaCluster = createMinimalKafkaClusterCR(fmt.Sprintf("kafkacluster-%d", count), namespace)
		kafkaCluster.Spec.ListenersConfig.ExternalListeners[0].HostnameOverride = "kafka.cluster.local"
		kafkaCluster.Spec.IngressController = "contour"
		contourListener := kafkaCluster.Spec.ListenersConfig.ExternalListeners[0]
		contourListener.AccessMethod = corev1.ServiceTypeClusterIP
		contourListener.ExternalStartingPort = -1
		contourListener.AnyCastPort = util.Int32Pointer(8443)
		contourListener.Type = "plaintext"
		contourListener.Name = "listener1"
		contourListener.Config = &v1beta1.Config{

			DefaultIngressConfig: "",
			IngressConfig: map[string]v1beta1.IngressConfig{
				"listener1-config1": {
					ContourIngressConfig: &v1beta1.ContourIngressConfig{
						TLSSecretName:      "test-tls-secret",
						BrokerFQDNTemplate: "broker-%d.kafka.cluster.local",
					},
				},
			},
		}

		kafkaCluster.Spec.ListenersConfig.ExternalListeners[0] = contourListener

	})
	JustBeforeEach(func(ctx SpecContext) {
		By("creating namespace " + namespace)
		err := k8sClient.Create(ctx, namespaceObj)
		Expect(err).NotTo(HaveOccurred())

		By("creating kafka cluster object " + kafkaCluster.Name + " in namespace " + namespace)
		err = k8sClient.Create(ctx, kafkaCluster)
		Expect(err).NotTo(HaveOccurred())

		waitForClusterRunningState(ctx, kafkaCluster, namespace)
	})
	JustAfterEach(func(ctx SpecContext) {
		By("deleting Kafka cluster object " + kafkaCluster.Name + " in namespace " + namespace)
		err := k8sClient.Delete(ctx, kafkaCluster)
		Expect(err).NotTo(HaveOccurred())

		kafkaCluster = nil
	})
	When("configuring Contour ingress expect broker ClusterIp svc", func() {
		BeforeEach(func() {
			kafkaCluster.Spec.Brokers[0].BrokerConfig = &v1beta1.BrokerConfig{BrokerIngressMapping: []string{"listener1"}}
			kafkaCluster.Spec.Brokers[1].BrokerConfig = &v1beta1.BrokerConfig{BrokerIngressMapping: []string{"listener1"}}
		})
		It("should reconcile object properly", func(ctx SpecContext) {
			// TODO: implement
			expectContour(ctx, kafkaCluster)
		})
	})
})

func expectContourClusterIpAnycastSvc(ctx context.Context, kafkaCluster *v1beta1.KafkaCluster, eListener v1beta1.ExternalListenerConfig) {

	var log logr.Logger
	var svc corev1.Service
	ingressConfigs, defaultControllerName, err := util.GetIngressConfigs(kafkaCluster.Spec, eListener)
	Expect(err).NotTo(HaveOccurred())

	for name, ingressConfig := range ingressConfigs {
		if !util.IsIngressConfigInUse(name, defaultControllerName, kafkaCluster, log) {
			continue
		}
		serviceName := util.GenerateEnvoyResourceName(contourutils.ContourServiceName, contourutils.ContourServiceNameWithScope,
			eListener, ingressConfig, name, kafkaCluster.GetName())
		Eventually(ctx, func() error {
			err := k8sClient.Get(ctx, types.NamespacedName{Namespace: kafkaCluster.Namespace, Name: serviceName}, &svc)
			return err
		}).Should(Succeed())
		Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		Expect(svc.Spec.Ports).To(HaveLen(1))
		Expect(svc.Spec.Ports[0].Port).To(Equal(*eListener.AnyCastPort))
		Expect(svc.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(int(eListener.ContainerPort))))
		Expect(svc.Spec.Ports[0].Name).To(Equal("tcp-all-broker"))
		Expect(svc.Spec.Selector).To(HaveKeyWithValue("app", "kafka"))
		Expect(svc.Spec.Selector).To(HaveKeyWithValue("kafka_cr", kafkaCluster.GetName()))
	}
}

func expectContourClusterIpBrokerSvc(ctx context.Context, kafkaCluster *v1beta1.KafkaCluster, eListener v1beta1.ExternalListenerConfig) {
	var log logr.Logger
	var svc corev1.Service
	ingressConfigs, defaultControllerName, err := util.GetIngressConfigs(kafkaCluster.Spec, eListener)
	Expect(err).NotTo(HaveOccurred())

	for name, _ := range ingressConfigs {
		if !util.IsIngressConfigInUse(name, defaultControllerName, kafkaCluster, log) {
			continue
		}
		for _, broker := range kafkaCluster.Spec.Brokers {
			serviceName := fmt.Sprintf(kafka.NodePortServiceTemplate, kafkaCluster.GetName(), broker.Id, eListener.Name)
			Eventually(ctx, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: kafkaCluster.Namespace, Name: serviceName}, &svc)
				return err
			}).Should(Succeed())
			Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(*eListener.AnyCastPort))
			Expect(svc.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(int(eListener.ContainerPort))))
			Expect(svc.Spec.Ports[0].Name).To(Equal(fmt.Sprintf("broker-%d", broker.Id)))
			Expect(svc.Spec.Selector).To(HaveKeyWithValue("app", "kafka"))
			Expect(svc.Spec.Selector).To(HaveKeyWithValue(v1beta1.BrokerIdLabelKey, fmt.Sprintf("%d", broker.Id)))
			Expect(svc.Spec.Selector).To(HaveKeyWithValue("kafka_cr", kafkaCluster.GetName()))
		}
	}
}

func expectContourHttpProxy(ctx context.Context, kafkaCluster *v1beta1.KafkaCluster, eListener v1beta1.ExternalListenerConfig) {
	Expect(BeTrue().Match(false))
}

func expectContour(ctx context.Context, kafkaCluster *v1beta1.KafkaCluster) {
	for _, eListenerName := range kafkaCluster.Spec.ListenersConfig.ExternalListeners {
		expectContourClusterIpAnycastSvc(ctx, kafkaCluster, eListenerName)
		expectContourClusterIpBrokerSvc(ctx, kafkaCluster, eListenerName)
		expectContourHttpProxy(ctx, kafkaCluster, eListenerName)
	}
}
