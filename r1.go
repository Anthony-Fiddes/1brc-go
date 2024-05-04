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

// Created without looking at a profiler at all. It ends up being about twice as
// slow. After looking at the profiler, it's clear that this approach wouldn't
// save a lot of time, since the work done by processRows took up way less time
// than the Read calls initialized by the CSV reader.
//
// Examples online open multiple file handles at the same time and seek to
// different parts of the file. My first instinct was that having a single large
// read would be the fastest way to access the data. I found a lovely source on
// how SSDs work and it appears to agree:
// https://codecapsule.com/2014/02/12/coding-for-ssds-part-5-access-patterns-and-system-optimizations/
//
// But I can't argue with the results that people are getting. Maybe the
// difference is that we're likely working on NVME SSDs where the article above
// may be focusing on SATA SSDs?
//
// Indeed as I search around more, this more recent paper suggests that they
// needed >100 concurrent read requests to saturate one NVME SSD:
// https://www.vldb.org/pvldb/vol16/p2090-haas.pdf.
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
