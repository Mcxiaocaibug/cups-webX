package ipp

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	goipp "github.com/OpenPrinting/goipp"
)

// PrintJobOptions holds all optional parameters for a print job.
type PrintJobOptions struct {
	DuplexMode   string // "one-sided" | "two-sided-long-edge" | "two-sided-short-edge"
	IsColor      bool   // true = color, false = monochrome
	Copies       int    // number of copies, 0 or 1 means 1
	Orientation  string // "portrait" | "landscape"
	PaperSize    string // "A4" | "A3" | "5inch" | "6inch" | "7inch" | "8inch" | "10inch"
	PaperType    string // "plain" | "photo" | "glossy" | "matte" | "envelope" | "cardstock" | "labels" | "auto"
	PrintScaling string // "auto" | "auto-fit" | "fit" | "fill" | "none"
	PageRange    string // e.g. "1-5 8 10-12"
	Mirror       bool   // mirror / horizontal flip
}

// JobStatus describes the latest observable state of an IPP print job.
type JobStatus struct {
	Status string
	Detail string
}

// SendPrintJob sends data to the printer via IPP using goipp to build the
// IPP Print-Job request. It returns a human-readable status or job identifier
// when available.
func SendPrintJob(printerURI string, r io.Reader, mime string, username string, jobName string, opts PrintJobOptions) (string, error) {
	// IPP attribute must use ipp:// scheme; HTTP transport uses http://
	ippURI := httpToIppURI(printerURI)

	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpPrintJob, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String("en-US")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(ippURI)))
	if username != "" {
		req.Operation.Add(goipp.MakeAttribute("requesting-user-name", goipp.TagName, goipp.String(username)))
	}
	if jobName != "" {
		req.Operation.Add(goipp.MakeAttribute("job-name", goipp.TagName, goipp.String(jobName)))
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	req.Operation.Add(goipp.MakeAttribute("document-format", goipp.TagMimeType, goipp.String(mime)))

	req.Job.Add(goipp.MakeAttribute("sides", goipp.TagKeyword, goipp.String(sidesKeywordForDuplexMode(opts.DuplexMode))))

	// Color mode
	if opts.IsColor {
		req.Job.Add(goipp.MakeAttribute("print-color-mode", goipp.TagKeyword, goipp.String("color")))
	} else {
		req.Job.Add(goipp.MakeAttribute("print-color-mode", goipp.TagKeyword, goipp.String("monochrome")))
	}

	// Copies
	copies := opts.Copies
	if copies < 1 {
		copies = 1
	}
	req.Job.Add(goipp.MakeAttribute("copies", goipp.TagInteger, goipp.Integer(copies)))

	// Orientation
	switch opts.Orientation {
	case "landscape":
		req.Job.Add(goipp.MakeAttribute("orientation-requested", goipp.TagEnum, goipp.Integer(4)))
	default:
		req.Job.Add(goipp.MakeAttribute("orientation-requested", goipp.TagEnum, goipp.Integer(3)))
	}

	// Paper size (media)
	if opts.PaperSize != "" {
		mediaName := paperSizeToIPP(opts.PaperSize)
		if mediaName != "" {
			req.Job.Add(goipp.MakeAttribute("media", goipp.TagKeyword, goipp.String(mediaName)))
		}
	}

	// Paper type (media-type)
	if opts.PaperType != "" && opts.PaperType != "auto" {
		mediaType := paperTypeToIPP(opts.PaperType)
		if mediaType != "" {
			req.Job.Add(goipp.MakeAttribute("media-type", goipp.TagKeyword, goipp.String(mediaType)))
		}
	}

	// Print scaling
	if opts.PrintScaling != "" {
		req.Job.Add(goipp.MakeAttribute("print-scaling", goipp.TagKeyword, goipp.String(opts.PrintScaling)))
	}

	// Page range
	if opts.PageRange != "" {
		ranges := parsePageRange(opts.PageRange)
		if len(ranges) > 0 {
			var vals goipp.Values
			for _, rng := range ranges {
				vals.Add(goipp.TagRange, goipp.Range{Lower: rng[0], Upper: rng[1]})
			}
			req.Job.Add(goipp.Attribute{Name: "page-ranges", Values: vals})
		}
	}

	// Mirror (best-effort)
	if opts.Mirror {
		req.Job.Add(goipp.MakeAttribute("mirror", goipp.TagBoolean, goipp.Boolean(true)))
	}

	payload, err := req.EncodeBytes()
	if err != nil {
		return "", fmt.Errorf("encode ipp request: %w", err)
	}

	body := io.MultiReader(bytes.NewBuffer(payload), r)

	httpReq, err := http.NewRequest(http.MethodPost, printerURI, body)
	if err != nil {
		return "", fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", goipp.ContentType)
	httpReq.Header.Set("Accept", goipp.ContentType)

	resp, err := http.DefaultClient.Do(httpReq)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("http status: %s", resp.Status)
	}

	var rsp goipp.Message
	if err := rsp.Decode(resp.Body); err != nil {
		return "", fmt.Errorf("decode ipp response: %w", err)
	}
	if goipp.Status(rsp.Code) != goipp.StatusOk {
		return "", fmt.Errorf("ipp error: %s", goipp.Status(rsp.Code).String())
	}

	for _, a := range rsp.Job {
		if a.Name == "job-uri" || a.Name == "job-id" {
			if len(a.Values) > 0 {
				return a.Values[0].V.String(), nil
			}
		}
	}

	return "ok", nil
}

