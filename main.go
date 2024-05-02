package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
)

const testMeasurementsPath = "test_measurements.csv"
const realMeasurementsPath = "measurements.txt"

type Stats struct {
	Max   float64
	Min   float64
	Sum   float64
	Count int
}

func (s *Stats) Add(reading float64) {
	if reading > s.Max {
		s.Max = reading
	}
	if reading < s.Min {
		s.Min = reading
	}
	s.Sum += reading
	s.Count++
}

func (s *Stats) Mean() float64 {
	return s.Sum / float64(s.Count)
}

func (s Stats) String() string {
	return fmt.Sprintf("%.1f/%.1f/%.1f", s.Min, s.Mean(), s.Max)
}

func main() {
	measurementsPath := testMeasurementsPath
	file, err := os.Open(measurementsPath)
	if err != nil {
		log.Fatalf("Could not open %s: %s", measurementsPath, err)
	}
	defer file.Close()

	stationStats := make(map[string]Stats)
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
