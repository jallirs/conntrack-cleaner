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
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"time"
    "errors"
	"k8s.io/klog"
)

var (
	sourceIPStr        = "src="
	destinationIPStr   = "dst="
	sourcePortStr      = "sport="
	destinationPortStr = "dport="
)

type conntrackCleaner struct {
	tableDumpFrequency   time.Duration
	connRenewalThreshold int
	connPurgeThreshold   time.Duration
	ciChannel            chan connectionInfo
	connectionMap        map[string]connectionInfoStore
	oldConnectionMap     map[string]connectionInfoStore
	isUdpEnabled         bool
	isTcpEnabled         bool
	oldTableInitialized  bool
}

type connectionInfo struct {
	expiryTime      int
	sourceIP        string
	destinationIP   string
	sourcePort      string
	destinationPort string
	protocol        string
}

func newConntrackCleaner(frequency time.Duration, threshold int,  udpEnabled bool, tcpEnabled bool, connPurgeThresholdValue time.Duration) *conntrackCleaner {
	return &conntrackCleaner{
		connPurgeThreshold:   connPurgeThresholdValue,
		tableDumpFrequency:   frequency,
		connRenewalThreshold: threshold,
		ciChannel:            make(chan connectionInfo),
		connectionMap:        make(map[string]connectionInfoStore),
		oldConnectionMap:     make(map[string]connectionInfoStore),
		isUdpEnabled:         udpEnabled,
		isTcpEnabled:         tcpEnabled,
	    oldTableInitialized:  false}
}

func extractConnInfo(parsedEntry []string, entryLen int) (*connectionInfo, error) {
	isUdp := contains(parsedEntry, "udp")
	isTcp := contains(parsedEntry, "tcp")
	var isSynSent bool
	if isTcp == true {
		isSynSent = contains(parsedEntry, "SYN_SENT")
	}
	//klog.Errorf("parsed entry 2 is : %v isUDP is %v", parsedEntry[2], strconv.FormatBool(isUdp))
	//for index, entry := range parsedEntry {
	//    klog.Errorf("parsed entry %v is : %v", index, entry)
	//}
	expTime, err := strconv.Atoi(parsedEntry[7])
	if err != nil {
		return nil, err
	}
	// [udp      17 13 src=0.0.0.0 dst=255.255.255.255 sport=68 dport=67 [UNREPLIED] src=255.255.255.255 dst=0.0.0.0 sport=67 dport=68 mark=0 use=1]
    // [tcp      6 86 SYN_SENT src=10.163.68.59 dst=10.163.221.95 sport=55162 dport=14250 [UNREPLIED] src=10.163.221.95 dst=10.163.68.59 sport=14250 dport=55162 mark=0 use=1]

	//klog.Errorf("string is : %v", parsedEntry)
	if isUdp == true {
		return &connectionInfo{
			protocol:        "udp",
			expiryTime:      expTime,
			sourceIP:        strings.Split(parsedEntry[8], sourceIPStr)[1],
			destinationIP:   strings.Split(parsedEntry[9], destinationIPStr)[1],
			sourcePort:      strings.Split(parsedEntry[10], sourcePortStr)[1],
			destinationPort: strings.Split(parsedEntry[11], destinationPortStr)[1],
		}, nil
	}
	if isTcp == true {
		if isSynSent == true {
			return &connectionInfo{
				protocol:        "tcp",
				expiryTime:      expTime,
				sourceIP:        strings.Split(parsedEntry[9], sourceIPStr)[1],
				destinationIP:   strings.Split(parsedEntry[10], destinationIPStr)[1],
				sourcePort:      strings.Split(parsedEntry[11], sourcePortStr)[1],
				destinationPort: strings.Split(parsedEntry[12], destinationPortStr)[1],
			}, nil
		}}
      return &connectionInfo{
		protocol:        "unknownprotocol",
		expiryTime:      expTime,
		sourceIP:        strings.Split(parsedEntry[9], sourceIPStr)[1],
		destinationIP:   strings.Split(parsedEntry[10], destinationIPStr)[1],
		sourcePort:      strings.Split(parsedEntry[11], sourcePortStr)[1],
		destinationPort: strings.Split(parsedEntry[12], destinationPortStr)[1],
	}, errors.New("not a tcp or udp connection")
	
}

