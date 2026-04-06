package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestInsertPrintRecordPersistsAllOptions(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "prints.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var user User
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		created, err := CreateUser(ctx, tx, CreateUserInput{
			Username:     "alice",
			PasswordHash: "hash",
			Role:         RoleUser,
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

	createdAt := time.Now().UTC().Format(time.RFC3339)
	record := PrintRecord{
		UserID:       user.ID,
		PrinterURI:   "http://localhost:631/printers/test",
		Filename:     "example.docx",
		StoredPath:   "records/example.docx",
		Pages:        4,
		JobID:        sql.NullString{String: "321", Valid: true},
		Status:       "queued",
		StatusDetail: "任务已进入打印队列",
		IsDuplex:     true,
		DuplexMode:   "two-sided-short-edge",
		IsColor:      true,
		Copies:       3,
		Orientation:  "landscape",
		PaperSize:    "A3",
		PaperType:    "glossy",
		PrintScaling: "fill",
		PageRange:    "1-2 4",
		Mirror:       true,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}

	var recordID int64
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		id, err := InsertPrintRecord(ctx, tx, &record)
		if err != nil {
			return err
		}
		recordID = id
		return nil
	})
	if err != nil {
		t.Fatalf("InsertPrintRecord: %v", err)
	}

	err = s.WithTx(ctx, true, func(tx *sql.Tx) error {
		stored, err := GetPrintRecordByID(ctx, tx, recordID)
		if err != nil {
			return err
		}
		if stored.Status != "queued" {
			t.Fatalf("expected queued status, got %q", stored.Status)
		}
		if stored.JobID.String != "321" {
			t.Fatalf("expected job id 321, got %q", stored.JobID.String)
		}
		if stored.StatusDetail != "任务已进入打印队列" {
			t.Fatalf("expected status detail to persist, got %q", stored.StatusDetail)
		}
		if stored.UpdatedAt != createdAt {
			t.Fatalf("expected updated_at to persist, got %q", stored.UpdatedAt)
		}
		if stored.DuplexMode != "two-sided-short-edge" || !stored.IsColor || stored.Copies != 3 {
			t.Fatalf("unexpected print options: %+v", stored)
		}
		if stored.Orientation != "landscape" || stored.PaperSize != "A3" || stored.PaperType != "glossy" {
			t.Fatalf("unexpected paper options: %+v", stored)
		}
		if stored.PrintScaling != "fill" || stored.PageRange != "1-2 4" || !stored.Mirror {
			t.Fatalf("unexpected finishing options: %+v", stored)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GetPrintRecordByID: %v", err)
	}
}
