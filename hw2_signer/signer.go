package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync/atomic"
)

// сюда писать код

var needCombine uint32
var processCombine uint32

func SingleHash(in, out chan interface{}) {
	for val := range in {
		data := fmt.Sprintf("%d", val)
		singleHash := DataSignerCrc32(data) + "~" + DataSignerCrc32(DataSignerMd5(data))
		out <- singleHash
	}
}

func MultiHash(in, out chan interface{}) {
	for val := range in {
		data := val.(string)
		multiHash := ""
		for th := 0; th <= 5; th++ {
			multiHash = multiHash + DataSignerCrc32(strconv.Itoa(th)+data)
		}
		out <- multiHash
	}
}

func CombineResults(in, out chan interface{}) {
	var data []string
	for val := range in {
		data = append(data, val.(string))
		atomic.AddUint32(&processCombine, 1)
		if needCombine == atomic.LoadUint32(&processCombine) {
			close(in)
		}
	}
	sort.Strings(data)
	prefix := ""
	combineResults := ""
	for _, val := range data {
		combineResults = combineResults + prefix + val
		prefix = "_"
	}
	fmt.Println(combineResults)
	out <- combineResults
}

func ExecutePipeline(jobs ...job) {
	out := make(chan interface{}, 100)

	jobs[0](make(chan interface{}), out)
	atomic.StoreUint32(&needCombine, uint32(len(out)))
	inForNext := out
	for _, jober := range jobs[1 : len(jobs)-1] {
		outForNext := make(chan interface{}, 100)
		go jober(inForNext, outForNext)
		inForNext = outForNext
	}

	jobs[len(jobs)-1](inForNext, make(chan interface{}))
}
