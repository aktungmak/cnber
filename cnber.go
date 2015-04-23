package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

//assumed number of slots in chassis
const NUM_SLOTS = 6

//HTTP response timeout in seconds
const TIMEOUT = 5

//API uri ready for fmt
const API_ENDP = "http://%s/%d/"

// represents a single descrambler within an RX9500 chassis
type Descrambler struct {
	slot int
	ipa  string
	ber  float64
	cn   float64
}

// contact unit via http and read the BER and CN value
func (d *Descrambler) Update(done chan bool) {
	defer func() { done <- true }()

	url := fmt.Sprintf(API_ENDP, d.ipa, d.slot)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error contacting %s", d.ipa)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s", d.ipa)
		return
	}

	log.Printf("%s", body)
	//deserialise

}

// TODO make a json struct to deserialise the response
type ApiResponse struct {
	ber string `json:"ber"`
	cn  string `json:"cn"`
}

func NewApiResponse(json string) (newar ApiResponse) {

	return newar
}

// read ip addr file and return slice of hosts
func ParseAddrFile(fname string) ([]string, error) {
	var lines []string
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		txt := scanner.Text()
		isip := net.ParseIP(txt)
		if isip != nil {
			lines = append(lines, txt)
		}
	}

	return lines, scanner.Err()
}

// take a slice of hosts and make Descramblers for each option card
// we assume each host has NUM_SLOTS Descramblers in it
// return a slice of pointers to these Descramblers
func CreateDescramblerArray(hosts []string) []*Descrambler {
	ret := make([]*Descrambler, len(hosts))
	for i, host := range hosts {
		ret[i] = &Descrambler{
			ipa: host,
		}
	}
	return ret
}

// creates a header line for the csv, putting BER then CN
func CreateCsvHeader(descramblers []*Descrambler) []string {
	hdr := make([]string, len(descramblers)+1)
	hdr[0] = "Timestamp"

	return hdr
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func main() {
	fmt.Println("Starting...")

	// TODO cmdline args
	infname := "in.txt"
	utfname := "out.txt"
	wait := time.Duration(30)

	hosts, err := ParseAddrFile(infname)
	check(err)

	outfile, err := os.Create(utfname)
	check(err)
	defer outfile.Close()
	csvout := csv.NewWriter(outfile)

	// make struct array
	descramblers := CreateDescramblerArray(hosts)

	log.Printf("%v", descramblers)

	// write csv header
	header := CreateCsvHeader(descramblers)
	csvout.Write(header)

	// this keeps track of the descrambler.Update()s finishing
	done := make(chan bool)

	// iterate forever
	for {
		// save start time of this run
		// starttime := time.Now()

		// contact each Descrambler and update their current state
		for _, descrambler := range descramblers {
			go descrambler.Update(done)
		}

		// need to wait for them to finish!
		for i := 0; i < len(descramblers); i++ {
			<-done
		}

		// now that is done, write it all to the file
		// var record []string
		// append(record, starttime.Format(time.RFC822))
		// for _, descrambler := range descramblers {
		// 	append(record, fmt.Sprint(descrambler.ber))
		// 	append(record, fmt.Sprint(descrambler.cn))
		// }
		// csvout.Write(record)
		break

		// finally, have a little rest
		time.Sleep(wait * time.Second)
	}

}

//
