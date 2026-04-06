package main

import (
	"context"
	"database/sql"
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cups-web/internal/auth"
	"cups-web/internal/ipp"
	"cups-web/internal/store"

	"github.com/gorilla/mux"
)

type printRecordResponse struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"userId"`
	Username     string `json:"username"`
	PrinterURI   string `json:"printerUri"`
	Filename     string `json:"filename"`
	Pages        int    `json:"pages"`
	JobID        string `json:"jobId"`
	Status       string `json:"status"`
	StatusDetail string `json:"statusDetail"`
	IsDuplex     bool   `json:"isDuplex"`
	DuplexMode   string `json:"duplexMode"`
	IsColor      bool   `json:"isColor"`
	Copies       int    `json:"copies"`
	Orientation  string `json:"orientation"`
	PaperSize    string `json:"paperSize"`
	PaperType    string `json:"paperType"`
	PrintScaling string `json:"printScaling"`
	PageRange    string `json:"pageRange"`
	Mirror       bool   `json:"mirror"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

var lookupPrintJobStatus = ipp.GetJobStatus

func printRecordsHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	startAt, endAt, err := parseDateRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid date range")
		return
	}

	var records []store.PrintRecord
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		user, err := store.GetUserByID(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}
		rows, err := store.ListPrintRecords(r.Context(), tx, store.PrintFilter{
			Username: user.Username,
			StartAt:  startAt,
			EndAt:    endAt,
		})
		if err != nil {
			return err
		}
		records = rows
		return nil
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load records")
		return
	}
	writeJSON(w, mapPrintRecords(syncPrintRecordStatuses(r.Context(), records)))
}

func adminPrintRecordsHandler(w http.ResponseWriter, r *http.Request) {
	startAt, endAt, err := parseDateRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid date range")
		return
	}
	username := r.URL.Query().Get("username")

	var records []store.PrintRecord
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		rows, err := store.ListPrintRecords(r.Context(), tx, store.PrintFilter{
			Username: username,
			StartAt:  startAt,
			EndAt:    endAt,
		})
		if err != nil {
			return err
		}
		records = rows
		return nil
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load records")
		return
	}
	writeJSON(w, mapPrintRecords(syncPrintRecordStatuses(r.Context(), records)))
}

func printRecordFileHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid record id")
		return
	}

	var record store.PrintRecord
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		rec, err := store.GetPrintRecordByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		record = rec
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "record not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load record")
		return
	}
	if sess.Role != store.RoleAdmin && record.UserID != sess.UserID {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	absPath := filepath.Join(uploadDir, filepath.FromSlash(record.StoredPath))
	f, err := os.Open(absPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "file not found")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to stat file")
		return
	}

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": record.Filename})
	w.Header().Set("Content-Disposition", disposition)
	http.ServeContent(w, r, record.Filename, stat.ModTime(), f)
}

func retryPrintRecordHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	record, err := loadAuthorizedPrintRecord(r, sess)
	if err != nil {
		writePrintRecordError(w, err)
		return
	}

	_, printAbs, err := resolveStoredPrintPath(record.StoredPath, uploadDir)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "print file not found")
		return
	}

	f, err := os.Open(printAbs)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to open print file")
		return
	}
	defer f.Close()

	mimeType, err := detectStoredFileMIME(printAbs)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to inspect print file")
		return
	}
	if mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(filepath.Ext(printAbs))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var newRecordID int64
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		newRecord := clonePrintRecordForRetry(record)
		newRecord.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		newRecord.UpdatedAt = newRecord.CreatedAt
		newRecord.Status = "queued"
		newRecord.JobID = sql.NullString{}
		id, err := store.InsertPrintRecord(r.Context(), tx, &newRecord)
		if err != nil {
			return err
		}
		newRecordID = id
		return nil
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create retry record")
		return
	}

	jobID, err := sendPrintJob(record.PrinterURI, f, mimeType, record.Username, record.Filename, ipp.PrintJobOptions{
		DuplexMode:   record.DuplexMode,
		IsColor:      record.IsColor,
		Copies:       record.Copies,
		Orientation:  record.Orientation,
		PaperSize:    record.PaperSize,
		PaperType:    record.PaperType,
		PrintScaling: record.PrintScaling,
		PageRange:    record.PageRange,
		Mirror:       record.Mirror,
	})
	if err != nil {
		_ = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
			return store.UpdatePrintStatus(r.Context(), tx, newRecordID, "failed", "", err.Error())
		})
		writeJSONError(w, http.StatusInternalServerError, "retry print failed: "+err.Error())
		return
	}

	_ = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		return store.UpdatePrintStatus(r.Context(), tx, newRecordID, "queued", jobID, "")
	})

	writeJSON(w, printResp{
		OK:         true,
		JobID:      jobID,
		Pages:      record.Pages,
		IsDuplex:   record.IsDuplex,
		DuplexMode: record.DuplexMode,
		IsColor:    record.IsColor,
		Copies:     record.Copies,
	})
}

func cancelPrintRecordHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	record, err := loadAuthorizedPrintRecord(r, sess)
	if err != nil {
		writePrintRecordError(w, err)
		return
	}
	if record.Status != "queued" && record.Status != "processing" {
		writeJSONError(w, http.StatusConflict, "only active jobs can be cancelled")
		return
	}
	if !record.JobID.Valid || record.JobID.String == "" {
		writeJSONError(w, http.StatusConflict, "missing job identifier")
		return
	}

	if err := cancelPrintJob(record.PrinterURI, record.JobID.String, sess.Username); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "cancel failed: "+err.Error())
		return
	}

	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		return store.UpdatePrintStatus(r.Context(), tx, record.ID, "cancelled", record.JobID.String, "由用户手动取消")
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to update print record")
		return
	}

	writeJSON(w, map[string]interface{}{
		"ok":     true,
		"id":     record.ID,
		"status": "cancelled",
	})
}

func parseDateRange(r *http.Request) (string, string, error) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" && end == "" {
		return "", "", nil
	}
	var startAt string
	var endAt string
	if start != "" {
		t, err := time.ParseInLocation("2006-01-02", start, time.Local)
		if err != nil {
			return "", "", err
		}
		startAt = t.UTC().Format(time.RFC3339)
	}
	if end != "" {
		t, err := time.ParseInLocation("2006-01-02", end, time.Local)
		if err != nil {
			return "", "", err
		}
		t = t.AddDate(0, 0, 1).Add(-time.Second)
		endAt = t.UTC().Format(time.RFC3339)
	}
	return startAt, endAt, nil
}

func mapPrintRecords(records []store.PrintRecord) []printRecordResponse {
	resp := make([]printRecordResponse, 0, len(records))
	for _, rec := range records {
		jobID := ""
		if rec.JobID.Valid {
			jobID = rec.JobID.String
		}
		resp = append(resp, printRecordResponse{
			ID:           rec.ID,
			UserID:       rec.UserID,
			Username:     rec.Username,
			PrinterURI:   rec.PrinterURI,
			Filename:     rec.Filename,
			Pages:        rec.Pages,
			JobID:        jobID,
			Status:       rec.Status,
			StatusDetail: rec.StatusDetail,
			IsDuplex:     rec.IsDuplex,
			DuplexMode:   rec.DuplexMode,
			IsColor:      rec.IsColor,
			Copies:       rec.Copies,
			Orientation:  rec.Orientation,
			PaperSize:    rec.PaperSize,
			PaperType:    rec.PaperType,
			PrintScaling: rec.PrintScaling,
			PageRange:    rec.PageRange,
			Mirror:       rec.Mirror,
			CreatedAt:    rec.CreatedAt,
			UpdatedAt:    rec.UpdatedAt,
		})
	}
	return resp
}

func syncPrintRecordStatuses(ctx context.Context, records []store.PrintRecord) []store.PrintRecord {
	for i := range records {
		if !shouldSyncPrintRecord(records[i]) {
			continue
		}
		jobState, err := lookupPrintJobStatus(records[i].PrinterURI, records[i].JobID.String)
		if err != nil || jobState.Status == "" {
			continue
		}
		if jobState.Status == records[i].Status && jobState.Detail == records[i].StatusDetail {
			continue
		}
		if err := appStore.WithTx(ctx, false, func(tx *sql.Tx) error {
			return store.UpdatePrintStatus(ctx, tx, records[i].ID, jobState.Status, records[i].JobID.String, jobState.Detail)
		}); err != nil {
			continue
		}
		records[i].Status = jobState.Status
		records[i].StatusDetail = jobState.Detail
	}
	return records
}

func shouldSyncPrintRecord(record store.PrintRecord) bool {
	if !record.JobID.Valid || strings.TrimSpace(record.JobID.String) == "" {
		return false
	}
	switch record.Status {
	case "queued", "processing":
		return true
	default:
		return false
	}
}

func loadAuthorizedPrintRecord(r *http.Request, sess auth.Session) (store.PrintRecord, error) {
	id, err := parseIDParam(r)
	if err != nil {
		return store.PrintRecord{}, err
	}

	var record store.PrintRecord
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		rec, err := store.GetPrintRecordByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		record = rec
		return nil
	})
	if err != nil {
		return store.PrintRecord{}, err
	}
	if sess.Role != store.RoleAdmin && record.UserID != sess.UserID {
		return store.PrintRecord{}, errPrintRecordForbidden
	}
	return record, nil
}

func clonePrintRecordForRetry(record store.PrintRecord) store.PrintRecord {
	return store.PrintRecord{
		UserID:       record.UserID,
		Username:     record.Username,
		PrinterURI:   record.PrinterURI,
		Filename:     record.Filename,
		StoredPath:   record.StoredPath,
		Pages:        record.Pages,
		Status:       record.Status,
		StatusDetail: "",
		IsDuplex:     record.IsDuplex,
		DuplexMode:   record.DuplexMode,
		IsColor:      record.IsColor,
		Copies:       record.Copies,
		Orientation:  record.Orientation,
		PaperSize:    record.PaperSize,
		PaperType:    record.PaperType,
		PrintScaling: record.PrintScaling,
		PageRange:    record.PageRange,
		Mirror:       record.Mirror,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

var errPrintRecordForbidden = errors.New("forbidden")

func writePrintRecordError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "record not found")
	case errors.Is(err, errPrintRecordForbidden):
		writeJSONError(w, http.StatusForbidden, "forbidden")
	default:
		if _, ok := err.(*strconv.NumError); ok {
			writeJSONError(w, http.StatusBadRequest, "invalid record id")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load record")
	}
}
