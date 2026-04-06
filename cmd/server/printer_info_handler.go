package main

import (
	"log"
	"net/http"

	"cups-web/internal/ipp"
)

// printerInfoHandler handles GET /api/printer-info?uri=<printer_uri>
// It queries the printer via IPP Get-Printer-Attributes and returns structured info.
func printerInfoHandler(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	log.Printf("[printer-info] request received, uri=%q", uri)

	if uri == "" {
		log.Printf("[printer-info] error: missing uri parameter")
		writeJSONError(w, http.StatusBadRequest, "missing uri parameter")
		return
	}

	log.Printf("[printer-info] calling GetPrinterAttributes for uri=%q", uri)
	info, err := ipp.GetPrinterAttributes(uri)
	if err != nil {
		log.Printf("[printer-info] GetPrinterAttributes error: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to get printer info: "+err.Error())
		return
	}

	log.Printf("[printer-info] success: name=%q state=%q jobs=%d", info.Name, info.State, info.QueuedJobs)
	writeJSON(w, info)
}
