package engines

import (
	"fmt"
	"strings"
)

// StringParser parses XML as strings.
type StringParser struct{}

// NewStringParser returns a DMR string parser.
func NewStringParser() *StringParser {
	return &StringParser{}
}

func getXMLVal(line string) string {
	start := strings.Index(line, ">") + 1
	end := strings.LastIndex(line, "<")
	return line[start:end]
}

// ParseExcerpt runs the string parser.
func (p *StringParser) ParseExcerpt(id int, lines <-chan []string, parsed chan<- string, done chan<- int) {
	var isCar bool
	csv, brand, model := "", "", ""
	proc := 0 // How many excerpts did we process?
	for excerpt := range lines {
		for _, line := range excerpt {
			if strings.HasPrefix(line, "<ns:KoeretoejArtNummer>") {
				isCar = strings.HasPrefix(line, "<ns:KoeretoejArtNummer>1<")
				continue
			}
			if isCar {
				if strings.HasPrefix(line, "<ns:KoeretoejMaerkeTypeNavn>") {
					brand = getXMLVal(line)
				} else if strings.HasPrefix(line, "<ns:KoeretoejModelTypeNavn>") {
					model = getXMLVal(line)
				}
				if brand != "" && model != "" {
					csv = fmt.Sprintf("%v;%v", brand, model)
					parsed <- csv
					proc++
					brand, model = "", ""
				}
			}
		}
	}
	fmt.Printf("String-worker %d finished after processing %d excerpts\n", id, proc)
	done <- id
}
