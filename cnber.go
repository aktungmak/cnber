package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
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

//number of data points per slot
const FIELDS_PER_SLOT = 4

//HTTP response timeout in seconds
const TIMEOUT = 5

//API uri ready for fmt
const API_ENDP = "http://%s/api/profiles/~1/sandbox/~0/inputTransportStreams/~%d/sources"

// represents a single descrambler card within an RX9500 chassis
type Descrambler struct {
	slot int
	ipa  string
	ber  string
	cnr  float64
	cnm  float64
	slv  float64
}

// contact unit via http and read the BER and CN value
func (d *Descrambler) Update(done chan bool) {
	defer func() { done <- true }()

	url := fmt.Sprintf(API_ENDP, d.ipa, d.slot)

	// make http request
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error contacting %s", d.ipa)
		return
	}
	defer resp.Body.Close()

	// read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response from %s", d.ipa)
		return
	}

	//deserialise
	apiresp, err := ApiResponseFromString(body)
	if err != nil {
		log.Printf("Error parsing JSON response from %s", d.ipa)
		return
	}

	// write values to struct
	d.ber = apiresp.GetBer()
	d.cnr = apiresp.GetCnr()
	d.cnm = apiresp.GetCnm()
	d.slv = apiresp.GetSlv()
}

// json struct to deserialise the response from REST
// This is liable to change, so I have provided accessor
// methods for modularity. If the API changes, just need to
// update these and everything else follows
type ApiResponse struct {
	Collection struct {
		Items []struct {
			Data struct {
				Ber struct {
					Value string `json:"value"`
				} `json:"ber"`
				Cnr struct {
					Value float64 `json:"value"`
				} `json:"carrierToNoiseRatio"`
				Cnm struct {
					Value float64 `json:"value"`
				} `json:"carrierToNoiseMargin"`
				Slv struct {
					Value float64 `json:"value"`
				} `json:"signalLevel"`
			} `json:"data"`
		} `json:"items"`
	} `json:"collection"`
}

func ApiResponseFromString(jsonstr []byte) (*ApiResponse, error) {
	newar := ApiResponse{}

	err := json.Unmarshal(jsonstr, &newar)
	if err != nil {
		return nil, err
	}

	return &newar, nil
}

func (ar *ApiResponse) GetBer() string {
	return ar.Collection.Items[0].Data.Ber.Value
}

func (ar *ApiResponse) GetCnr() float64 {
	return ar.Collection.Items[0].Data.Cnr.Value
}

func (ar *ApiResponse) GetCnm() float64 {
	return ar.Collection.Items[0].Data.Cnm.Value
}

func (ar *ApiResponse) GetSlv() float64 {
	return ar.Collection.Items[0].Data.Slv.Value
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
	ret := make([]*Descrambler, len(hosts)*6)
	for i, host := range hosts {
		for j := 0; j < NUM_SLOTS; j++ {
			ret[(i*NUM_SLOTS)+j] = &Descrambler{
				ipa:  host,
				slot: j + 1,
			}
		}
	}
	return ret
}

// creates a header line for the csv, including a timestamp column
func CreateCsvHeader(descramblers []*Descrambler) []string {
	hdr := make([]string, (len(descramblers)*FIELDS_PER_SLOT)+1)
	hfmt := "%s|%d|%s"
	hdr[0] = "Timestamp"
	for i, desc := range descramblers {
		hdr[i*FIELDS_PER_SLOT+1] = fmt.Sprintf(hfmt, desc.ipa, desc.slot, "BER")
		hdr[i*FIELDS_PER_SLOT+2] = fmt.Sprintf(hfmt, desc.ipa, desc.slot, "CN Ratio")
		hdr[i*FIELDS_PER_SLOT+3] = fmt.Sprintf(hfmt, desc.ipa, desc.slot, "CN Margin")
		hdr[i*FIELDS_PER_SLOT+4] = fmt.Sprintf(hfmt, desc.ipa, desc.slot, "Sig Level")
	}
	return hdr
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func main() {
	// TODO cmdline args
	var infname, utfname string
	var iwait int

	flag.StringVar(&infname, "i", "REQUIRED",
		"Input list of RX9500 IP addresses")
	flag.StringVar(&utfname, "o", "REQUIRED",
		"Output CSV file name")
	flag.IntVar(&iwait, "w", 30,
		"Time in seconds to wait between rounds of polling")
	flag.Parse()
	if infname == "REQUIRED" || utfname == "REQUIRED" {
		flag.PrintDefaults()
		log.Fatal("Error: Missing arguments!\n\n")
	}

	wait := time.Duration(iwait) * time.Second

	fmt.Println("Starting...")

	hosts, err := ParseAddrFile(infname)
	check(err)

	outfile, err := os.Create(utfname)
	check(err)
	defer outfile.Close()
	csvout := csv.NewWriter(outfile)

	// make struct array
	descramblers := CreateDescramblerArray(hosts)

	// write csv header
	header := CreateCsvHeader(descramblers)
	csvout.Write(header)

	// this keeps track of the descrambler.Update()s finishing
	done := make(chan bool)

	// iterate forever
	for {
		// save start time of this run
		starttime := time.Now()

		// contact each Descrambler and update their current state
		for _, descrambler := range descramblers {
			go descrambler.Update(done)
		}

		// need to wait for them to finish!
		for i := 0; i < len(descramblers); i++ {
			<-done
		}

		// now that is done, write it all to the file
		var record []string
		record = append(record, starttime.Format(time.RFC1123))
		for _, descrambler := range descramblers {
			record = append(record, fmt.Sprint(descrambler.ber))
			record = append(record, fmt.Sprint(descrambler.cnr))
			record = append(record, fmt.Sprint(descrambler.cnm))
			record = append(record, fmt.Sprint(descrambler.slv))
		}
		csvout.Write(record)
		csvout.Flush()

		log.Printf("Finished gathering data in %s", time.Since(starttime))
		time.Sleep(wait)
	}
}
