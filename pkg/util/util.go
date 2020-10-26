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

package util

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/go-logr/logr"
	"github.com/imdario/mergo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8s_zap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/banzaicloud/kafka-operator/api/v1beta1"
)

const (
	symbolSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

// IntstrPointer generate IntOrString pointer from int
func IntstrPointer(i int) *intstr.IntOrString {
	is := intstr.FromInt(i)
	return &is
}

// Int64Pointer generates int64 pointer from int64
func Int64Pointer(i int64) *int64 {
	return &i
}

// Int32Pointer generates int32 pointer from int32
func Int32Pointer(i int32) *int32 {
	return &i
}

// BoolPointer generates bool pointer from bool
func BoolPointer(b bool) *bool {
	return &b
}

// IntPointer generates int pointer from int
func IntPointer(i int) *int {
	return &i
}

// StringPointer generates string pointer from string
func StringPointer(s string) *string {
	return &s
}

// QuantityPointer generates Quantity pointer from Quantity
func QuantityPointer(q resource.Quantity) *resource.Quantity {
	return &q
}

// MapStringStringPointer generates a map[string]*string
func MapStringStringPointer(in map[string]string) (out map[string]*string) {
	out = make(map[string]*string, 0)
	for k, v := range in {
		out[k] = StringPointer(v)
	}
	return
}

// MergeLabels merges two given labels
func MergeLabels(l ...map[string]string) map[string]string {
	res := make(map[string]string)

	for _, v := range l {
		for lKey, lValue := range v {
			res[lKey] = lValue
		}
	}
	return res
}

func MergeAnnotations(annotations ...map[string]string) map[string]string {
	rtn := make(map[string]string)
	for _, a := range annotations {
		for k, v := range a {
			rtn[k] = v
		}
	}

	return rtn
}

// ConvertStringToInt32 converts the given string to int32
func ConvertStringToInt32(s string) int32 {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return -1
	}
	return int32(i)
}

// IsSSLEnabledForInternalCommunication checks if ssl is enabled for internal communication
func IsSSLEnabledForInternalCommunication(l []v1beta1.InternalListenerConfig) (enabled bool) {

	for _, listener := range l {
		if strings.ToLower(listener.Type) == "ssl" {
			enabled = true
			break
		}
	}
	return enabled
}

// ConvertMapStringToMapStringPointer converts a simple map[string]string to map[string]*string
func ConvertMapStringToMapStringPointer(inputMap map[string]string) map[string]*string {

	result := map[string]*string{}
	for key, value := range inputMap {
		result[key] = StringPointer(value)
	}
	return result
}

// StringSliceContains returns true if list contains s
func StringSliceContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// StringSliceRemove will remove s from list
func StringSliceRemove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// ParsePropertiesFormat parses the properties format configuration into map[string]string
func ParsePropertiesFormat(properties string) map[string]string {
	kafkaProperties := strings.Split(strings.TrimSpace(properties), "\n")
	config := make(map[string]string, len(kafkaProperties))

	for i := range kafkaProperties {
		prop := kafkaProperties[i]
		kvSeparatorIdx := strings.Index(prop, "=")
		if kvSeparatorIdx >= 0 {
			config[strings.TrimSpace(prop[:kvSeparatorIdx])] = strings.TrimSpace(prop[kvSeparatorIdx+1:])
		}
	}

	return config
}

func AreStringSlicesIdentical(a, b []string) bool {
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}

func GetBrokerIdsFromStatus(brokerStatuses map[string]v1beta1.BrokerState, log logr.Logger) []int {
	brokerIds := make([]int, 0, len(brokerStatuses))
	for brokerId := range brokerStatuses {
		id, err := strconv.Atoi(brokerId)
		if err != nil {
			log.Error(err, "could not parse brokerId properly")
			continue
		}
		brokerIds = append(brokerIds, id)
	}
	sort.Ints(brokerIds)
	return brokerIds
}

func GetBrokerConfigGroupFromStatus(brokerStatuses map[string]v1beta1.BrokerState, brokerId int, log logr.Logger) string {
	stringId := strconv.Itoa(brokerId)
	if brokerStatus, ok := brokerStatuses[stringId]; ok {
		return string(brokerStatus.ConfigGroupState)
	}
	return ""
}

