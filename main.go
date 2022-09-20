package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/kelvins/geocoder" // could just call the API directly, but i'm lazy
)

var googleMapsAPIKey = flag.String("apiKey", "", "--apiKey=1111222233334444555566667777888899990000 - see https://developers.google.com/maps/documentation/geocoding/get-api-key")

// TODO: accept an address (once we figure out how to do esri geometry conversions).

type Geometry struct {
	SpatialReference *SpatialReference `json:"spatialReference"`
	X                float64           `json:"x"`
	Y                float64           `json:"y"`
}

type SpatialReference struct {
	LatestWKID int `json:"latestWkid"`
	WKID       int `json:"wkid"`
}

type Result struct {
	Features []Feature `json:"features"`
}

type Feature struct {
	Attritubes Attributes `json:"attributes"`
}

type Attributes struct {
	State                string  `json:"STATE"`
	County               string  `json:"COUNTY"`
	RiskScore            float64 `json:"RISK_SCORE"`
	RiskRating           string  `json:"RISK_RATNG"`
	DroughtRiskScore     float64 `json:"DRGT_RISKS"`
	DroughtRiskRating    string  `json:"DRGT_RISKR"`
	EarthquakeRiskScore  float64 `json:"ERQK_RISKS"`
	EarthquakeRiskRating string  `json:"ERQK_RISKR"`
	TornadoRiskScore     float64 `json:"TRND_RISKS"`
	TornadoRiskRating    string  `json:"TRND_RISKR"`
}

func (a Attributes) String() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "State:", a.State)
	fmt.Fprintln(b, "County:", a.County)
	fmt.Fprintf(b, "RiskScore: %.2f\n", a.RiskScore)
	fmt.Fprintln(b, "RiskRating:", a.RiskRating)
	fmt.Fprintf(b, "DroughtRiskScore: %.2f\n", a.DroughtRiskScore)
	fmt.Fprintln(b, "DroughtRiskRating:", a.DroughtRiskRating)
	fmt.Fprintf(b, "EarthquakeRiskScore: %.2f\n", a.EarthquakeRiskScore)
	fmt.Fprintln(b, "EarthquakeRiskRating:", a.EarthquakeRiskRating)
	fmt.Fprintf(b, "TornadoRiskScore: %.2f\n", a.TornadoRiskScore)
	fmt.Fprintln(b, "TornadoRiskRating:", a.TornadoRiskRating)
	return b.String()
}

func main() {
	flag.Parse()

	if *googleMapsAPIKey == "" {
		fmt.Println("please provide a value for all --apiKey")
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	x, y, err := esriGeometryPointForAddress()
	if err != nil {
		return fmt.Errorf("error getting geometry for address: %v", err)
	}
	g := &Geometry{
		SpatialReference: &SpatialReference{LatestWKID: 3857, WKID: 102100},
		X:                x,
		Y:                y,
	}
	gBytes, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("error marshaling Geometry: %v", err)
	}

	u, err := url.Parse("https://services.arcgis.com/XG15cJAlne2vxtgt/arcgis/rest/services/National_Risk_Index_Counties/FeatureServer/0/query")
	if err != nil {
		return fmt.Errorf("error parsing arcgis url: %v", err)
	}
	v := u.Query()
	v.Add("geometry", string(gBytes))
	v.Add("f", "json")
	v.Add("outFields", "*")
	v.Add("spatialRel", "esriSpatialRelIntersects")
	v.Add("where", "1=1")
	v.Add("geometryType", "esriGeometryPoint")
	v.Add("inSR", "102100")
	v.Add("outSR", "102100")
	u.RawQuery = v.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("error querying arcgis: %v", err)
	}
	outBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading arcgis response: %v", err)
	}
	var res Result
	if err := json.Unmarshal(outBytes, &res); err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}

	if l := len(res.Features); l != 1 {
		return fmt.Errorf("expected 1 Features, but got %d", l)
	}

	fmt.Println("Here's some risk information about your home:")
	fmt.Println(res.Features[0].Attritubes)

	return nil
}

func esriGeometryPointForAddress() (x, y float64, _ error) {
	address := geocoder.Address{
		Street:  "Central Park West",
		Number:  115,
		City:    "New York",
		State:   "New York",
		Country: "United States",
	}
	geocoder.ApiKey = *googleMapsAPIKey
	location, err := geocoder.Geocoding(address)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get coords from google maps: %v", err)
	}
	_ = location
	// TODO: convert location lat/lon to arcgis esri geometry point, and return them.

	// Dummy values, mined from https://hazards.fema.gov/nri/map#.
	return -11677620.771308051, 4854685.755719238, nil
}
