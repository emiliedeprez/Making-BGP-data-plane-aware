package goRQM

import (
	"bufio"
	"container/heap"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"sync"
	"time"

	"net"
	"strconv"

	"github.com/spf13/viper"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

var config ServerConfiguration
var connectionMutex sync.Mutex

type NeighborConfiguration struct {
	MeasurementAddress string `yaml:"measurementaddress"`
	VRF                string `yaml:"vrf"`
}

type ServerConfiguration struct {
	Server struct {
		Port                 int     `yaml:"port"`
		Address              string  `yaml:"address"`
		WorkingTreads       int     `yaml:"workingtreads"`
		Alpha                float64 `yaml:"alpha"`
		MaxNumberMeasurement int     `yaml:"maxnumbermeasurement"`
	} `yaml:"server"`
	Neighbors map[string]NeighborConfiguration `yaml:"neighbors"`
}

type NeighborInformation struct {
	Time_limit        time.Time
	Sampling_interval time.Duration
	Last_measurements []float64
	MeasurementIP     []string
}

type TTestResult struct {
	t                   float64
	pValue              bool
	one_lower_than_two  bool
	one_higher_than_two bool
	one_equal_two       bool
}

type PrefixMeasurement struct {
	mutex      sync.Mutex
	prefix     string
	nextRun    time.Time
	interval   time.Duration
	neighbors  map[string]*NeighborInformation
	connection net.Conn
}

type MeasurementPriorityQueue []*PrefixMeasurement

// For the heap interface
// https://pkg.go.dev/container/heap
func (pq MeasurementPriorityQueue) Len() int           { return len(pq) }
func (pq MeasurementPriorityQueue) Less(i, j int) bool { return pq[i].nextRun.Before(pq[j].nextRun) }
func (pq MeasurementPriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *MeasurementPriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*PrefixMeasurement))
}
func (pq *MeasurementPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	task := old[n-1]
	*pq = old[0 : n-1]
	return task
}

var mapMeasurement map[string]*PrefixMeasurement
var mapMeasurementMutex sync.Mutex

var addMeasurement func(string)
var removeMeasurement func(string)
var restartMeasurement func(string)
var resetMeasurement func(string)

func LaunchServer(configPath string) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	var err error
	if err = v.ReadInConfig(); err != nil {
		fmt.Println(err)
		return
	}
	if err = v.UnmarshalExact(&config); err != nil {
		fmt.Println(err)
		return
	}

	mapMeasurement = make(map[string]*PrefixMeasurement)

	listener, err := net.Listen("tcp", config.Server.Address+":"+strconv.Itoa(config.Server.Port))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()

	fmt.Println("Server is listening on", config.Server.Address+":"+strconv.Itoa(config.Server.Port))

	go MeasurementManager()

	time.Sleep(1 * time.Second)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			return
		}
		fmt.Println("Connected to", conn.RemoteAddr().String())
		messageChannel := make(chan map[string]interface{})
		go HandleConnection(conn, messageChannel)
		for i := 0; i < config.Server.WorkingTreads; i++ {
			go HandleMessage(conn, messageChannel)
		}
	}
}

