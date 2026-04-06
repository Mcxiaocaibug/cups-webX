package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type PrintRecord struct {
	ID           int64
	UserID       int64
	Username     string
	PrinterURI   string
	Filename     string
	StoredPath   string
	Pages        int
	JobID        sql.NullString
	Status       string
	StatusDetail string
	IsDuplex     bool
	DuplexMode   string
	IsColor      bool
	Copies       int
	Orientation  string
	PaperSize    string
	PaperType    string
	PrintScaling string
	PageRange    string
	Mirror       bool
	CreatedAt    string
	UpdatedAt    string
}

type PrintFilter struct {
	Username string
	StartAt  string
	EndAt    string
	Limit    int
}

func InsertPrintRecord(ctx context.Context, tx *sql.Tx, rec *PrintRecord) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO print_jobs (
		user_id, printer_uri, filename, stored_path, pages,
		job_id, status, status_detail, is_duplex, duplex_mode, is_color, copies, orientation, paper_size, paper_type, print_scaling, page_range, mirror, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.UserID, rec.PrinterURI, rec.Filename, rec.StoredPath, rec.Pages,
		rec.JobID, rec.Status, rec.StatusDetail, rec.IsDuplex, rec.DuplexMode, rec.IsColor, rec.Copies, rec.Orientation, rec.PaperSize, rec.PaperType, rec.PrintScaling, rec.PageRange, rec.Mirror, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdatePrintStatus(ctx context.Context, tx *sql.Tx, id int64, status string, jobID string, detail string) error {
	_, err := tx.ExecContext(ctx, "UPDATE print_jobs SET status = ?, job_id = ?, status_detail = ?, updated_at = ? WHERE id = ?", status, jobID, detail, nowUTC(), id)
	return err
}

func GetPrintRecordByID(ctx context.Context, tx *sql.Tx, id int64) (PrintRecord, error) {
	row := tx.QueryRowContext(ctx, `SELECT
		p.id, p.user_id, u.username, p.printer_uri, p.filename, p.stored_path, p.pages,
		p.job_id, p.status, p.status_detail, p.is_duplex, p.duplex_mode, p.is_color, p.copies, p.orientation, p.paper_size, p.paper_type, p.print_scaling, p.page_range, p.mirror, p.created_at, p.updated_at
		FROM print_jobs p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = ?`, id)
	var rec PrintRecord
	err := row.Scan(
		&rec.ID, &rec.UserID, &rec.Username, &rec.PrinterURI, &rec.Filename, &rec.StoredPath,
		&rec.Pages, &rec.JobID, &rec.Status, &rec.StatusDetail, &rec.IsDuplex, &rec.DuplexMode, &rec.IsColor, &rec.Copies, &rec.Orientation, &rec.PaperSize, &rec.PaperType, &rec.PrintScaling, &rec.PageRange, &rec.Mirror, &rec.CreatedAt, &rec.UpdatedAt,
	)
	return rec, err
}

func ListPrintRecords(ctx context.Context, tx *sql.Tx, filter PrintFilter) ([]PrintRecord, error) {
	args := []interface{}{}
	conds := []string{"1=1"}
	if filter.Username != "" {
		conds = append(conds, "u.username = ?")
		args = append(args, filter.Username)
	}
	if filter.StartAt != "" {
		conds = append(conds, "p.created_at >= ?")
		args = append(args, filter.StartAt)
	}
	if filter.EndAt != "" {
		conds = append(conds, "p.created_at <= ?")
		args = append(args, filter.EndAt)
	}
	query := fmt.Sprintf(`SELECT
		p.id, p.user_id, u.username, p.printer_uri, p.filename, p.stored_path, p.pages,
		p.job_id, p.status, p.status_detail, p.is_duplex, p.duplex_mode, p.is_color, p.copies, p.orientation, p.paper_size, p.paper_type, p.print_scaling, p.page_range, p.mirror, p.created_at, p.updated_at
		FROM print_jobs p
		JOIN users u ON u.id = p.user_id
		WHERE %s
		ORDER BY p.created_at DESC`, strings.Join(conds, " AND "))
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []PrintRecord
	for rows.Next() {
		var rec PrintRecord
		if err := rows.Scan(
			&rec.ID, &rec.UserID, &rec.Username, &rec.PrinterURI, &rec.Filename, &rec.StoredPath,
			&rec.Pages, &rec.JobID, &rec.Status, &rec.StatusDetail, &rec.IsDuplex, &rec.DuplexMode, &rec.IsColor, &rec.Copies, &rec.Orientation, &rec.PaperSize, &rec.PaperType, &rec.PrintScaling, &rec.PageRange, &rec.Mirror, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
