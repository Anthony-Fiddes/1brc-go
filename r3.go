package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
)

// What if we just increase the buffer size? Looks like that saves another
// ~15%. Seems like there's pretty diminishing returns on increasing the
// buffer size over 1MB.
func r3(measurementsPath string) {
	file, err := os.Open(measurementsPath)
	if err != nil {
		log.Fatalf("Could not open %s: %s", measurementsPath, err)
	}
	defer file.Close()
	bufReader := bufio.NewReaderSize(file, 1024*1024)

	stationStats := make(map[string]Stats)
	csvReader := csv.NewReader(bufReader)
	csvReader.Comma = ';'
	csvReader.Comment = '#'
	csvReader.FieldsPerRecord = 2
	csvReader.ReuseRecord = true
	for {
		row, err := csvReader.Read()

		if row == nil && err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Could not read row from %s: %s", measurementsPath, err)
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
