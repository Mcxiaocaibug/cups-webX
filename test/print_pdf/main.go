package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OpenPrinting/goipp"
)

const PrinterURL = "http://localhost:631/printers/EPSON_L380_Series"

func checkErr(err error, format string, args ...interface{}) {
	if err != nil {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
		os.Exit(1)
	}
}

func openTestPage() (*os.File, error) {
	candidates := []string{
		filepath.Join("test", "page.pdf"),
		filepath.Join("..", "page.pdf"),
		"page.pdf",
	}
	for _, candidate := range candidates {
		f, err := os.Open(candidate)
		if err == nil {
			return f, nil
		}
	}
	return nil, os.ErrNotExist
}

func main() {
	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpPrintJob, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset",
		goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language",
		goipp.TagLanguage, goipp.String("en-US")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri",
		goipp.TagURI, goipp.String(PrinterURL)))
	req.Operation.Add(goipp.MakeAttribute("requesting-user-name",
		goipp.TagName, goipp.String("John Doe")))
	req.Operation.Add(goipp.MakeAttribute("job-name",
		goipp.TagName, goipp.String("job name")))
	req.Operation.Add(goipp.MakeAttribute("document-format",
		goipp.TagMimeType, goipp.String("application/pdf")))

	payload, err := req.EncodeBytes()
	checkErr(err, "IPP encode")

	file, err := openTestPage()
	checkErr(err, "Open document file")
	defer file.Close()

	body := io.MultiReader(bytes.NewBuffer(payload), file)

	httpReq, err := http.NewRequest(http.MethodPost, PrinterURL, body)
	checkErr(err, "HTTP")

	httpReq.Header.Set("content-type", goipp.ContentType)
	httpReq.Header.Set("accept", goipp.ContentType)

	httpRsp, err := http.DefaultClient.Do(httpReq)
	if httpRsp != nil {
		defer httpRsp.Body.Close()
	}

	checkErr(err, "HTTP")

	if httpRsp.StatusCode/100 != 2 {
		checkErr(errors.New(httpRsp.Status), "HTTP")
	}

	rsp := &goipp.Message{}
	err = rsp.Decode(httpRsp.Body)
	checkErr(err, "IPP decode")

	if goipp.Status(rsp.Code) != goipp.StatusOk {
		err = errors.New(goipp.Status(rsp.Code).String())
		checkErr(err, "IPP")
	}
}