// Scheduler
// https://reintech.io/blog/implementing-job-scheduler-in-go
func MeasurementManager() {
	var heapMutex sync.Mutex
	measurementPQ := &MeasurementPriorityQueue{}
	heap.Init(measurementPQ)

	addMeasurement = func(prefix string) {
		heapMutex.Lock()
		defer heapMutex.Unlock()
		mapMeasurement[prefix].mutex.Lock()
		defer mapMeasurement[prefix].mutex.Unlock()
		mapMeasurement[prefix].nextRun = time.Now()
		heap.Push(measurementPQ, mapMeasurement[prefix])
	}

	resetMeasurement = func(prefix string) {
		heapMutex.Lock()
		defer heapMutex.Unlock()
		mapMeasurement[prefix].mutex.Lock()
		defer mapMeasurement[prefix].mutex.Unlock()
		if _, exists := mapMeasurement[prefix]; exists {
			for i, task := range *measurementPQ {
				if task.prefix == prefix {
					heap.Remove(measurementPQ, i)
					break
				}
			}
		}
		mapMeasurement[prefix].nextRun = time.Now()
		heap.Push(measurementPQ, mapMeasurement[prefix])
	}

	restartMeasurement = func(prefix string) {
		heapMutex.Lock()
		defer heapMutex.Unlock()
		mapMeasurement[prefix].mutex.Lock()
		defer mapMeasurement[prefix].mutex.Unlock()
		mapMeasurement[prefix].nextRun = time.Now().Add(mapMeasurement[prefix].interval)
		heap.Push(measurementPQ, mapMeasurement[prefix])
	}

	removeMeasurement = func(prefix string) {
		heapMutex.Lock()
		defer heapMutex.Unlock()
		if _, exists := mapMeasurement[prefix]; exists {
			for i, task := range *measurementPQ {
				if task.prefix == prefix {
					heap.Remove(measurementPQ, i)
					delete(mapMeasurement, prefix)
					return
				}
			}
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		heapMutex.Lock()
		if measurementPQ.Len() == 0 {
			heapMutex.Unlock()
			continue
		}
		nextTask := (*measurementPQ)[0]
		if time.Now().After(nextTask.nextRun) {
			heap.Pop(measurementPQ)
			heapMutex.Unlock()
			go MakeMeasurement(nextTask.prefix)
		} else {
			heapMutex.Unlock()
		}
	}
}

func HandleConnection(conn net.Conn, messageChannel chan map[string]interface{}) error {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	decoder := json.NewDecoder(reader)

	for {
		var req map[string]interface{}
		connectionMutex.Lock()
		if err := decoder.Decode(&req); err != nil {
			return err
		}
		connectionMutex.Unlock()
		messageChannel <- req
	}
}

// Handle subscribe-unsubscribe messages
func HandleMessage(conn net.Conn, messageChannel chan map[string]interface{}) {
	for req := range messageChannel {
		switch req["type"].(string) {
		case "subscribe":
			mapMeasurementMutex.Lock()
			if prefixInfo, ok := mapMeasurement[req["prefix"].(string)]; ok {
				mapMeasurementMutex.Unlock()
				prefixInfo.mutex.Lock()
				last_measurements := make([]float64, 0)
				if _, ok := prefixInfo.neighbors[req["neighbor"].(string)]; ok {
					last_measurements = prefixInfo.neighbors[req["neighbor"].(string)].Last_measurements
				}
				ips := make([]string, 0)
				for _, ip := range req["measurementIPs"].([]interface{}) {
					ips = append(ips, ip.(string))
				}
				Neighbor := &NeighborInformation{
					Time_limit:        time.Unix(int64(req["timeLimit"].(float64)), 0),
					Sampling_interval: time.Duration(int64(req["samplingInterval"].(float64))) * time.Millisecond,
					Last_measurements: last_measurements,
					MeasurementIP:     ips,
				}
				prefixInfo.neighbors[req["neighbor"].(string)] = Neighbor

				if prefixInfo.interval > prefixInfo.neighbors[req["neighbor"].(string)].Sampling_interval {
					prefixInfo.interval = prefixInfo.neighbors[req["neighbor"].(string)].Sampling_interval
				}
				prefixInfo.mutex.Unlock()
				resetMeasurement(req["prefix"].(string))
			} else {
				ips := make([]string, 0)
				for _, ip := range req["measurementIPs"].([]interface{}) {
					ips = append(ips, ip.(string))
				}
				Neighbor := &NeighborInformation{
					Time_limit:        time.Unix(int64(req["timeLimit"].(float64)), 0),
					Sampling_interval: time.Duration(int64(req["samplingInterval"].(float64))) * time.Millisecond,
					Last_measurements: make([]float64, 0),
					MeasurementIP:     ips,
				}

				mapMeasurement[req["prefix"].(string)] = &PrefixMeasurement{
					prefix:     req["prefix"].(string),
					interval:   Neighbor.Sampling_interval,
					neighbors:  make(map[string]*NeighborInformation),
					connection: conn,
				}
				mapMeasurement[req["prefix"].(string)].neighbors[req["neighbor"].(string)] = Neighbor
				addMeasurement(req["prefix"].(string))
				mapMeasurementMutex.Unlock()
			}
		case "unsubscribe":
			if prefixInfo, ok := mapMeasurement[req["prefix"].(string)]; ok {
				if _, ok := prefixInfo.neighbors[req["neighbor"].(string)]; ok {
					prefixInfo.mutex.Lock()
					delete(prefixInfo.neighbors, req["neighbor"].(string))
					mapMeasurementMutex.Lock()
					if len(prefixInfo.neighbors) > 0 {
						mapMeasurementMutex.Unlock()
						prefixInfo.interval = time.Duration(math.MaxInt64)
						for _, neighbor := range mapMeasurement[req["prefix"].(string)].neighbors {
							if prefixInfo.interval > neighbor.Sampling_interval {
								prefixInfo.interval = neighbor.Sampling_interval
							}
						}
						prefixInfo.mutex.Unlock()
					} else {
						prefixInfo.mutex.Unlock()
						removeMeasurement(req["prefix"].(string))
						mapMeasurementMutex.Unlock()
					}
				}
			}
		}
	}
}

// Perform the measurement of a prefix
// Does the pings, ranks and sends response to BGP
func MakeMeasurement(prefix string) {
	if _, y := mapMeasurement[prefix]; !y {
		return
	}
	encoder := json.NewEncoder(mapMeasurement[prefix].connection)
	mapMeasurement[prefix].mutex.Lock()
	for name, informations := range mapMeasurement[prefix].neighbors {
		if informations.Time_limit != time.Unix(0, 0) && time.Now().After(informations.Time_limit) {
			resp := map[string]interface{}{
				"type":     "expired_timer",
				"neighbor": name,
				"prefix":   prefix,
			}
			encoder.Encode(&resp)
			delete(mapMeasurement[prefix].neighbors, name)
		} else {
			Ping(informations.MeasurementIP, name, prefix)
		}
	}
	if len(mapMeasurement[prefix].neighbors) == 0 {
		removeMeasurement(prefix)
		return
	}
	resp := Ranking(prefix, mapMeasurement[prefix].neighbors)
	encoder.Encode(&resp)
	mapMeasurement[prefix].mutex.Unlock()
	restartMeasurement(prefix)
}

// Rank the neighbors of one prefix
func Ranking(prefix string, measurement_neighbors map[string]*NeighborInformation) (map[string]interface{}){
	array_pair := make(map[string]map[string]TTestResult, 0)
	neighbors := make([]string, 0)
	for key1, value1 := range measurement_neighbors {
		for key2, value2 := range measurement_neighbors {
			if _, ok := array_pair[key1]; !ok {
				array_pair[key1] = make(map[string]TTestResult, 0)
			}
			if _, ok := array_pair[key2]; !ok {
				array_pair[key2] = make(map[string]TTestResult, 0)
			}
			t, p_value, equal, lower, greater := TTest(value1.Last_measurements, value2.Last_measurements, config.Server.Alpha)
			array_pair[key1][key2] = TTestResult{t, p_value, lower, greater, equal}
			array_pair[key2][key1] = TTestResult{t, p_value, greater, lower, equal}

		}
		neighbors = append(neighbors, key1)
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return array_pair[neighbors[i]][neighbors[j]].pValue && array_pair[neighbors[i]][neighbors[j]].one_lower_than_two && array_pair[neighbors[j]][neighbors[i]].one_higher_than_two
	})

	
	resp := map[string]interface{}{
		"type":      "quality",
		"prefix":    prefix,
		"neighbors": make(map[string]uint64, 0),
	}

	var rank uint64
	rank = 1
	for i, neighbor := range neighbors {
		if i == 0 {
			resp["neighbors"].(map[string]uint64)[neighbor] = rank
		} else {
			if array_pair[neighbors[i]][neighbors[i-1]].pValue && array_pair[neighbors[i-1]][neighbors[i]].one_lower_than_two && array_pair[neighbors[i]][neighbors[i-1]].one_higher_than_two {
				rank += 1
			}
			resp["neighbors"].(map[string]uint64)[neighbor] = rank
		}
	}
	return resp
}