func contains(s []string, str string) bool {
	for _, v := range s {
	  if v == str {
		return true
	  }
	}
	return false
  }

func parseConntrackEntry(entry string) []string {
	return strings.Split(entry, " ")
}

func parseConntrackTable(table string) []string {
	return strings.Split(table, "\n")
}

// Copy values from one map to the other
func copyMap() {

}

func (c *conntrackCleaner) cleanNCopyConntrackTable() {
    // First run of the program, value is false
	if c.oldTableInitialized == false {
		for key, value := range c.connectionMap  {
			c.oldConnectionMap[key] = value
		  }
		c.oldTableInitialized = true
		return
	}
	// Every other time, we should have an old table
	// Check to see if old entries are in new map or not
	else {
		currentTime := time.Now()
		for oldKey := range c.oldConnectionMap  {
			value, ok := c.connectionMap[oldKey]
			if !ok {
				// Eviction based on new table for values that might disappear between iterations
				// Key is not in new dump, remove it from the old map
				delete(c.oldConnectionMap, key)
			}
			else if (ok) && (value.firstSeen.Sub(currentTime).Seconds() >= c.connPurgeThreshold ) {
			    // Time based eviction
			   // Remove old values that are beyond this time anyway
			   delete(c.connectionMap, key)
			   delete(c.oldConnectionMap, key)
			}  
	
		  }
		  // Move New Conntrack Table to Old Conntrack Table
		  // Todo: Functionalize this
		  for key, value := range c.connectionMap  {
			c.oldConnectionMap[key] = value
		  }

	}

}

func (c *conntrackCleaner) processConntrackTable(table *bytes.Buffer) {
	entryList := parseConntrackTable(table.String())
	for _, entry := range entryList {
		if len(entry) != 0 {
			entrylength := len(entry)
			parsedEntry := parseConntrackEntry(entry)
			connInfo, err := extractConnInfo(parsedEntry, entrylength)
			if err != nil {
				klog.Errorf("error extracting connection info : %v", err)
				continue
			}
			c.ciChannel <- *connInfo
		}
	}
	c.cleanNCopyConntrackTable()
}

func executeCmd(output *bytes.Buffer, udpEnabled bool, tcpEnabled bool) error {
	var err error
	var tcpConnList *exec.Cmd
	if (udpEnabled) && (tcpEnabled) {
		//both enabled
		tcpConnList = exec.Command("conntrack", "-L")
	} else if (udpEnabled) && !(tcpEnabled) {
		//only udp enabled
		tcpConnList = exec.Command("conntrack", "-L", "-p", "udp")
	} else if !(udpEnabled) && (tcpEnabled) {
		//only tcp enabled
		tcpConnList = exec.Command("conntrack", "-L", "-p", "tcp")
	}
	//tcpConnList := exec.Command("conntrack", "-L")
	grep := exec.Command("grep", "UNREPLIED")
	//grepMinusE := exec.Command("grep", "-E", "tcp|udp")
	grep.Stdin, err = tcpConnList.StdoutPipe()
	if err != nil {
		return err
	}


	grep.Stdout = output
	// Start the grep command first. (The order will be last command first)
	grep.Start()
	tcpConnList.Run()
	grep.Wait()
	return nil
}

func (c *conntrackCleaner) runConntrackTableDump() {
	//Periodically take dump of conntrack table.
	for {
		func() {
			defer time.Sleep(c.tableDumpFrequency)
			var output bytes.Buffer
			err := executeCmd(&output, c.isUdpEnabled, c.isTcpEnabled)
			if err != nil {
				klog.Errorf("error executing conntrack cmd : %v", err)
				return
			}
			if output.Len() != 0 {
				c.processConntrackTable(&output)
			}
		}()
	}
}
