package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

// сюда писать код

func SingleHash(in, out chan interface{}) {

	var muMD5 sync.Mutex

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

	inProcess := make(chan interface{}, 100)
	wg := &sync.WaitGroup{}
	go processMultiHash(wg, inProcess, out)
	for ch_singleHash := range in {
		val := <-ch_singleHash.(chan interface{})
		data := val.(string)
		chans := make([]<-chan string, base)
		for th := 0; th < base; th++ {
			chans[th] = dataSignerCrc32(strconv.Itoa(th) + data)
		}
		wg.Add(1)
		inProcess <- chans
	}
	wg.Wait()
	close(inProcess)

}

func processMultiHash(wg *sync.WaitGroup, inProcess chan interface{}, out chan interface{}) {
	for val := range inProcess {
		chans := val.([]<-chan string)
		multiHash := ""
		for _, chanCrc := range chans {
			multiHash2 := <-chanCrc
			multiHash = multiHash + multiHash2
		}
		out <- multiHash
		wg.Done()
	}
}

func CombineResults(in, out chan interface{}) {

	var data []string
	for val := range in {
		data = append(data, val.(string))
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

func runner(wg *sync.WaitGroup, in, out chan interface{}, j job) {
	j(in, out)
	close(out)
	wg.Done()
}

func ExecutePipeline(jobs ...job) {
	size := 100
	in := make(chan interface{}, size)
	out := make(chan interface{}, size)

	wg := &sync.WaitGroup{}
	for _, jober := range jobs {
		wg.Add(1)
		go runner(wg, in, out, jober)
		in = out
		out = make(chan interface{}, size)
	}
	wg.Wait()
}
