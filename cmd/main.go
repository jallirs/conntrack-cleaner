/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"os"
	"strconv"
	"time"

	"k8s.io/klog"
)

func main() {
	cleaner := newConntrackCleaner(getConntrackDumpFrequency(), getConntrackPurgeThreshold(), getThreshold(), getUdpCleaning(), getTcpCleaning())
	go cleaner.runConntrackTableDump()
	cleaner.runConnCleaner()
}

func getConntrackDumpFrequency() time.Duration {
	defaultDumpFrequency := time.Duration(1) * time.Second
	frequency, ok := os.LookupEnv("CONNTRACK_TABLE_DUMP_FREQUENCY")
	if !ok {
		klog.Warning("CONNTRACK_TABLE_DUMP_FREQUENCY env variable not set in podspec. Taking default value as 1sec")
		return defaultDumpFrequency
	}
	configuredDumpFrequency, err := time.ParseDuration(frequency)
	if err != nil {
		klog.Warning("invalid value given for CONNTRACK_TABLE_DUMP_FREQUENCY in podspec. Taking default value as 1sec")
		return defaultDumpFrequency
	}
	return configuredDumpFrequency
}
func getConntrackPurgeThreshold() time.Duration {
	defaultPurgeThreshold := time.Duration(60) * time.Second
	purgeThreshold, ok := os.LookupEnv("CONNTRACK_PURGE_THRESHOLD")
	if !ok {
		klog.Warning("CONNTRACK_PURGE_THRESHOLD env variable not set in podspec. Taking default value as 60 sec")
		return defaultPurgeThreshold
	}
	configuredPurgeThreshold, err := time.ParseDuration(purgeThreshold)
	if err != nil {
		klog.Warning("invalid value given for CONNTRACK_TABLE_DUMP_FREQUENCY in podspec. Taking default value as 1sec")
		return defaultPurgeThreshold
	}
	return configuredPurgeThreshold
}

func getThreshold() int {
	defaultThreshold := 0
	threshold, ok := os.LookupEnv("CONNECTION_RENEWAL_THRESHOLD")
	if !ok {
		klog.Warning("CONNECTION_RENEWAL_THRESHOLD env variable not set in podspec. Taking default value as 0")
		return defaultThreshold
	}
	configuredThreshold, err := strconv.Atoi(threshold)
	if err != nil {
		klog.Warning("invalid value given for CONNECTION_RENEWAL_THRESHOLD in podspec. Taking default value as 0")
		return defaultThreshold
	}
	return configuredThreshold
}

func getTcpCleaning() bool {
	defaultTcpCleaningEnabled := false
	tcpEnabled, ok := os.LookupEnv("TCP_CLEANING_ENABLED")
	if !ok {
		klog.Warning("TCP_CLEANING_ENABLED env variable not set in podspec. Taking default value as false")
		return defaultTcpCleaningEnabled
	}
	configuredTcpCleaning, err := strconv.ParseBool(tcpEnabled)
	if err != nil {
		klog.Warning("invalid value given for TCP_CLEANING_ENABLED in podspec. Taking default value as false")
		return defaultTcpCleaningEnabled
	}
	return configuredTcpCleaning
}

func getUdpCleaning() bool {
	defaultUdpCleaningEnabled := true
	udpEnabled, ok := os.LookupEnv("UDP_CLEANING_ENABLED")
	if !ok {
		klog.Warning("UDP_CLEANING_ENABLED env variable not set in podspec. Taking default value as true")
		return defaultUdpCleaningEnabled
	}
	configuredUdpCleaning, err := strconv.ParseBool(udpEnabled)
	if err != nil {
		klog.Warning("invalid value given for UDP_CLEANING_ENABLED in podspec. Taking default value as true")
		return defaultUdpCleaningEnabled
	}
	return configuredUdpCleaning
}
