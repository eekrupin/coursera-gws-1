package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// сюда писать код

var needCombine uint32
var processCombine uint32
var muMD5 sync.Mutex
var Sh time.Duration

func SingleHash(in, out chan interface{}) {

	for val := range in {
		data := fmt.Sprintf("%d", val)

		ch_crc32_1 := dataSignerCrc32(data)
		muMD5.Lock()
		md5 := DataSignerMd5(data)
		muMD5.Unlock()
		ch_crc32_2 := dataSignerCrc32(md5)
		ch_singleHash := func(crc32_1, crc32_2 chan string) chan interface{} {
			out := make(chan interface{})
			go func(crc32_1, crc32_2 chan string, out chan interface{}) {
				out <- <-crc32_1 + "~" + <-crc32_2
			}(crc32_1, crc32_2, out)
			return out
		}(ch_crc32_1, ch_crc32_2)
		out <- ch_singleHash
	}

}

func dataSignerCrc32(data string) chan string {

	out := make(chan string)
	go func(data string, out chan<- string) {

		out <- DataSignerCrc32(data)

	}(data, out)

	return out
}

func MultiHash(in, out chan interface{}) {

	base := 6
	for ch_singleHash := range in {
		//st := time.Now()
		val := <-ch_singleHash.(chan interface{})
		//fmt.Println("11", time.Since(st))
		data := val.(string)
		chans := make([]<-chan string, base)
		multiHash := ""
		for th := 0; th < base; th++ {
			val_th := th
			chans[th] = dataSignerCrc32(strconv.Itoa(val_th) + data)
			//multiHash = multiHash + DataSignerCrc32(strconv.Itoa(th)+data)
		}
		for _, chanCrc := range chans {
			multiHash = multiHash + <-chanCrc
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

		/*if ind == 1{
		fmt.Println("jober", runtime.FuncForPC(reflect.ValueOf(jober).Pointer()).Name())
		st := time.Now()
		for _ = range outForNext {

			end := time.Since(st)
			//muMD5.Lock()
			//Sh = Sh + end
			//muMD5.Unlock()
			fmt.Println("Sh", end)
			st = time.Now()
		}
		}*/

		inForNext = outForNext
	}

	jobs[len(jobs)-1](inForNext, make(chan interface{}))

}
