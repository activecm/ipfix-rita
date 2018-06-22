package stitching

import "sync"

type exporterMap struct {
	exporters map[string]exporter
	mutex     *sync.RWMutex
}

func newExporterMap() exporterMap {
	return exporterMap{
		exporters: make(map[string]exporter),
		mutex:     new(sync.RWMutex),
	}
}

func (e exporterMap) get(address string) (exporter, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	exp, ok := e.exporters[address]
	return exp, ok
}

func (e exporterMap) add(newExporter exporter) {
	e.mutex.Lock()
	e.exporters[newExporter.address] = newExporter
	e.mutex.Unlock()
}