// Perform a t-test between two samples
func TTest(sample1 []float64, sample2 []float64, alpha float64) (float64, bool, bool, bool, bool) {
	if len(sample1) < 2 || len(sample2) < 2 {
		return math.Inf(1), false, len(sample1) == len(sample2), len(sample1) > len(sample2), len(sample1) < len(sample2)

	}
	mean_x := stat.Mean(sample1, nil)
	mean_y := stat.Mean(sample2, nil)
	std_x := stat.StdDev(sample1, nil)
	std_y := stat.StdDev(sample2, nil)

	n_x := float64(len(sample1))
	n_y := float64(len(sample2))

	sPool := (((n_x - 1) * math.Pow(std_x, 2)) + ((n_y - 1) * math.Pow(std_y, 2))) / (n_x + n_y - 2)
	t := (mean_x - mean_y) / (math.Sqrt(sPool) * math.Sqrt(1/n_x+1/n_y))

	dist := distuv.StudentsT{
		Mu:    0,
		Sigma: 1,
		Nu:    n_x + n_y - 2,
	}

	pValue := 2 * (1 - dist.CDF(math.Abs(t)))

	return t, pValue < alpha, math.Abs(t) < dist.CDF(math.Abs(alpha)), math.Abs(t) > dist.CDF(math.Abs(alpha)) && t < 0, math.Abs(t) > dist.CDF(math.Abs(alpha)) && t > 0
}

