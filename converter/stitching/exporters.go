package stitching

import "sync"

//ExporterMap maintains a synchronous map of exporter specific information
type ExporterMap struct {
	exporters    map[string]Exporter
	mutex        *sync.Mutex
	numStitchers int
}

//Exporter holds exporter specific information
type Exporter struct {
	lastPossFlowEnds MinMap
	//Counters per exporter could be stored here
}

//NewExporterMap creates an ExporterMap with the info needed to
//initialize a new Exporter
func NewExporterMap(numStitchers int) ExporterMap {
	return ExporterMap{
		exporters:    make(map[string]Exporter),
		mutex:        new(sync.Mutex),
		numStitchers: numStitchers,
	}
}

//NewExporter creates an Exporter which contains Exporter specific information
//such as time tracking details
func NewExporter(numStitchers int) Exporter {
	return Exporter{
		lastPossFlowEnds: NewSliceMinMap(numStitchers),
	}
}

//Get retrieves the Exporter entry for a given address.
//If an entry doesn't exist, one will be created.
func (e ExporterMap) Get(exporterAddress string) Exporter {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	exporter, ok := e.exporters[exporterAddress]
	if !ok {
		exporter = NewExporter(e.numStitchers)
		e.exporters[exporterAddress] = exporter
	}
	return exporter
}
