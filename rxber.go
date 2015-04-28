package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/soniah/gosnmp"
	"net"
	"os"
	"strings"
)

// these fields are common to all devices, so only define them
// once and embed in each struct
type Descrambler struct {
	ipa    string
	ber    string
	cnr    string
	cnm    string
	slv    string
	berOid string
	cnrOid string
	cnmOid string
	slvOid string
}

func (d *Descrambler) UpdateAll() {
	d.UpdateBer()
	d.UpdateCnr()
	d.UpdateCnm()
	d.UpdateSlv()
}
func (d *Descrambler) UpdateBer() {
	res, err := RequestOidAsString(d.ipa, d.berOid)
	if err == nil {
		d.ber = res
	}
}
func (d *Descrambler) UpdateCnr() {
	res, err := RequestOidAsString(d.ipa, d.cnrOid)
	if err == nil {
		d.cnr = res
	}
}
func (d *Descrambler) UpdateCnm() {
	res, err := RequestOidAsString(d.ipa, d.cnmOid)
	if err == nil {
		d.cnm = res
	}
}
func (d *Descrambler) UpdateSlv() {
	res, err := RequestOidAsString(d.ipa, d.slvOid)
	if err == nil {
		d.slv = res
	}
}

func RequestOidAsString(ipa string, oids ...string) (string, error) {
	gosnmp.Default.Target = ipa
	err := gosnmp.Default.Connect()
	if err != nil {
		return "-", err
	}
	defer gosnmp.Default.Conn.Close()

	res, err := gosnmp.Default.Get(oids)
	if err != nil {
		return "-", err
	}

	switch res.Variables[0].Type {
	case gosnmp.OctetString:
		return string(res.Variables[0].Value.([]byte)), nil
	case gosnmp.Integer:
		return fmt.Sprintf("%d", res.Variables[0].Value), nil
	default:
		return "-", errors.New("Unknown BER type!!")
	}

}

func NewRx8200(ipa string) (ret *Descrambler) {
	ret = &Descrambler{}
	ret.ipa = ipa
	ret.slvOid = "1.3.6.1.4.1.1773.1.3.208.2.2.3.0"
	ret.berOid = "1.3.6.1.4.1.1773.1.3.208.2.2.4.0"
	ret.cnrOid = "1.3.6.1.4.1.1773.1.3.208.2.2.5.0"
	ret.cnmOid = "1.3.6.1.4.1.1773.1.3.208.2.2.6.0"

	return ret
}

func NewRx1290(ipa string) (ret *Descrambler) {
	ret = &Descrambler{}
	ret.ipa = ipa
	ret.berOid = "1.3.5.1.4.1.1773.1.3.200.4.1.4.0"
	ret.cnrOid = "1.3.6.1.4.1.1773.1.3.200.4.3.3.1.6.3.0"
	ret.cnmOid = "1.3.6.1.4.1.1773.1.3.200.4.3.3.1.2.0"
	ret.slvOid = "1.3.6.1.4.1.1773.1.3.200.4.3.3.1.6.4.0"

	return ret
}

func NewTt1222(ipa string) (ret *Descrambler) {
	ret = &Descrambler{}
	ret.ipa = ipa
	ret.berOid = ""
	ret.cnrOid = ""
	ret.cnmOid = ""
	ret.slvOid = ""

	return ret
}

// read ip addr file and return slice of hosts
func ParseConfigFile(fname string) ([]*Descrambler, error) {
	var units []*Descrambler
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		txt := strings.Split(scanner.Text(), " ")
		isip := net.ParseIP(txt[0])
		if isip == nil {
			continue
		}
		var newunit *Descrambler
		switch txt[1] {
		case "8200":
			newunit = NewRx8200(txt[0])

		case "1290":
			newunit = NewRx1290(txt[0])

		case "1222":
			newunit = NewTt1222(txt[0])

		default:
			continue
		}

		units = append(units, newunit)
	}

	return units, scanner.Err()
}

func main() {

	rx := NewRx8200("192.168.32.155")
	rx.UpdateAll()
	fmt.Printf("%v", rx)

	// parse cmdline flags

	//read config file and create unit array

	//write CSV header

	//for ever
	//for unit
	//update values

	// write to CSV

}
