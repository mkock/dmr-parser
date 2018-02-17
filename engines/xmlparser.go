package engines

import (
	"encoding/xml"
	"fmt"
	"strings"
	"sync"
)

// <ns:Statistik>
type vehicleStat struct {
	Ident uint64      `xml:"KoeretoejIdent"`
	Type  uint64      `xml:"KoeretoejArtNummer"`
	Info  vehicleInfo `xml:"KoeretoejOplysningGrundStruktur"`
}

// <ns:KoeretoejOplysningGrundStruktur>
type vehicleInfo struct {
	Designation vehicleDesignation `xml:"KoeretoejBetegnelseStruktur"`
}

// <ns:Model>
type vehicleModel struct {
	Type uint64 `xml:"KoeretoejModelTypeNummer"`
	Name string `xml:"KoeretoejModelTypeNavn"`
}

// <ns:Variant>
type vehicleVariant struct {
	Type uint64 `xml:"KoeretoejVariantTypeNummer"`
	Name string `xml:"KoeretoejVariantTypeNavn"`
}

// <ns:Type>
type vehicleType struct {
	Type uint64 `xml:"KoeretoejTypeTypeNummer"`
	Name string `xml:"KoeretoejTypeTypeNavn"`
}

// <ns:KoeretoejBetegnelseStruktur>
type vehicleDesignation struct {
	BrandTypeNr   uint64         `xml:"KoeretoejMaerkeTypeNummer"`
	BrandTypeName string         `xml:"KoeretoejMaerkeTypeNavn"`
	Model         vehicleModel   `xml:"Model"`
	Variant       vehicleVariant `xml:"Variant"`
	Type          vehicleType    `xml:"Type"`
}

// XMLParser represents an XML parser.
type XMLParser struct {
	decoder  *xml.Decoder
	decMutex *sync.Mutex
	mapMutex *sync.Mutex
}

// NewXMLParser creates a new XML parser.
func NewXMLParser() *XMLParser {
	return &XMLParser{nil, &sync.Mutex{}, &sync.Mutex{}}
}

// ParseExcerpt parses XML file using XML decoding.
func (p *XMLParser) ParseExcerpt(id int, lines <-chan []string, parsed chan<- string, done chan<- int) {
	proc := 0 // How many excerpts did we process?
	var stat vehicleStat
	for excerpt := range lines {
		if err := xml.Unmarshal([]byte(strings.Join(excerpt, "\n")), &stat); err != nil {
			panic(err) // We _could_ skip it, but it's better to halt execution here.
		}
		if stat.Type == 1 {
			csv := fmt.Sprintf("%v;%v", stat.Info.Designation.BrandTypeName, stat.Info.Designation.Model.Name)
			parsed <- csv
			proc++
		}
	}
	fmt.Printf("XML-worker %d finished after processing %d excerpts\n", id, proc)
	done <- id
}
