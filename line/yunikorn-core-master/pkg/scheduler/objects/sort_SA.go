package objects

import (
	
	"github.com/apache/yunikorn-core/pkg/common/resources"
	
	"fmt"
	"math"
	"math/rand"
	"time"
)

/************ 基本型別 ************/

type Resource = map[string]float64

type Pod struct {
	Demand Resource
}

type Node struct {
	Capacity Resource
}

type Problem struct {
	Pods        []Pod
	Nodes       []Node
	ResourceKey []string // 要考慮的資源鍵，例如: []string{"cpu","memory"}
}

type SAParams struct {
	Iterations   int
	InitialTemp  float64
	CoolingRate  float64
	Seed         int64
	// 目標權重（可調整）
	WOveruse     float64 // 超量懲罰（>0 變大）
	WVariance    float64 // 資源利用率方差（平衡）（>0 變大）
	WActiveNodes float64 // 啟用節點數（>0 會偏向「合併、少用節點」，<0 會偏向「分散」）
}

/************ 小工具 ************/

func cloneAssignment(a []int) []int {
	b := make([]int, len(a))
	copy(b, a)
	return b
}

func zeroUsage(nNodes int, keys []string) []Resource {
	usage := make([]Resource, nNodes)
	for i := 0; i < nNodes; i++ {
		usage[i] = make(Resource)
		for _, k := range keys {
			usage[i][k] = 0
		}
	}
	return usage
}

func addRes(dst Resource, src Resource) {
	for k, v := range src {
		dst[k] += v
	}
}

func max(a, b float64) float64 {
	if a > b { return a }
	return b
}

/************ 成本函式（越小越好） ************/
/*
目標：
  1) 不超量：任何節點任何資源 used/cap > 1 會有大量懲罰（WOveruse）
  2) 負載平衡：各節點利用率的方差越小越好（WVariance）
  3) 合併/分散：依 WActiveNodes 控制。>0 會懲罰使用的節點數（鼓勵合併），<0 會獎勵使用更多節點（鼓勵分散）
*/
func evalCost(prob *Problem, assign []int, params SAParams) (float64, []Resource) {
	n := len(prob.Nodes)
	keys := prob.ResourceKey
	usage := zeroUsage(n, keys)

	// 聚合使用量
	for i, pod := range prob.Pods {
		ni := assign[i]
		if ni < 0 || ni >= n {
			continue // -1 表示暫不放置（本範例沒用到）
		}
		addRes(usage[ni], pod.Demand)
	}

	// 1) 超量懲罰
	over := 0.0
	for i := 0; i < n; i++ {
		for _, k := range keys {
			cap := prob.Nodes[i].Capacity[k]
			if cap <= 0 {
				// 沒容量的資源，若用到就當作超量
				if usage[i][k] > 0 {
					over += 1e6 // 大罰
				}
				continue
			}
			util := usage[i][k] / cap
			if util > 1.0 {
				over += (util - 1.0) * (util - 1.0) // 超越越多罰越重（平方）
			}
		}
	}

	// 2) 平衡（各資源在節點間的利用率方差）
	variance := 0.0
	for _, k := range keys {
		// 收集每個節點該資源利用率
		utils := make([]float64, 0, n)
		for i := 0; i < n; i++ {
			cap := prob.Nodes[i].Capacity[k]
			if cap <= 0 {
				continue
			}
			u := usage[i][k] / cap
			if u < 0 { u = 0 }
			utils = append(utils, u)
		}
		if len(utils) == 0 { continue }
		// 算方差
		mu := 0.0
		for _, v := range utils { mu += v }
		mu /= float64(len(utils))
		var v2 float64
		for _, v := range utils {
			d := v - mu
			v2 += d * d
		}
		v2 /= float64(len(utils))
		variance += v2
	}
	// 讓不同資源的方差平均一下
	if len(keys) > 0 {
		variance /= float64(len(keys))
	}

	// 3) 啟用節點數（有任何資源使用量>0 視為啟用）
	active := 0.0
	for i := 0; i < n; i++ {
		sum := 0.0
		for _, k := range keys {
			sum += usage[i][k]
		}
		if sum > 0 {
			active += 1.0
		}
	}

	cost := params.WOveruse*over + params.WVariance*variance + params.WActiveNodes*active
	return cost, usage
}

