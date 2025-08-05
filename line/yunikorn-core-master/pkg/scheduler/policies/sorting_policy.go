/*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package policies

import (
	"fmt"

	"github.com/apache/yunikorn-core/pkg/log"
	
	"math"	// 新增這個library(?)
	"math/rand"
	"time"	// 給seed的亂數用
)

// Sort type for queues & apps.
type SortPolicy int
// 新增saPolicy struct
type saPolicy struct{
	initialTemp float64		// 老師說要改專有名詞
	coolingRate float64
	seed int64		// 你哪位
}	

const (
	FifoSortPolicy             SortPolicy = iota // first in first out, submit time
	FairSortPolicy                               // fair based on usage
	SimulatedAnnealingPolicy					 // 新增退火演算法
	deprecatedStateAwarePolicy                   // deprecated: now alias for FIFO
	Undefined                                    // not initialised or parsing failed
)

func (s SortPolicy) String() string {
	return [...]string{"fifo", "fair", "sa", "stateaware", "undefined"}[s]		// 新增sa
}

// 新增saPolicy的參數寫入
func NewSAPolicy() *saPolicy {
	return &saPolicy {
		initialTemp: 100.0,
		coolingRate: 0.95,
		seed:  time.Now().UnixNano(), // 目前是亂數，可用固定值
	}
}

func SortPolicyFromString(str string) (SortPolicy, error) {
	switch str {
	// fifo is the default policy when not set
	case FifoSortPolicy.String(), "":
		return FifoSortPolicy, nil
	case FairSortPolicy.String():
		return FairSortPolicy, nil
	// 新增sa
	case SimulatedAnnealingPolicy.String():  
		log.Log(log.Scheduling).Info("Using Simulated Annealing scheduling policy")  // 顯示目前策略 
    	return SimulatedAnnealingPolicy, nil		
	case deprecatedStateAwarePolicy.String():
		log.Log(log.Deprecation).Warn("Sort policy 'stateaware' is deprecated; using 'fifo' instead")
		return FifoSortPolicy, nil
	default:
		return Undefined, fmt.Errorf("undefined policy: %s", str)
	}
}