func GetBrokerSpecFromId(clusterSpec v1beta1.KafkaClusterSpec, brokerId int32, log logr.Logger) *v1beta1.Broker {
	for _, broker := range clusterSpec.Brokers {
		if broker.Id == brokerId {
			return &broker
		}
	}
	log.Info(fmt.Sprintf("Could not found brokerId %d in Spec. Broker ID is invalid or broker was removed from the spec", brokerId))
	return nil
}

// GetBrokerConfig compose the brokerConfig for a given broker
func GetBrokerConfig(broker v1beta1.Broker, clusterSpec v1beta1.KafkaClusterSpec) (*v1beta1.BrokerConfig, error) {

	bConfig := &v1beta1.BrokerConfig{}
	if broker.BrokerConfigGroup == "" {
		return broker.BrokerConfig, nil
	} else if broker.BrokerConfig != nil {
		bConfig = broker.BrokerConfig.DeepCopy()
	}

	err := mergo.Merge(bConfig, clusterSpec.BrokerConfigGroups[broker.BrokerConfigGroup], mergo.WithAppendSlice)
	if err != nil {
		return nil, errors.WrapIf(err, "could not merge brokerConfig with ConfigGroup")
	}

	bConfig.StorageConfigs = dedupStorageConfigs(bConfig.StorageConfigs)

	return bConfig, nil
}

func dedupStorageConfigs(elements []v1beta1.StorageConfig) []v1beta1.StorageConfig {
	encountered := make(map[string]struct{})
	result := []v1beta1.StorageConfig{}

	for _, v := range elements {
		if _, ok := encountered[v.MountPath]; !ok {
			encountered[v.MountPath] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

// GetEnvoyConfig compose the Envoy config from a specific broker config and the global config
func GetEnvoyConfig(configId string, brokerConfig v1beta1.BrokerConfig, clusterSpec v1beta1.KafkaClusterSpec) *v1beta1.EnvoyConfig {
	envoyConfig := clusterSpec.EnvoyConfig.DeepCopy()

	if envoyConfig == nil {
		return nil
	}

	envoyConfig.Id = configId

	if !clusterSpec.EnvoyConfig.EnvoyPerBrokerGroup {
		return envoyConfig
	}

	// Broker config level overrides
	if brokerConfig.Envoy != nil && brokerConfig.Envoy.Replicas > 0 {
		envoyConfig.Replicas = brokerConfig.Envoy.Replicas
	}
	if brokerConfig.NodeSelector != nil {
		envoyConfig.NodeSelector = brokerConfig.NodeSelector
	}
	if brokerConfig.NodeAffinity != nil {
		envoyConfig.NodeAffinity = brokerConfig.NodeAffinity
	}

	return envoyConfig
}

// GetBrokerImage returns the used broker image
func GetBrokerImage(brokerConfig *v1beta1.BrokerConfig, clusterImage string) string {
	if brokerConfig.Image != "" {
		return brokerConfig.Image
	}
	return clusterImage
}

// getRandomString returns a random string containing uppercase, lowercase and number characters with the length given
func GetRandomString(length int) (string, error) {
	rand.Seed(time.Now().UnixNano())

	chars := []rune(symbolSet)

	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String(), nil
}

// computes the max between 2 ints
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func CreateLogger(debug bool, development bool) logr.Logger {
	// create encoder config
	var config zapcore.EncoderConfig
	if development {
		config = zap.NewDevelopmentEncoderConfig()
	} else {
		config = zap.NewProductionEncoderConfig()
	}
	// set human readable timestamp format regardless whether development mode is on
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	// create the encoder
	var encoder zapcore.Encoder
	if development {
		encoder = zapcore.NewConsoleEncoder(config)
	} else {
		encoder = zapcore.NewJSONEncoder(config)
	}

	// set the log level
	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}

	return k8s_zap.New(k8s_zap.UseDevMode(development), k8s_zap.Encoder(encoder), k8s_zap.Level(level))
}
