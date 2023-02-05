package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type HashData struct {
	Hashed string
	Data   string
}

// сюда писать код
// SingleHash считает значение crc32(data)+"~"+crc32(md5(data))
// ( конкатенация двух строк через ~), где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	preventOverheat := make(chan interface{}, 1)
	for elem := range in {
		elemStr := fmt.Sprintf("%v", elem)
		// hashedElem := fmt.Sprintf("%s~%s", DataSignerCrc32(elemStr), DataSignerCrc32(DataSignerMd5(elemStr)))
		newChan := make(chan interface{}, 1)
		// mu := &sync.Mutex{}
		wg.Add(1)
		go func(elemStr string, out chan interface{}, preventOverheat chan interface{}) {
			defer wg.Done()
			preventOverheat <- nil
			hashedData := DataSignerMd5(elemStr)
			<-preventOverheat
			newhashedData := DataSignerCrc32(hashedData)
			out <- newhashedData
		}(elemStr, newChan, preventOverheat)
		// out <- hashedElem
		wg.Add(1)
		go func(elemStr string, in chan interface{}, out chan interface{}) {
			defer wg.Done()
			hashedData := DataSignerCrc32(elemStr)
			chanData := <-in
			res := hashedData + "~" + chanData.(string)
			out <- res
		}(elemStr, newChan, out)

	}
	wg.Wait()
}

//   - MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки),
//     где th=0..5 ( т.е. 6 хешей на каждое входящее значение ),
//     потом берёт конкатенацию результатов в порядке расчета (0..5), где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for elem := range in {
		data := elem.(string)
		hashedArr := make([]string, 6)
		intWg := &sync.WaitGroup{}
		mu := &sync.Mutex{}
		for i := 0; i < 6; i++ {
			intWg.Add(1)
			// hashedElem += DataSignerCrc32(fmt.Sprintf("%v%s", i, elem.(string)))
			go func(i int, hashStr string) {
				defer intWg.Done()
				hashedData := DataSignerCrc32(fmt.Sprintf("%v%s", i, hashStr))
				mu.Lock()
				hashedArr[i] = hashedData
				mu.Unlock()
			}(i, data)
		}
		wg.Add(1)
		go func(out chan interface{}) {
			defer wg.Done()
			intWg.Wait()
			res := strings.Join(hashedArr, "")
			out <- res
		}(out)

		// out <- hashedElem
	}
	wg.Wait()
}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	hashArr := []string{}
	for elem := range in {
		hashArr = append(hashArr, elem.(string))
	}
	sort.Strings(hashArr)
	res := strings.Join(hashArr, "_")
	out <- res
}

func ExecutePipeline(hashRes ...job) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	in := make(chan interface{})
	for _, elem := range hashRes {
		out := make(chan interface{})
		wg.Add(1)
		go func(myJob job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)
			myJob(in, out)
		}(elem, in, out)
		in = out
	}
}
