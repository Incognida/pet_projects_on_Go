package main

import (
	"bufio"
	"flag"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

type urlCounter struct {
	Url       string
	Count     int64
	ErrString string
}

func crawl(c chan *urlCounter, url string) {
	uC := new(urlCounter)
	uC.Url = url
	resp, err := http.Get(url)
	if err != nil {
		uC.ErrString = err.Error()
		c <- uC
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		uC.ErrString = err.Error()
		c <- uC
		return
	}

	reg, err := regexp.Compile("(([\\n ]|)Go( |))")
	if err != nil {
		uC.ErrString = err.Error()
		c <- uC
		return
	}
	index := suffixarray.New([]byte(data))
	matches := index.FindAllIndex(reg, -1)
	uC.Count = int64(len(matches))
	c <- uC

}

func main() {
	numOfGors := flag.Int(
		"k", 2, "number of concurrent/parallel goroutines")
	flag.Parse()

	urlArr := make([]*urlCounter, 0)
	scanner := bufio.NewScanner(os.Stdin)
	ch := make(chan *urlCounter, *numOfGors)
	fmt.Println(*numOfGors)

	isEOF := false

	for !isEOF {
		i := 0
		for i < *numOfGors {
			isEOF = !scanner.Scan()
			if isEOF {
				break
			}
			line := scanner.Text()
			go crawl(ch, line)
			i++
		}

		for i > 0 {
			data := <-ch
			urlArr = append(urlArr, data)
			if data.ErrString != "" {
				fmt.Println(data.ErrString)
			}
			i--
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, elem := range urlArr {
		fmt.Println(elem.Url, elem.Count, elem.ErrString)
	}
}
