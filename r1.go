package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"slices"
	"strconv"
)

func (s *Stats) Combine(other Stats) {
	if other.Max > s.Max {
		s.Max = other.Max
	}
	if other.Min < s.Min {
		s.Min = other.Min
	}
	s.Sum += other.Sum
	s.Count += other.Count
}

func processRows(rows <-chan []string, results chan<- map[string]Stats) {
	stationStats := make(map[string]Stats)
	for row := range rows {
		station := row[0]
		reading, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			log.Fatalf("Could not convert reading '%s' to float64", row[1])
		}

		stats, ok := stationStats[station]
		if !ok {
			stationStats[station] = Stats{
				Max:   reading,
				Min:   reading,
				Sum:   reading,
				Count: 1,
			}
		} else {
			stats.Add(reading)
			stationStats[station] = stats
		}
	}
	results <- stationStats
}

var numWorkers = runtime.GOMAXPROCS(0)

func r1(measurementsPath string) {
	file, err := os.Open(measurementsPath)
	if err != nil {
		log.Fatalf("Could not open %s: %s", measurementsPath, err)
	}
	defer file.Close()

	rows := make(chan []string)
	results := make(chan map[string]Stats)
	for i := 0; i < numWorkers; i++ {
		go processRows(rows, results)
	}

	go func() {
		csvReader := csv.NewReader(file)
		csvReader.Comma = ';'
		csvReader.Comment = '#'
		for {
			row, err := csvReader.Read()

			if row == nil && err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Could not read row from %s: %s", measurementsPath, err)
			}
			if len(row) != 2 {
				log.Fatalf(
					"Data in %s is malformed. There should be exactly 2 columns, read row %v",
					measurementsPath,
					row,
				)
			}
			rows <- row
		}
		close(rows)
	}()

	stationStats := make(map[string]Stats)
	for i := 0; i < numWorkers; i++ {
		result := <-results
		for station, stats := range result {
			currStats, ok := stationStats[station]
			if !ok {
				stationStats[station] = stats
			} else {
				stats.Combine(currStats)
				stationStats[station] = stats
			}
		}
	}

	stations := make([]string, 0, len(stationStats))
	for station := range stationStats {
		stations = append(stations, station)
	}
	slices.Sort(stations)

	fmt.Print("{")
	for i, station := range stations {
		stats := stationStats[station]
		fmt.Printf("%s=%s", station, stats)
		if i != len(stations)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Println("}")
}
