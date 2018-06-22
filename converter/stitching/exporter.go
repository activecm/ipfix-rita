package stitching

type exporter struct {
	address string
	flusher flusher
}

func newExporter(address string) exporter {
	return exporter{
		address: address,
		flusher: newFlusher(address),
	}
}
