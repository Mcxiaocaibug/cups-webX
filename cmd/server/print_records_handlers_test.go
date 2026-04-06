package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"cups-web/internal/auth"
	"cups-web/internal/ipp"
	"cups-web/internal/store"

	"github.com/gorilla/mux"
)

func TestPrintRecordsHandlerSyncsQueuedStatus(t *testing.T) {
	ctx, s, user := setupPrintRecordHandlerTest(t)

	recordID := createPrintRecordForTest(t, ctx, s, store.PrintRecord{
		UserID:      user.ID,
		PrinterURI:  "http://localhost:631/printers/demo",
		Filename:    "queue.pdf",
		StoredPath:  "records/queue.pdf",
		Pages:       2,
		JobID:       sql.NullString{String: "42", Valid: true},
		Status:      "queued",
		IsColor:     true,
		Copies:      1,
		Orientation: "portrait",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	})

	prevLookup := lookupPrintJobStatus
	lookupPrintJobStatus = func(printerURI string, job string) (ipp.JobStatus, error) {
		if printerURI != "http://localhost:631/printers/demo" || job != "42" {
			t.Fatalf("unexpected lookup args: %q %q", printerURI, job)
		}
		return ipp.JobStatus{Status: "processing", Detail: "打印机正在处理该任务"}, nil
	}
	defer func() { lookupPrintJobStatus = prevLookup }()

	req := authenticatedRequest(t, user, http.MethodGet, "/api/print-records")
	rr := httptest.NewRecorder()

	printRecordsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp []printRecordResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 || resp[0].Status != "processing" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp[0].StatusDetail != "打印机正在处理该任务" {
		t.Fatalf("unexpected status detail: %+v", resp[0])
	}

	assertPrintRecordStatus(t, ctx, s, recordID, "processing", "42", "打印机正在处理该任务")
}

func TestRetryPrintRecordHandlerCreatesQueuedCloneAndUsesConvertedPDF(t *testing.T) {
	ctx, s, user := setupPrintRecordHandlerTest(t)

	baseDir := t.TempDir()
	prevUploadDir := uploadDir
	uploadDir = baseDir
	defer func() { uploadDir = prevUploadDir }()

	storedRel := filepath.ToSlash(filepath.Join("records", "report.docx"))
	storedAbs := filepath.Join(baseDir, filepath.FromSlash(storedRel))
	if err := os.MkdirAll(filepath.Dir(storedAbs), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(storedAbs, []byte("original office content"), 0o644); err != nil {
		t.Fatalf("WriteFile original: %v", err)
	}
	convertedAbs := storedAbs + ".print.pdf"
	if err := os.WriteFile(convertedAbs, []byte("%PDF-1.4\nretry-check"), 0o644); err != nil {
		t.Fatalf("WriteFile converted: %v", err)
	}

	recordID := createPrintRecordForTest(t, ctx, s, store.PrintRecord{
		UserID:       user.ID,
		PrinterURI:   "http://localhost:631/printers/demo",
		Filename:     "report.docx",
		StoredPath:   storedRel,
		Pages:        6,
		JobID:        sql.NullString{String: "123", Valid: true},
		Status:       "printed",
		IsDuplex:     true,
		DuplexMode:   "two-sided-short-edge",
		IsColor:      true,
		Copies:       2,
		Orientation:  "landscape",
		PaperSize:    "A3",
		PaperType:    "glossy",
		PrintScaling: "fill",
		PageRange:    "1-3 5",
		Mirror:       true,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	})

	var captured struct {
		mime     string
		username string
		jobName  string
		body     string
		opts     ipp.PrintJobOptions
	}
	prevSend := sendPrintJob
	sendPrintJob = func(printerURI string, r io.Reader, mime string, username string, jobName string, opts ipp.PrintJobOptions) (string, error) {
		payload, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}
		captured.mime = mime
		captured.username = username
		captured.jobName = jobName
		captured.body = string(payload)
		captured.opts = opts
		return "job-retry-999", nil
	}
	defer func() { sendPrintJob = prevSend }()

	req := authenticatedRequest(t, user, http.MethodPost, "/api/print-records/"+strconv.FormatInt(recordID, 10)+"/retry")
	req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(recordID, 10)})
	rr := httptest.NewRecorder()

	retryPrintRecordHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if captured.mime != "application/pdf" {
		t.Fatalf("expected retry MIME application/pdf, got %q", captured.mime)
	}
	if !strings.HasPrefix(captured.body, "%PDF-1.4") {
		t.Fatalf("expected converted PDF payload, got %q", captured.body)
	}
	if captured.username != user.Username || captured.jobName != "report.docx" {
		t.Fatalf("unexpected retry metadata: %+v", captured)
	}
	if captured.opts.DuplexMode != "two-sided-short-edge" || captured.opts.Copies != 2 || !captured.opts.Mirror {
		t.Fatalf("retry did not preserve options: %+v", captured.opts)
	}

	records := listPrintRecordsForTest(t, ctx, s, user.Username)
	if len(records) != 2 {
		t.Fatalf("expected 2 records after retry, got %d", len(records))
	}
	var retried store.PrintRecord
	for _, rec := range records {
		if rec.ID != recordID && rec.JobID.Valid && rec.JobID.String == "job-retry-999" {
			retried = rec
			break
		}
	}
	if retried.ID == 0 {
		t.Fatalf("retry record not found: %+v", records)
	}
	if retried.Status != "queued" {
		t.Fatalf("expected queued retry record, got %q", retried.Status)
	}
	if retried.StoredPath != storedRel || retried.PaperSize != "A3" || retried.PageRange != "1-3 5" {
		t.Fatalf("unexpected retry record: %+v", retried)
	}
}

