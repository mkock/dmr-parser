package engines

// VehicleList contains vehicles that were found during parsing.
type VehicleList map[string]struct{}

// IDMRParser is an interface that parsers must implement in order to be callable by the application.
type IDMRParser interface {
	ParseExcerpt(id int, lines <-chan []string, parsed chan<- string, done chan<- int)
}