// CancelJob cancels an active print job by job URI or numeric job ID.
func CancelJob(printerURI string, job string, username string) error {
	ippURI := httpToIppURI(printerURI)

	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpCancelJob, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String("en-US")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(ippURI)))
	if username != "" {
		req.Operation.Add(goipp.MakeAttribute("requesting-user-name", goipp.TagName, goipp.String(username)))
	}

	jobID, jobURI := parseJobIdentifier(job)
	switch {
	case jobURI != "":
		req.Operation.Add(goipp.MakeAttribute("job-uri", goipp.TagURI, goipp.String(jobURI)))
	case jobID > 0:
		req.Operation.Add(goipp.MakeAttribute("job-id", goipp.TagInteger, goipp.Integer(jobID)))
	default:
		return fmt.Errorf("unsupported job identifier: %q", job)
	}

	payload, err := req.EncodeBytes()
	if err != nil {
		return fmt.Errorf("encode ipp request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, printerURI, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", goipp.ContentType)
	httpReq.Header.Set("Accept", goipp.ContentType)

	resp, err := http.DefaultClient.Do(httpReq)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("http status: %s", resp.Status)
	}

	var rsp goipp.Message
	if err := rsp.Decode(resp.Body); err != nil {
		return fmt.Errorf("decode ipp response: %w", err)
	}
	if goipp.Status(rsp.Code) != goipp.StatusOk {
		return fmt.Errorf("ipp error: %s", goipp.Status(rsp.Code).String())
	}

	return nil
}

