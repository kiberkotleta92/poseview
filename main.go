package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"time"
)

const HOST = "https://proteins.plus/api/poseview_rest"

type poseviewRequest map[string]map[string]string

type poseviewResponse struct {
	Status   int    `json:"status_code"`
	Message  string `json:"message"`
	Location string `json:"location"`
	Err      string `json:"error"`
	Png      string `json:"result_png_picture"`
	Pdf      string `json:"result_pdf_picture"`
	Svg      string `json:"result_svg_picture"`
}

type httpClient struct {
	*http.Client
}

func (client *httpClient) makeRequest(pdb, ligand string) (poseviewResponse, error) {

	var result poseviewResponse

	req := poseviewRequest{"poseview": {"pdbCode": pdb, "ligand": ligand}}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return result, err
	}

	request, err := http.NewRequest("POST", HOST, bytes.NewBuffer(reqBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}
	if result.Err != "" {
		return result, fmt.Errorf(result.Message)
	}
	log.Println(result.Message)
	return result, nil
}

func (client *httpClient) processResponse(firstResp poseviewResponse) (poseviewResponse, error) {
	var result poseviewResponse

	resp, err := client.Get(firstResp.Location)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}
	if result.Err != "" {
		return result, fmt.Errorf(result.Message)
	}
	for result.Status == 202 {
		time.Sleep(1 * time.Second)
		log.Println(result.Message)
	}

	return result, nil
}

func (client *httpClient) loadImage(firstResp poseviewResponse, format string) error {
	var image string

	switch format {
	case "png":
		image = firstResp.Png
	case "pdf":
		image = firstResp.Pdf
	case "svg":
		image = firstResp.Svg
	}

	filename := path.Base(image)

	resp, err := client.Get(image)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, body, 0644); err != nil {
		return err
	}
	log.Println("Loaded ", filename)

	return nil
}

func main() {

	format := flag.String("format", "png", "format of file {png|pdf|svg}")
	flag.Usage = func() {
		fmt.Println("Usage: poseview [-format] pdb ligand\n",
			" pdb string\n\tselect a structure form the Protein Data Bank (PDB) via its PDB code\n",
			" ligand string\n\tset a ligand with respect to specified pdbCode or \"\"",
		)
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		flag.Usage()
		return
	}

	client := &httpClient{&http.Client{Timeout: time.Duration(10 * time.Second)}}

	fresp, err := client.makeRequest(args[0], args[1])
	if err != nil {
		log.Fatal(err)
	}

	sresp, err := client.processResponse(fresp)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.loadImage(sresp, *format); err != nil {
		log.Fatal(err)
	}
}

// func main() {
// 	format := flag.String("format", "png", "format of file {png|pdf|svg}")
// 	flag.Usage = func() {
// 		fmt.Println("Usage: poseview [-format] [-region] pdbCode ligand")
// 		flag.PrintDefaults()
// 	}
// 	flag.Parse()
// 	args := flag.Args()

// 	if len(args) < 3 {
// 		flag.Usage()
// 		return
// 	}

// 	fmt.Println(format)
// }
