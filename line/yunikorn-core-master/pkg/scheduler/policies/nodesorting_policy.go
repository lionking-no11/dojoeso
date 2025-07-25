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
)

type SortingPolicy int

const (
	BinPackingPolicy SortingPolicy = iota // 從0開始自動遞增 -> 0
	FairnessPolicy // -> 1
  SimulatedAnnealingPolicy // -> 2
)

func (nsp SortingPolicy) String() string {
	return [...]string{"binpacking", "fair", "sa"}[nsp] /* 新增sa */
}

func SortingPolicyFromString(str string) (SortingPolicy, error) {
	switch str {
	// fair is the default policy when not set
	case FairnessPolicy.String(), "":
		return FairnessPolicy, nil
	case BinPackingPolicy.String():
		return BinPackingPolicy, nil
  // 新增sa的case
  case SimulatedAnnealingPolicy.String():
    return SimulatedAnnealingPolicy, nil
	default:
		return FairnessPolicy, fmt.Errorf("undefined policy: %s", str)
	}
}