// GetJobStatus queries a job via IPP Get-Job-Attributes and maps the printer
// state into the application's record status lifecycle.
func GetJobStatus(printerURI string, job string) (JobStatus, error) {
	ippURI := httpToIppURI(printerURI)

	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpGetJobAttributes, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String("en-US")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(ippURI)))
	req.Operation.Add(goipp.MakeAttribute("requested-attributes", goipp.TagKeyword, goipp.String("all")))

	jobID, jobURI := parseJobIdentifier(job)
	switch {
	case jobURI != "":
		req.Operation.Add(goipp.MakeAttribute("job-uri", goipp.TagURI, goipp.String(jobURI)))
	case jobID > 0:
		req.Operation.Add(goipp.MakeAttribute("job-id", goipp.TagInteger, goipp.Integer(jobID)))
	default:
		return JobStatus{}, fmt.Errorf("unsupported job identifier: %q", job)
	}

	payload, err := req.EncodeBytes()
	if err != nil {
		return JobStatus{}, fmt.Errorf("encode ipp request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, printerURI, bytes.NewReader(payload))
	if err != nil {
		return JobStatus{}, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", goipp.ContentType)
	httpReq.Header.Set("Accept", goipp.ContentType)

	resp, err := http.DefaultClient.Do(httpReq)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return JobStatus{}, fmt.Errorf("http post: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return JobStatus{}, fmt.Errorf("http status: %s", resp.Status)
	}

	var rsp goipp.Message
	if err := rsp.Decode(resp.Body); err != nil {
		return JobStatus{}, fmt.Errorf("decode ipp response: %w", err)
	}
	if goipp.Status(rsp.Code) != goipp.StatusOk {
		return JobStatus{}, fmt.Errorf("ipp error: %s", goipp.Status(rsp.Code).String())
	}

	var stateRaw string
	var stateMessage string
	var stateReasons []string
	for _, a := range rsp.Job {
		if len(a.Values) == 0 {
			continue
		}
		switch a.Name {
		case "job-state":
			stateRaw = a.Values[0].V.String()
		case "job-state-message":
			stateMessage = a.Values[0].V.String()
		case "job-state-reasons":
			for _, v := range a.Values {
				stateReasons = append(stateReasons, v.V.String())
			}
		}
	}

	status := recordStatusForJobState(stateRaw)
	if status == "" {
		return JobStatus{}, fmt.Errorf("unsupported job state: %q", stateRaw)
	}
	return JobStatus{
		Status: status,
		Detail: statusDetailForJobState(status, stateMessage, stateReasons),
	}, nil
}

// paperSizeToIPP converts a paper size name to an IPP media keyword.
func paperSizeToIPP(size string) string {
	m := map[string]string{
		"A4":     "iso_a4_210x297mm",
		"A3":     "iso_a3_297x420mm",
		"A2":     "iso_a2_420x594mm",
		"A1":     "iso_a1_594x841mm",
		"5inch":  "oe_photo-5x7_5x7in",
		"6inch":  "oe_photo-l_3.5x5in",
		"7inch":  "oe_photo-7x5_7x5in",
		"8inch":  "oe_photo-8x10_8x10in",
		"10inch": "oe_photo-10x12_10x12in",
		"Letter": "na_letter_8.5x11in",
		"Legal":  "na_legal_8.5x14in",
	}
	return m[size]
}

// paperTypeToIPP converts a paper type name to an IPP media-type keyword.
func paperTypeToIPP(t string) string {
	m := map[string]string{
		"plain":     "stationery",
		"photo":     "photographic",
		"glossy":    "photographic-glossy",
		"matte":     "photographic-matte",
		"envelope":  "envelope",
		"cardstock": "cardstock",
		"labels":    "labels",
	}
	return m[t]
}

// parsePageRange parses a page range string like "1-5 8 10-12" into [][2]int32 pairs.
func parsePageRange(s string) [][2]int {
	var result [][2]int
	parts := strings.Fields(s)
	for _, p := range parts {
		if idx := strings.Index(p, "-"); idx > 0 {
			var lo, hi int
			fmt.Sscanf(p[:idx], "%d", &lo)
			fmt.Sscanf(p[idx+1:], "%d", &hi)
			if lo > 0 && hi >= lo {
				result = append(result, [2]int{lo, hi})
			}
		} else {
			var n int
			fmt.Sscanf(p, "%d", &n)
			if n > 0 {
				result = append(result, [2]int{n, n})
			}
		}
	}
	return result
}

func sidesKeywordForDuplexMode(mode string) string {
	switch mode {
	case "two-sided-long-edge", "two-sided-short-edge":
		return mode
	default:
		return "one-sided"
	}
}

func parseJobIdentifier(job string) (int, string) {
	job = strings.TrimSpace(job)
	if job == "" {
		return 0, ""
	}
	if n, err := strconv.Atoi(job); err == nil && n > 0 {
		return n, ""
	}
	if strings.Contains(job, "://") {
		if idx := strings.LastIndex(job, "/"); idx >= 0 && idx+1 < len(job) {
			if n, err := strconv.Atoi(job[idx+1:]); err == nil && n > 0 {
				return n, job
			}
		}
		return 0, job
	}
	return 0, ""
}

func recordStatusForJobState(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "3", "4", "pending", "pending-held":
		return "queued"
	case "5", "6", "processing", "processing-stopped":
		return "processing"
	case "7", "canceled", "cancelled":
		return "cancelled"
	case "8", "aborted":
		return "failed"
	case "9", "completed":
		return "printed"
	default:
		return ""
	}
}

func statusDetailForJobState(status string, message string, reasons []string) string {
	message = strings.TrimSpace(message)
	if message != "" && strings.ToLower(message) != "none" {
		return message
	}

	labels := make([]string, 0, len(reasons))
	seen := map[string]struct{}{}
	for _, reason := range reasons {
		reason = strings.TrimSpace(reason)
		if reason == "" || strings.EqualFold(reason, "none") {
			continue
		}
		label := translateJobStateReason(reason)
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	if len(labels) > 0 {
		return strings.Join(labels, " / ")
	}

	switch status {
	case "queued":
		return "任务已进入打印队列"
	case "processing":
		return "打印机正在处理该任务"
	case "failed":
		return "打印任务已中止"
	case "cancelled":
		return "打印任务已取消"
	default:
		return ""
	}
}

func translateJobStateReason(reason string) string {
	m := map[string]string{
		"job-incoming":                "数据接收中",
		"job-data-insufficient":       "打印数据不完整",
		"job-printing":                "正在打印",
		"processing-to-stop-point":    "正在处理停止指令",
		"job-completed-successfully":  "打印完成",
		"job-canceled-by-user":        "用户取消",
		"job-canceled-at-device":      "设备端取消",
		"job-aborted-by-system":       "系统中止",
		"unsupported-document-format": "不支持的文档格式",
		"document-format-error":       "文档格式错误",
		"resources-are-not-ready":     "打印资源未就绪",
		"printer-stopped":             "打印机已停止",
	}
	if label, ok := m[strings.ToLower(reason)]; ok {
		return label
	}
	return reason
}

// PrinterInfo holds information about a printer retrieved via IPP Get-Printer-Attributes.
type PrinterInfo struct {
	Name            string            `json:"name"`
	URI             string            `json:"uri"`
	State           string            `json:"state"`
	StateMessage    string            `json:"stateMessage"`
	StateReasons    []string          `json:"stateReasons"`
	QueuedJobs      int               `json:"queuedJobs"`
	FirmwareVersion string            `json:"firmwareVersion"`
	UptimeSeconds   int               `json:"uptimeSeconds"`
	MarkerNames     []string          `json:"markerNames"`
	MarkerTypes     []string          `json:"markerTypes"`
	MarkerLevels    []int             `json:"markerLevels"`
	MarkerColors    []string          `json:"markerColors"`
	MediaReady      []string          `json:"mediaReady"`
	Attributes      map[string]string `json:"attributes"`
}

// httpToIppURI converts an http:// URI to ipp:// for use in IPP request attributes.
// CUPS requires the printer-uri attribute to use the ipp:// scheme.
func httpToIppURI(uri string) string {
	if strings.HasPrefix(uri, "http://") {
		return "ipp://" + uri[len("http://"):]
	}
	if strings.HasPrefix(uri, "https://") {
		return "ipps://" + uri[len("https://"):]
	}
	return uri
}

// GetPrinterAttributes queries a printer via IPP Get-Printer-Attributes and returns structured info.
func GetPrinterAttributes(printerURI string) (*PrinterInfo, error) {
	log.Printf("[ipp] GetPrinterAttributes start, uri=%q", printerURI)

	// IPP attribute must use ipp:// scheme; HTTP transport uses http://
	ippURI := httpToIppURI(printerURI)
	log.Printf("[ipp] using ipp-scheme uri for attribute: %q", ippURI)

	req := goipp.NewRequest(goipp.DefaultVersion, goipp.OpGetPrinterAttributes, 1)
	req.Operation.Add(goipp.MakeAttribute("attributes-charset", goipp.TagCharset, goipp.String("utf-8")))
	req.Operation.Add(goipp.MakeAttribute("attributes-natural-language", goipp.TagLanguage, goipp.String("en-US")))
	req.Operation.Add(goipp.MakeAttribute("printer-uri", goipp.TagURI, goipp.String(ippURI)))
	req.Operation.Add(goipp.MakeAttribute("requested-attributes", goipp.TagKeyword, goipp.String("all")))

	payload, err := req.EncodeBytes()
	if err != nil {
		log.Printf("[ipp] encode error: %v", err)
		return nil, fmt.Errorf("encode ipp request: %w", err)
	}
	log.Printf("[ipp] encoded payload, %d bytes", len(payload))

	httpReq, err := http.NewRequest(http.MethodPost, printerURI, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[ipp] create http request error: %v", err)
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", goipp.ContentType)
	httpReq.Header.Set("Accept", goipp.ContentType)

	log.Printf("[ipp] sending HTTP POST to %q", printerURI)
	resp, err := http.DefaultClient.Do(httpReq)
	if resp != nil {
		defer resp.Body.Close()
		log.Printf("[ipp] HTTP response status: %s", resp.Status)
	}
	if err != nil {
		log.Printf("[ipp] http post error: %v", err)
		return nil, fmt.Errorf("http post: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		log.Printf("[ipp] non-2xx status: %s", resp.Status)
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}

	log.Printf("[ipp] decoding IPP response")
	var rsp goipp.Message
	if err := rsp.Decode(resp.Body); err != nil {
		log.Printf("[ipp] decode error: %v", err)
		return nil, fmt.Errorf("decode ipp response: %w", err)
	}
	log.Printf("[ipp] IPP response code: %d, printer attrs count: %d", rsp.Code, len(rsp.Printer))

	info := &PrinterInfo{
		URI:        printerURI,
		Attributes: make(map[string]string),
	}

	for _, a := range rsp.Printer {
		if len(a.Values) == 0 {
			continue
		}
		switch a.Name {
		case "printer-name":
			info.Name = a.Values[0].V.String()
			log.Printf("[ipp] printer-name=%q", info.Name)
		case "printer-state":
			raw := a.Values[0].V.String()
			log.Printf("[ipp] printer-state raw=%q (tag=%v)", raw, a.Values[0].T)
			switch raw {
			case "3":
				info.State = "idle"
			case "4":
				info.State = "processing"
			case "5":
				info.State = "stopped"
			default:
				info.State = raw
			}
		case "printer-state-message":
			info.StateMessage = a.Values[0].V.String()
		case "printer-state-reasons":
			for _, v := range a.Values {
				info.StateReasons = append(info.StateReasons, v.V.String())
			}
		case "queued-job-count":
			fmt.Sscanf(a.Values[0].V.String(), "%d", &info.QueuedJobs)
		case "printer-firmware-string-version":
			info.FirmwareVersion = a.Values[0].V.String()
		case "printer-up-time":
			fmt.Sscanf(a.Values[0].V.String(), "%d", &info.UptimeSeconds)
		case "marker-names":
			for _, v := range a.Values {
				info.MarkerNames = append(info.MarkerNames, v.V.String())
			}
		case "marker-types":
			for _, v := range a.Values {
				info.MarkerTypes = append(info.MarkerTypes, v.V.String())
			}
		case "marker-levels":
			for _, v := range a.Values {
				var lvl int
				fmt.Sscanf(v.V.String(), "%d", &lvl)
				info.MarkerLevels = append(info.MarkerLevels, lvl)
			}
		case "marker-colors":
			for _, v := range a.Values {
				info.MarkerColors = append(info.MarkerColors, v.V.String())
			}
		case "media-ready":
			for _, v := range a.Values {
				info.MediaReady = append(info.MediaReady, v.V.String())
			}
		default:
			vals := make([]string, 0, len(a.Values))
			for _, v := range a.Values {
				vals = append(vals, v.V.String())
			}
			info.Attributes[a.Name] = strings.Join(vals, ", ")
		}
	}

	log.Printf("[ipp] GetPrinterAttributes done: name=%q state=%q jobs=%d markers=%d",
		info.Name, info.State, info.QueuedJobs, len(info.MarkerNames))
	return info, nil
}

// Printer represents a CUPS printer.
type Printer struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// ListPrinters fetches the CUPS /printers HTML page on the given host and extracts printers.
func ListPrinters(host string) ([]Printer, error) {
	u := host
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %w", err)
	}

	hostOnly := parsed.Host
	if !strings.Contains(hostOnly, ":") {
		hostOnly = hostOnly + ":631"
	}

	listURL := (&url.URL{Scheme: "http", Host: hostOnly, Path: "/printers"}).String()

	resp, err := http.Get(listURL)
	if err != nil {
		return nil, fmt.Errorf("fetch printers page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read printers page: %w", err)
	}

	re := regexp.MustCompile(`(?i)<a[^>]+href=["']/printers/([^"'/>]+)["'][^>]*>([^<]+)</a>`)
	matches := re.FindAllSubmatch(body, -1)
	printers := make([]Printer, 0, len(matches))
	for _, m := range matches {
		name := string(m[1])
		display := string(m[2])
		uri := fmt.Sprintf("http://%s/printers/%s", hostOnly, name)
		printers = append(printers, Printer{Name: display, URI: uri})
	}

	return printers, nil
}
