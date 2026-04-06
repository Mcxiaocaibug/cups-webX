package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"cups-web/internal/auth"
	"cups-web/internal/ipp"
	"cups-web/internal/store"
)

type printResp struct {
	JobID      string `json:"jobId,omitempty"`
	OK         bool   `json:"ok"`
	Pages      int    `json:"pages"`
	IsDuplex   bool   `json:"isDuplex"`
	DuplexMode string `json:"duplexMode"`
	IsColor    bool   `json:"isColor"`
	Copies     int    `json:"copies"`
}

var sendPrintJob = ipp.SendPrintJob
var cancelPrintJob = ipp.CancelJob

func printHandler(w http.ResponseWriter, r *http.Request) {
	// Expect multipart form
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, fh, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	printer := r.FormValue("printer")
	if printer == "" {
		writeJSONError(w, http.StatusBadRequest, "missing printer field")
		return
	}

	duplexMode := normalizeDuplexMode(r.FormValue("duplex"))
	isDuplex := duplexMode != "one-sided"
	isColor := r.FormValue("color") == "true"

	// Extended print options
	copiesStr := r.FormValue("copies")
	copies := 1
	if copiesStr != "" {
		if n, err := strconv.Atoi(copiesStr); err == nil && n > 0 {
			copies = n
		}
	}
	orientation := r.FormValue("orientation")
	paperSize := r.FormValue("paper_size")
	paperType := r.FormValue("paper_type")
	printScaling := r.FormValue("print_scaling")
	pageRange := r.FormValue("page_range")
	mirror := r.FormValue("mirror") == "true"

	storedRel, storedAbs, err := saveUploadedFile(file, fh.Filename, uploadDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	countCtx, cancel := convertTimeoutContext(r.Context())
	defer cancel()
	printPath := storedAbs
	var printCleanup func()
	printMime := ""
	var pages int
	kind := detectFileKind(storedAbs, fh.Filename)
	switch kind {
	case fileKindPDF:
		var err error
		pages, err = countPDFPages(storedAbs)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "failed to read pages")
			return
		}
		printMime = "application/pdf"
	case fileKindOffice:
		outPath, cleanup, err := convertOfficeToPDF(countCtx, storedAbs)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "conversion failed")
			return
		}
		pages, err = countPDFPages(outPath)
		if err != nil {
			cleanup()
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "failed to read pages")
			return
		}
		_, convertedAbs, err := saveConvertedPDFToUploads(outPath, storedRel, uploadDir)
		if err != nil {
			cleanup()
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusInternalServerError, "failed to save converted file")
			return
		}
		printPath = convertedAbs
		printCleanup = cleanup
		printMime = "application/pdf"
	case fileKindImage:
		outPath, cleanup, err := convertImageToPDF(storedAbs)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "conversion failed")
			return
		}
		_, convertedAbs, err := saveConvertedPDFToUploads(outPath, storedRel, uploadDir)
		if err != nil {
			cleanup()
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusInternalServerError, "failed to save converted file")
			return
		}
		printPath = convertedAbs
		printCleanup = cleanup
		printMime = "application/pdf"
		pages = 1
	case fileKindText:
		var err error
		pages, err = estimateTextPages(storedAbs)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "failed to read pages")
			return
		}
		outPath, cleanup, err := convertTextToPDF(storedAbs)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "conversion failed")
			return
		}
		_, convertedAbs, err := saveConvertedPDFToUploads(outPath, storedRel, uploadDir)
		if err != nil {
			cleanup()
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusInternalServerError, "failed to save converted file")
			return
		}
		printPath = convertedAbs
		printCleanup = cleanup
		printMime = "application/pdf"
	default:
		var err error
		pages, _, err = countPages(countCtx, storedAbs, fh.Filename)
		if err != nil {
			_ = os.Remove(storedAbs)
			writeJSONError(w, http.StatusBadRequest, "failed to read pages")
			return
		}
	}
	if pages < 1 {
		pages = 1
	}
	if printCleanup != nil {
		defer printCleanup()
	}

	sess, _ := auth.GetSession(r)
	var recordID int64

	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		user, err := store.GetUserByID(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}

		rec := store.PrintRecord{
			UserID:       user.ID,
			PrinterURI:   printer,
			Filename:     fh.Filename,
			StoredPath:   storedRel,
			Pages:        pages,
			Status:       "queued",
			StatusDetail: "",
			IsDuplex:     isDuplex,
			DuplexMode:   duplexMode,
			IsColor:      isColor,
			Copies:       copies,
			Orientation:  orientation,
			PaperSize:    paperSize,
			PaperType:    paperType,
			PrintScaling: printScaling,
			PageRange:    pageRange,
			Mirror:       mirror,
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		}
		id, err := store.InsertPrintRecord(r.Context(), tx, &rec)
		if err != nil {
			return err
		}
		recordID = id
		return nil
	})
	if err != nil {
		_ = os.Remove(storedAbs)
		writeJSONError(w, http.StatusInternalServerError, "failed to create print record")
		return
	}

	markRecordFailed := func(detail string) {
		_ = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
			return store.UpdatePrintStatus(r.Context(), tx, recordID, "failed", "", detail)
		})
	}

	f, err := os.Open(printPath)
	if err != nil {
		markRecordFailed("无法打开待打印文件")
		writeJSONError(w, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer f.Close()

	mime := printMime
	if mime == "" {
		mime = fh.Header.Get("Content-Type")
	}
	if mime == "" {
		buf := make([]byte, 512)
		if n, _ := f.Read(buf); n > 0 {
			mime = http.DetectContentType(buf[:n])
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				markRecordFailed("无法重新读取待打印文件")
				writeJSONError(w, http.StatusInternalServerError, "failed to read file")
				return
			}
		}
	}

	printOpts := ipp.PrintJobOptions{
		DuplexMode:   duplexMode,
		IsColor:      isColor,
		Copies:       copies,
		Orientation:  orientation,
		PaperSize:    paperSize,
		PaperType:    paperType,
		PrintScaling: printScaling,
		PageRange:    pageRange,
		Mirror:       mirror,
	}

	job, err := sendPrintJob(printer, f, mime, sess.Username, fh.Filename, printOpts)
	if err != nil {
		markRecordFailed(err.Error())
		writeJSONError(w, http.StatusInternalServerError, "print error: "+err.Error())
		return
	}

	_ = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		return store.UpdatePrintStatus(r.Context(), tx, recordID, "queued", job, "")
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(printResp{
		JobID:      job,
		OK:         true,
		Pages:      pages,
		IsDuplex:   isDuplex,
		DuplexMode: duplexMode,
		IsColor:    isColor,
		Copies:     copies,
	})
}

func normalizeDuplexMode(value string) string {
	switch value {
	case "", "false", "one-sided":
		return "one-sided"
	case "true", "two-sided-long-edge":
		return "two-sided-long-edge"
	case "two-sided-short-edge":
		return "two-sided-short-edge"
	default:
		return "one-sided"
	}
}
