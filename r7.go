package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"slices"
)

// Only change is buffering the results channel. It's interesting to see the
// difference that it makes.
func r7(measurementsPath string) {
	file, err := os.Open(measurementsPath)
	if err != nil {
		log.Fatalf("Could not open %s: %s", measurementsPath, err)
	}
	defer file.Close()

	fileStats, err := os.Stat(measurementsPath)
	if err != nil {
		log.Fatal(err)
	}
	fileSize := fileStats.Size()

	targetChunkSize := fileSize / int64(numWorkers)
	chunkStarts := make([]int64, numWorkers)
	const maxLineLength = 100
	buf := make([]byte, maxLineLength)
	for i := 1; i < numWorkers; i++ {
		currOffset, err := file.Seek(targetChunkSize-maxLineLength, 1)
		if err != nil {
			log.Fatalf("Error seeking to next chunk in measurements file: %s", err)
		}
		bytesRead, err := file.Read(buf)
		if err != nil {
			log.Fatalf("Error reading measurements file: %s", err)
		}
		newLineIndex := bytes.IndexByte(buf[:bytesRead], '\n')
		if newLineIndex == -1 {
			log.Fatalf(
				"Could not find a newline to split measurement file into chunks. Is there a line longer than %d bytes?",
				maxLineLength,
			)
		}
		afterNewLine := currOffset + int64(newLineIndex) + 1
		chunkStarts[i] = afterNewLine
		_, err = file.Seek(afterNewLine, 0)
		if err != nil {
			log.Fatal("Failed to seek to the beginning of the next chunk.")
		}
	}

	results := make(chan map[string]*Stats, numWorkers)
	for i, offset := range chunkStarts {
		var size int64
		if i < len(chunkStarts)-1 {
			// read to the start of the next chunk
			size = chunkStarts[i+1] - offset
		} else {
			// read to the end of the file
			size = fileSize - offset + 1
		}
		go processChunkR6(measurementsPath, offset, size, results)
	}

	stationStats := make(map[string]*Stats)
	for i := 0; i < numWorkers; i++ {
		result := <-results
		for station, stats := range result {
			aggregateStats, ok := stationStats[station]
			if !ok {
				stationStats[station] = stats
			} else {
				// inline the Combine method to use the pointer
				if stats.Max > aggregateStats.Max {
					aggregateStats.Max = stats.Max
				}
				if stats.Min < aggregateStats.Min {
					aggregateStats.Min = stats.Min
				}
				aggregateStats.Sum += stats.Sum
				aggregateStats.Count += stats.Count
			}
		}
	}
	close(results)

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
