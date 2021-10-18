package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"strings"

	"github.com/adrianrudnik/base45-go"
	"github.com/fxamacker/cbor/v2"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// https://github.com/fxamacker/cbor#go-struct-tags
type coseHeader struct {
	Alg int    `cbor:"1,keyasint,omitempty"`
	Kit []byte `cbor:"4,keyasint,omitempty"`
}

type signedCWT struct {
	_           struct{} `cbor:",toarray"`
	Protected   []byte
	Unprotected coseHeader
	Payload     []byte
	Signature   []byte
}

type Payload struct {
	QRCodeIssuerCountry string `cbor:"1,keyasint" json:"1"`
	QRCodeExpiry        int64  `cbor:"4,keyasint" json:"4"`
	QRCodeGenerated     int64  `cbor:"6,keyasint" json:"6"`
	Payload             struct {
		Data DCC `cbor:"1,keyasint" json:"1"`
	} `cbor:"-260,keyasint" json:"-260"`
}

// https://github.com/ehn-dcc-development/ehn-dcc-schema/blob/release/1.3.0/DCC.schema.json
type DCC struct {
	Name            Name            `cbor:"nam" json:"nam"`
	DateOfBirth     string          `cbor:"dob" json:"dob"`
	Version         string          `cbor:"ver" json:"ver"`
	VaccineEntries  []VaccineEntry  `cbor:"v" json:"v,omitempty"`
	RecoveryEntries []RecoveryEntry `cbor:"r" json:"r,omitempty"`
	// TestEntry not implemented
	// TestEntries []TestEntry `cbor:"t" json:"t"`
}

// https://github.com/ehn-dcc-development/ehn-dcc-schema/blob/release/1.3.0/DCC.Types.schema.json
type VaccineEntry struct {
	Target                      string `cbor:"tg" json:"tg"`
	VaccineOrProphylaxis        string `cbor:"vp" json:"vp"`
	VaccineMedicinalProduct     string `cbor:"mp" json:"mp"`
	MarketingAuthHolder         string `cbor:"ma" json:"ma"`
	DoseNumber                  int64  `cbor:"dn" json:"dn"`
	TotalSeriesOfDoses          int64  `cbor:"sd" json:"sd"`
	DateOfVaccination           string `cbor:"dt" json:"dt"`
	CountryOfVaccination        string `cbor:"co" json:"co"`
	CertificateIssuer           string `cbor:"is" json:"is"`
	UniqueCertificateIdentifier string `cbor:"ci" json:"ci"`
}

type RecoveryEntry struct {
	Target                      string `cbor:"tg" json:"tg"`
	FirstPositiveResultDate     string `cbor:"fr" json:"fr"`
	CountryOfTest               string `cbor:"co" json:"co"`
	CertificateIssuer           string `cbor:"is" json:"is"`
	CertificateValidFrom        string `cbor:"df" json:"df"`
	CertificateValidUntil       string `cbor:"du" json:"du"`
	UniqueCertificateIdentifier string `cbor:"ci" json:"ci"`
}

// https://ec.europa.eu/health/sites/default/files/ehealth/docs/covid-certificate_json_specification_en.pdf
type Name struct {
	FirstName             string `cbor:"gn" json:"gn"`
	FirstNameStandardized string `cbor:"gnt" json:"gnt"`
	LastName              string `cbor:"fn" json:"fn"`
	LastNameStandardized  string `cbor:"fnt" json:"fnt"`
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Error:\nUsage: covid-cert-reader <pngfilename>")
		os.Exit(1)
	}

	err := run(os.Args[1])
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func parseQRCodeFromPNGFile(pngFilename string) (string, error) {
	file, err := os.Open(pngFilename)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return "", err
	}

	return result.GetText(), nil
}

func base45Decode(qr string) ([]byte, error) {
	// strip "HC1:" from the beginning
	qr = strings.TrimPrefix(qr, "HC1:")

	decoded, err := base45.Decode([]byte(qr))
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func zlibUncompress(base45qr []byte) (io.ReadCloser, error) {
	rc, err := zlib.NewReader(bytes.NewReader(base45qr))
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func cborDecode(r io.Reader) (*Payload, error) {
	decoder := cbor.NewDecoder(r)

	// We're ignoring signature verification, since couldn't find
	// the keys THL uses online. They can gotten by emailing THL, see
	// https://github.com/eu-digital-green-certificates/dgc-participating-countries/issues/10#issuecomment-873050445
	cwt := signedCWT{}
	err := decoder.Decode(&cwt)
	if err != nil {
		return nil, err
	}

	pl := &Payload{}

	payloadDecoder := cbor.NewDecoder(bytes.NewBuffer(cwt.Payload))

	err = payloadDecoder.Decode(pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

func run(pngFilename string) error {
	qr, err := parseQRCodeFromPNGFile(pngFilename)
	if err != nil {
		return err
	}

	b45decoded, err := base45Decode(qr)
	if err != nil {
		return err
	}

	unzlibed, err := zlibUncompress(b45decoded)
	if err != nil {
		return err
	}
	defer unzlibed.Close()

	pl, err := cborDecode(unzlibed)
	if err != nil {
		return err
	}

	// Print as JSON format
	b, err := json.Marshal(pl)
	if err != nil {
		return err
	}
	fmt.Println(string(b))

	return nil
}
