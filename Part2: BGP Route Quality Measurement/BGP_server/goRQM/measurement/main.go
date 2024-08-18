package measurement

import (
	"encoding/csv"
	"fmt"
	"goRQM"
	"os"

	"time"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

func LaunchMeasurement() {
	TTestMeasurements()
// 	RankingMeasurements()

}

func TTestMeasurements() {
	records := [][]interface{}{{"Sample_size", "Iteration", "Time_taken_same_Mean", "Time_taken_different_mean"}}
	i := 1
	for i <= 50 {
		j := 0
		for j < 10 {
			measurement1, measurement2 := MeasurementTTest(i)
			records = append(records, []interface{}{i, j, measurement1, measurement2})
			j += 1
		}
		i += 1
	}
	file, _ := os.Create("measurement_results/measurement_ttest.csv")
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		strRecord := make([]string, len(record))
		for i, v := range record {
			strRecord[i] = fmt.Sprint(v)
		}
		if err := writer.Write(strRecord); err != nil {
			return
		}
	}
}

func MeasurementTTest(sample_size int) (int64, int64) {
	samples_20 := make([][]float64, 0)
	i := 0
	for i <= 50001 {
		samples_20 = append(samples_20, generateSamples(20, 1, sample_size))
		i++
	}

	samples_40 := make([][]float64, 0)
	i = 0
	for i <= 50001 {
		samples_40 = append(samples_40, generateSamples(40, 1, sample_size))
		i++
	}

	// warm up the access
	i = 0
	for i < 50001 {
		goRQM.TTest(samples_20[i], samples_20[i+1], 0.005)
		i++
	}
	
	i = 0
	start := time.Now()
	for i < 50001 {
		goRQM.TTest(samples_20[i], samples_20[i+1], 0.005)
		i++
	}
	duration1 := time.Since(start)
	
	// warm up the access
	i = 0
	for i < 50001 {
		goRQM.TTest(samples_20[i], samples_40[i], 0.005)
		i++
	}
	
	start = time.Now()
	i = 0
	for i < 50001 {
		goRQM.TTest(samples_20[i], samples_40[i], 0.005)
		i++
	}
	duration2 := time.Since(start)
	return duration1.Microseconds(), duration2.Microseconds()
}

func RankingMeasurements() {
	records := [][]interface{}{{"Nbr_neighbor", "Iteration", "Time_taken"}}
	nbr_neighbor := []int{2, 4, 8, 16, 32, 64, 128}
	for _, i := range nbr_neighbor {
		j := 0
		for j < 10 {
			records = append(records, []interface{}{i, j, MeasurementRanking(i)})
			j++
		}
	}
	file, _ := os.Create("measurement_results/measurement_ranking.csv")
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		strRecord := make([]string, len(record))
		for i, v := range record {
			strRecord[i] = fmt.Sprint(v)
		}

		if err := writer.Write(strRecord); err != nil {
			return
		}
	}
}

func MeasurementRanking(nbr_neighbor int) int64 {
	prefix := "192.168.0.0/24"
	neighbors := make([]map[string]*goRQM.NeighborInformation, 0)
	samples_mean := []float64{20, 40}
	it := 0
	for it < 1000 {
		neigh := make(map[string]*goRQM.NeighborInformation, 0)
		i := 0
		for i < nbr_neighbor {
			neigh[fmt.Sprintf("10.0.0.%d", i)] = &goRQM.NeighborInformation{
				Last_measurements: generateSamples(samples_mean[rand.Intn(len(samples_mean))], 1, 5),
			}
			i++
		}
		neighbors = append(neighbors, neigh)
		it ++
	}

	start := time.Now()
	i := 0
	for i < 1000{
		goRQM.Ranking(prefix, neighbors[i])
		i ++
	}
	duration := time.Since(start)
	return duration.Microseconds()
}

func generateSamples(mean float64, stdDev float64, size int) []float64 {
	values := make([]float64, size)
	normalDist := distuv.Normal{
		Mu:    mean,
		Sigma: stdDev,
		Src:   rand.NewSource(uint64(time.Now().UnixMicro())),
	}
	for i := 0; i < size; i++ {
		values[i] = normalDist.Rand()
	}
	return values
}