// Do the ping
// Put the route in the ip table of the corresponding VRF, pings and remove the route in the ip table
func Ping(IPs []string, neighbor string, prefix string) {
	cmd := exec.Command("ip", "route", "add", "vrf", config.Neighbors[neighbor].VRF, mapMeasurement[prefix].neighbors[neighbor].MeasurementIP[0], "via", config.Neighbors[neighbor].MeasurementAddress)
	err := cmd.Run()
	if err != nil {
		fmt.Println("error ping add ", err)
	}

	cmd = exec.Command("ping", "-I", config.Neighbors[neighbor].VRF, "-c", "4", mapMeasurement[prefix].neighbors[neighbor].MeasurementIP[0], "-w", "4")
	output, _ := cmd.Output()
	cmd = exec.Command("ip", "route", "del", "vrf", config.Neighbors[neighbor].VRF, mapMeasurement[prefix].neighbors[neighbor].MeasurementIP[0])
	if err := cmd.Run(); err != nil {
		fmt.Println("Error removing route", err)
	}

	re := regexp.MustCompile(`time=([\d.]+) ms`)
	first := true
	for _, match := range re.FindAllStringSubmatch(string(output), -1) {
		if len(match) > 1 {
			if first {
				first = false
			} else {
				timeFloat, _ := strconv.ParseFloat(match[1], 64)
				mapMeasurement[prefix].neighbors[neighbor].Last_measurements = append(mapMeasurement[prefix].neighbors[neighbor].Last_measurements, timeFloat)
			}
		}
	}
	if len(mapMeasurement[prefix].neighbors[neighbor].Last_measurements) > config.Server.MaxNumberMeasurement {
		mapMeasurement[prefix].neighbors[neighbor].Last_measurements = mapMeasurement[prefix].neighbors[neighbor].Last_measurements[len(mapMeasurement[prefix].neighbors[neighbor].Last_measurements)-config.Server.MaxNumberMeasurement:]
	}
}
