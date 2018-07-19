package ipfix

import (
	"context"
	"reflect"
)

func DrainNReaders(readers []Reader, ctx context.Context) (<-chan Flow, <-chan error) {

	readerData := make([]<-chan Flow, 0, len(readers))
	readerDataCases := make([]reflect.SelectCase, 0, len(readers))
	readerErrors := make([]<-chan error, 0, len(readers))
	readerErrorCases := make([]reflect.SelectCase, 0, len(readers))

	for i := range readers {
		data, errors := readers[i].Drain(ctx)
		readerData = append(readerData, data)
		readerDataCases = append(readerDataCases,
			reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(data)},
		)
		readerErrors = append(readerErrors, errors)
		readerErrorCases = append(readerErrorCases,
			reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(errors)},
		)
	}

	outData := make(chan Flow)
	outErrors := make(chan error)

	go func() {
		for len(readerData) != 0 {
			chosen, flow, ok := reflect.Select(readerDataCases)
			if !ok {
				//remove the closed channel from the channel array
				//overwrite and shrink
				readerData[chosen] = readerData[len(readerData)-1]
				readerData[len(readerData)-1] = nil
				readerData = readerData[:len(readerData)-1]

				//remove the closed channel from the select case array
				//overwrite and shrink
				readerDataCases[chosen] = readerDataCases[len(readerDataCases)-1]
				readerDataCases[len(readerDataCases)-1] = reflect.SelectCase{}
				readerDataCases = readerDataCases[:len(readerDataCases)-1]
			} else {
				outData <- flow.Interface().(Flow)
			}
		}
		close(outData)
	}()

	go func() {
		for len(readerErrors) != 0 {
			chosen, err, ok := reflect.Select(readerErrorCases)
			if !ok {
				//remove the closed channel from the channel array
				//overwrite and shrink
				readerErrors[chosen] = readerErrors[len(readerErrors)-1]
				readerErrors[len(readerErrors)-1] = nil
				readerErrors = readerErrors[:len(readerErrors)-1]

				//remove the closed channel from the select case array
				//overwrite and shrink
				readerErrorCases[chosen] = readerErrorCases[len(readerErrorCases)-1]
				readerErrorCases[len(readerErrorCases)-1] = reflect.SelectCase{}
				readerErrorCases = readerErrorCases[:len(readerErrorCases)-1]
			} else {
				outErrors <- err.Interface().(error)
			}
		}
		close(outErrors)
	}()

	return outData, outErrors
}
