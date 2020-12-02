package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	prefix = "https://www.ethernodes.org/data?columns%5B0%5D%5Bdata%5D=id&columns%5B0%5D%5Bname%5D=&columns%5B0%5D%5Bsearchable%5D=true&columns%5B0%5D%5Borderable%5D=true&columns%5B0%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B0%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B1%5D%5Bdata%5D=host&columns%5B1%5D%5Bname%5D=&columns%5B1%5D%5Bsearchable%5D=true&columns%5B1%5D%5Borderable%5D=true&columns%5B1%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B1%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B2%5D%5Bdata%5D=isp&columns%5B2%5D%5Bname%5D=&columns%5B2%5D%5Bsearchable%5D=true&columns%5B2%5D%5Borderable%5D=true&columns%5B2%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B2%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B3%5D%5Bdata%5D=country&columns%5B3%5D%5Bname%5D=&columns%5B3%5D%5Bsearchable%5D=true&columns%5B3%5D%5Borderable%5D=true&columns%5B3%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B3%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B4%5D%5Bdata%5D=client&columns%5B4%5D%5Bname%5D=&columns%5B4%5D%5Bsearchable%5D=true&columns%5B4%5D%5Borderable%5D=true&columns%5B4%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B4%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B5%5D%5Bdata%5D=clientVersion&columns%5B5%5D%5Bname%5D=&columns%5B5%5D%5Bsearchable%5D=true&columns%5B5%5D%5Borderable%5D=true&columns%5B5%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B5%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B6%5D%5Bdata%5D=os&columns%5B6%5D%5Bname%5D=&columns%5B6%5D%5Bsearchable%5D=true&columns%5B6%5D%5Borderable%5D=true&columns%5B6%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B6%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B7%5D%5Bdata%5D=lastUpdate&columns%5B7%5D%5Bname%5D=&columns%5B7%5D%5Bsearchable%5D=true&columns%5B7%5D%5Borderable%5D=true&columns%5B7%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B7%5D%5Bsearch%5D%5Bregex%5D=false&columns%5B8%5D%5Bdata%5D=inSync&columns%5B8%5D%5Bname%5D=&columns%5B8%5D%5Bsearchable%5D=true&columns%5B8%5D%5Borderable%5D=true&columns%5B8%5D%5Bsearch%5D%5Bvalue%5D=&columns%5B8%5D%5Bsearch%5D%5Bregex%5D=false&order%5B0%5D%5Bcolumn%5D=0&order%5B0%5D%5Bdir%5D=asc&length=10&search%5Bvalue%5D=&search%5Bregex%5D=false&_="
)

type Page struct {
	Draw            int `json:"draw"`
	RecordsTotal    int `json:"recordsTotal"`
	RecordsFiltered int `json:"recordsFiltered"`
	Data            []struct {
		ID            string    `json:"id"`
		Host          string    `json:"host"`
		Port          int       `json:"port"`
		Client        string    `json:"client"`
		ClientVersion string    `json:"clientVersion"`
		Os            string    `json:"os"`
		LastUpdate    time.Time `json:"lastUpdate"`
		Country       string    `json:"country"`
		InSync        int       `json:"inSync"`
		Isp           string    `json:"isp"`
	} `json:"data"`
}

type PageLoader struct {
	timestamp int64
	client    *http.Client

	requestLimit int

	C chan string
}

func NewLoader(moment time.Time) *PageLoader {
	return &PageLoader{
		timestamp:    moment.UnixNano(),
		client:       &http.Client{},
		requestLimit: 100,

		C: make(chan string, 1),
	}
}

func (l PageLoader) load() {
	defer close(l.C)

	page, err := l.getPage(0, l.requestLimit)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("loading %d records \n", page.RecordsTotal)

	pagesNum := page.RecordsTotal/l.requestLimit + 1
	for i := 0; i < pagesNum; i++ {
		page, err := l.getPage(i+1, i*l.requestLimit)
		if err != nil {
			log.Fatal(err)
		}

		for j := 0; j < 10; j++ {
			str := fmt.Sprintf("enode://%s@%s:%d\n", page.Data[j].ID, page.Data[j].Host, page.Data[j].Port)
			l.C <- str
		}
	}
}

// step is just a sequential request number after HTML page load
func (l PageLoader) getPage(step, offset int) (*Page, error) {
	page := &Page{}
	url := prefix + strconv.FormatInt(int64(l.timestamp), 10) +
		"&start=" + strconv.FormatInt(int64(offset), 10) +
		"&draw=" + strconv.FormatInt(int64(step), 10)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	res, getErr := l.client.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	err = json.Unmarshal(body, page)
	if err != nil {
		return nil, err
	}

	return page, nil
}

func main() {
	now := time.Now()

	defaultFilename := fmt.Sprintf("nodes-%s.txt", now.UTC().Format(time.RFC3339))
	filename := flag.String("filename", defaultFilename, "file for output")

	file, err := os.OpenFile(*filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	loader := NewLoader(now)
	go loader.load()

	for row := range loader.C {
		_, err := file.Write([]byte(row))
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("records dumped to %s \n", *filename)
}
