package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mkock/dmr-parser/engines"
)

var (
	errWriteFile = errors.New("Unable to write to /tmp directory")
)

var engine = flag.String("parser", "string", "Parser to use: 'string' or 'xml'")
var inFile = flag.String("infile", "input.xml", "DMR XML file in UTF-8 format")
var outFile = flag.String("outfile", "out.csv", "Name of file to stream CSV data to")

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()
	f := r.File[0] // Assuming that zip file contains a single file.
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			panic(err)
		}
	}()
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, f.Mode())
	if err != nil {
		panic(errWriteFile)
	}
	_, writeErr := io.Copy(file, rc)
	if writeErr != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	xmlFileName := filepath.Join("/tmp/", "out.xml")

	if _, err := os.Stat(*inFile); os.IsNotExist(err) {
		fmt.Printf("abort: file %q does not seem to exist\n", *inFile)
		return
	}

	// Detect if ZIP-file and unpack it if so.
	isZip := filepath.Ext(*inFile) == ".zip"
	if isZip {
		err := unzip(*inFile, xmlFileName)
		if err != nil {
			panic(err)
		}
	} else {
		xmlFileName = *inFile
	}

	xmlFile, err := os.Open(xmlFileName)
	if err != nil {
		fmt.Println("Unable to open file:", err)
		return
	}
	defer func() {
		if err := xmlFile.Close(); err != nil {
			panic(err)
		}
	}()

	// Pick a parser based on CLI flag.
	var parser engines.IDMRParser
	switch *engine {
	case "xml":
		parser = engines.NewXMLParser()
	case "string":
		parser = engines.NewStringParser()
	default:
		fmt.Printf("Invalid parser: %q\n", engine)
		return
	}

	// Nr. of workers = cpu core count - 1 for the main go routine.
	numWorkers := int(math.Max(1.0, float64(runtime.NumCPU()-1)))

	// Prepare channels for communicating parsed data and termination.
	lines, parsed, done := make(chan []string, numWorkers), make(chan string, numWorkers), make(chan int)

	// Start the number of workers (parsers) determined by numWorkers.
	fmt.Printf("Starting %v workers...\n", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go parser.ParseExcerpt(i, lines, parsed, done)
	}

	// Main file scanner go routine.
	go func() {
		scanner := bufio.NewScanner(xmlFile)
		excerpt := []string{}
		grab := false
		defer func() {
			close(lines)
		}()
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "<ns:Statistik>") {
				grab = true
			} else if strings.HasPrefix(line, "</ns:Statistik>") {
				grab = false
				excerpt = append(excerpt, line)
				lines <- excerpt // On every closing elem. we send the excerpt to a worker and move on.
				excerpt = nil
			}
			if grab {
				excerpt = append(excerpt, line)
			}
		}
	}()

	var vehicles engines.VehicleList = make(map[string]struct{}) // For keeping track of unique vehicles.

	// Wait for parsed excerpts to come in, and ensure their uniqueness by using a map.
	waits := numWorkers
	for {
		select {
		case vehicle := <-parsed:
			if _, ok := vehicles[vehicle]; !ok {
				vehicles[vehicle] = struct{}{}
			}
		case <-done:
			waits--
			if waits == 0 {
				writeToFile(vehicles, *outFile)
				return
			}
		}
	}

}

func writeToFile(vehicles engines.VehicleList, outFile string) {
	out, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Unable to open output file %v for writing.\n", outFile)
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(err)
		}
	}()

	// Write CSV data.
	for vehicle := range vehicles {
		_, err := out.WriteString(vehicle + "\n")
		if err != nil {
			fmt.Println("Unable to write to output file, unknown write error")
		}
	}

	fmt.Printf("Done - CSV data written to %q\n", outFile)
}