func TestCancelPrintRecordHandlerUpdatesStatus(t *testing.T) {
	ctx, s, user := setupPrintRecordHandlerTest(t)

	recordID := createPrintRecordForTest(t, ctx, s, store.PrintRecord{
		UserID:      user.ID,
		PrinterURI:  "http://localhost:631/printers/demo",
		Filename:    "cancel.pdf",
		StoredPath:  "records/cancel.pdf",
		Pages:       1,
		JobID:       sql.NullString{String: "job://123", Valid: true},
		Status:      "queued",
		IsColor:     false,
		Copies:      1,
		Orientation: "portrait",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	})

	var gotPrinter, gotJob, gotUser string
	prevCancel := cancelPrintJob
	cancelPrintJob = func(printerURI string, job string, username string) error {
		gotPrinter, gotJob, gotUser = printerURI, job, username
		return nil
	}
	defer func() { cancelPrintJob = prevCancel }()

	req := authenticatedRequest(t, user, http.MethodPost, "/api/print-records/"+strconv.FormatInt(recordID, 10)+"/cancel")
	req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(recordID, 10)})
	rr := httptest.NewRecorder()

	cancelPrintRecordHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if gotPrinter != "http://localhost:631/printers/demo" || gotJob != "job://123" || gotUser != user.Username {
		t.Fatalf("unexpected cancel args: %q %q %q", gotPrinter, gotJob, gotUser)
	}
	assertPrintRecordStatus(t, ctx, s, recordID, "cancelled", "job://123", "由用户手动取消")
}

func setupPrintRecordHandlerTest(t *testing.T) (context.Context, *store.Store, store.User) {
	t.Helper()

	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	prevStore := appStore
	appStore = s
	t.Cleanup(func() { appStore = prevStore })

	auth.SetupSecureCookie("", "")

	var user store.User
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		created, err := store.CreateUser(ctx, tx, store.CreateUserInput{
			Username:     "alice",
			PasswordHash: "hash",
			Role:         store.RoleUser,
		})
		if err != nil {
			return err
		}
		user = created
		return nil
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	return ctx, s, user
}

func createPrintRecordForTest(t *testing.T, ctx context.Context, s *store.Store, rec store.PrintRecord) int64 {
	t.Helper()

	if rec.CreatedAt == "" {
		rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if rec.UpdatedAt == "" {
		rec.UpdatedAt = rec.CreatedAt
	}

	var id int64
	err := s.WithTx(ctx, false, func(tx *sql.Tx) error {
		recordID, err := store.InsertPrintRecord(ctx, tx, &rec)
		if err != nil {
			return err
		}
		id = recordID
		return nil
	})
	if err != nil {
		t.Fatalf("InsertPrintRecord: %v", err)
	}
	return id
}

func authenticatedRequest(t *testing.T, user store.User, method string, target string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	cookieWriter := httptest.NewRecorder()
	if err := auth.SetSession(cookieWriter, auth.Session{UserID: user.ID, Username: user.Username, Role: user.Role}); err != nil {
		t.Fatalf("SetSession: %v", err)
	}
	for _, cookie := range cookieWriter.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func assertPrintRecordStatus(t *testing.T, ctx context.Context, s *store.Store, id int64, wantStatus string, wantJobID string, wantDetail string) {
	t.Helper()

	err := s.WithTx(ctx, true, func(tx *sql.Tx) error {
		record, err := store.GetPrintRecordByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if record.Status != wantStatus {
			t.Fatalf("expected status %q, got %q", wantStatus, record.Status)
		}
		gotJobID := ""
		if record.JobID.Valid {
			gotJobID = record.JobID.String
		}
		if gotJobID != wantJobID {
			t.Fatalf("expected job id %q, got %q", wantJobID, gotJobID)
		}
		if record.StatusDetail != wantDetail {
			t.Fatalf("expected status detail %q, got %q", wantDetail, record.StatusDetail)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GetPrintRecordByID: %v", err)
	}
}

func listPrintRecordsForTest(t *testing.T, ctx context.Context, s *store.Store, username string) []store.PrintRecord {
	t.Helper()

	var records []store.PrintRecord
	err := s.WithTx(ctx, true, func(tx *sql.Tx) error {
		rows, err := store.ListPrintRecords(ctx, tx, store.PrintFilter{Username: username})
		if err != nil {
			return err
		}
		records = rows
		return nil
	})
	if err != nil {
		t.Fatalf("ListPrintRecords: %v", err)
	}
	return records
}
