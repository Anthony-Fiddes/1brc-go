package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
)

func processChunkR5(
	measurementsPath string,
	offset int64,
	length int64,
	results chan<- map[string]Stats,
) {
	file, err := os.Open(measurementsPath)
	if err != nil {
		log.Fatalf("Could not open %s: %s", measurementsPath, err)
	}
	defer file.Close()

	_, err = file.Seek(offset, 0)
	if err != nil {
		log.Fatalf("processChunk failed to seek to beginning of chunk: %s", err)
	}
	chunkReader := io.LimitReader(file, length)
	bufReader := bufio.NewReaderSize(chunkReader, 1024*1024)

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
	results <- stationStats
}

// the only difference from r4 is that we give each goroutine a 1MB buffer.
func r5(measurementsPath string) {
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

	results := make(chan map[string]Stats)
	for i, offset := range chunkStarts {
		var size int64
		if i < len(chunkStarts)-1 {
			// read to the start of the next chunk
			size = chunkStarts[i+1] - offset
		} else {
			// read to the end of the file
			size = fileSize - offset + 1
		}
		go processChunkR5(measurementsPath, offset, size, results)
	}

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