/************ 初始解（簡易貪婪） ************/
func initialGreedy(prob *Problem, params SAParams) []int {
	nPods := len(prob.Pods)
	nNodes := len(prob.Nodes)
	assign := make([]int, nPods)
	rnd := rand.New(rand.NewSource(params.Seed))

	for i := 0; i < nPods; i++ {
		bestNode := 0
		bestCost := math.Inf(1)
		tryOrder := rnd.Perm(nNodes)
		// 嘗試把 pod i 放入每個節點，挑成本最低的
		for _, ni := range tryOrder {
			assign[i] = ni
			c, _ := evalCost(prob, assign, params)
			if c < bestCost {
				bestCost = c
				bestNode = ni
			}
		}
		assign[i] = bestNode
	}
	return assign
}

/************ 鄰域操作：移動或交換 ************/
func neighbor(rnd *rand.Rand, assign []int, nNodes int) []int {
	nPods := len(assign)
	out := cloneAssignment(assign)
	if nPods == 0 { return out }
	if rnd.Float64() < 0.5 || nPods == 1 {
		// move：選一個 pod 改到另一個節點
		i := rnd.Intn(nPods)
		old := out[i]
		var to int
		for {
			to = rnd.Intn(nNodes)
			if to != old { break }
		}
		out[i] = to
	} else {
		// swap：兩個 pod 互換節點
		i := rnd.Intn(nPods)
		j := rnd.Intn(nPods)
		if i == j { return out }
		out[i], out[j] = out[j], out[i]
	}
	return out
}

/************ SA 主流程 ************/
func SolveSA(prob *Problem, params SAParams) (bestAssign []int, bestCost float64) {
	if params.Iterations <= 0 { params.Iterations = 2000 }
	if params.InitialTemp <= 0 { params.InitialTemp = 1.0 }
	if params.CoolingRate <= 0 || params.CoolingRate >= 1 { params.CoolingRate = 0.995 }
	if params.Seed == 0 { params.Seed = time.Now().UnixNano() }

	rnd := rand.New(rand.NewSource(params.Seed))

	curr := initialGreedy(prob, params)
	currCost, _ := evalCost(prob, curr, params)
	best := cloneAssignment(curr)
	bestCost = currCost

	T := params.InitialTemp
	for it := 0; it < params.Iterations; it++ {
		next := neighbor(rnd, curr, len(prob.Nodes))
		nextCost, _ := evalCost(prob, next, params)
		delta := nextCost - currCost
		if delta < 0 || rnd.Float64() < math.Exp(-delta/T) {
			curr = next
			currCost = nextCost
			if currCost < bestCost {
				best = cloneAssignment(curr)
				bestCost = currCost
			}
		}
		T *= params.CoolingRate
		if T < 1e-6 { break }
	}
	return best, bestCost
}

/************ Demo：直接跑一個例子 ************/
func main() {
	prob := &Problem{
		Pods: []Pod{
			{Demand: Resource{"cpu": 500, "memory": 1024}}, // mCPU / MiB 只是示意
			{Demand: Resource{"cpu": 250, "memory":  512}},
			{Demand: Resource{"cpu": 800, "memory": 2048}},
			{Demand: Resource{"cpu": 200, "memory":  256}},
			{Demand: Resource{"cpu": 400, "memory": 1024}},
		},
		Nodes: []Node{
			{Capacity: Resource{"cpu": 2000, "memory": 4096}},
			{Capacity: Resource{"cpu": 1500, "memory": 3072}},
			{Capacity: Resource{"cpu": 1000, "memory": 2048}},
		},
		ResourceKey: []string{"cpu", "memory"},
	}

	// 目標：嚴懲超量 + 平衡（方差） + 偏「合併使用較少節點」
	params := SAParams{
		Iterations:   4000,
		InitialTemp:  1.2,
		CoolingRate:  0.996,
		Seed:         42,
		WOveruse:     1000.0, // 超量要很痛
		WVariance:    1.0,    // 越平均越好
		WActiveNodes: 0.05,   // 稍微鼓勵合併（數值大會更偏向把工作擠到較少節點）
	}

	assign, cost := SolveSA(prob, params)

	fmt.Println("Best cost:", cost)
	for i, ni := range assign {
		fmt.Printf("Pod %d -> Node %d\n", i, ni)
	}
	// 額外印每個節點的利用率
	_, usage := evalCost(prob, assign, params)
	for i := range prob.Nodes {
		cpuU := usage[i]["cpu"] / max(prob.Nodes[i].Capacity["cpu"], 1)
		memU := usage[i]["memory"] / max(prob.Nodes[i].Capacity["memory"], 1)
		fmt.Printf("Node %d util: CPU=%.2f, MEM=%.2f\n", i, cpuU, memU)
	}
}
