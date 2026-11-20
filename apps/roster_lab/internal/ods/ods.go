package ods

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Sheet struct {
	Name string
	Rows [][]string
}

type Workbook struct {
	Sheets []Sheet
}

func Write(path string, workbook Workbook) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp := path + ".tmp"
	defer os.Remove(temp)
	file, err := os.Create(temp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(file)
	mimeType := []byte("application/vnd.oasis.opendocument.spreadsheet")
	if err := addStored(zw, "mimetype", mimeType); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	manifest := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:full-path="/" manifest:version="1.2" manifest:media-type="application/vnd.oasis.opendocument.spreadsheet"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="settings.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`
	if err := addDeflated(zw, "META-INF/manifest.xml", []byte(manifest)); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	meta := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">
  <office:meta><meta:generator>roster_lab</meta:generator><meta:creation-date>%s</meta:creation-date></office:meta>
</office:document-meta>`, time.Now().UTC().Format(time.RFC3339))
	if err := addDeflated(zw, "meta.xml", []byte(meta)); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	styles := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles office:version="1.2" xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0">
  <office:styles/>
  <office:automatic-styles/>
  <office:master-styles/>
</office:document-styles>`
	if err := addDeflated(zw, "styles.xml", []byte(styles)); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	settings := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-settings office:version="1.2" xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0">
  <office:settings/>
</office:document-settings>`
	if err := addDeflated(zw, "settings.xml", []byte(settings)); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	content := buildContent(workbook)
	if err := addDeflated(zw, "content.xml", []byte(content)); err != nil {
		_ = zw.Close()
		_ = file.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replaceFile(temp, path)
}

func Read(path string) (Workbook, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Workbook{}, err
	}
	defer reader.Close()
	var content io.ReadCloser
	for _, file := range reader.File {
		if file.Name == "content.xml" {
			content, err = file.Open()
			break
		}
	}
	if content == nil {
		if err != nil {
			return Workbook{}, err
		}
		return Workbook{}, fmt.Errorf("content.xml missing from %s", path)
	}
	defer content.Close()
	var document contentDocument
	if err := xml.NewDecoder(content).Decode(&document); err != nil {
		return Workbook{}, fmt.Errorf("parse ODS content.xml: %w", err)
	}
	workbook := Workbook{Sheets: make([]Sheet, 0, len(document.Body.Spreadsheet.Tables))}
	for _, table := range document.Body.Spreadsheet.Tables {
		sheet := Sheet{Name: table.Name}
		for _, row := range table.Rows {
			values := make([]string, 0)
			for _, cell := range row.Cells {
				repeat, _ := strconv.Atoi(cell.RepeatColumns)
				if repeat < 1 {
					repeat = 1
				}
				value := strings.Join(cell.Paragraphs, "\n")
				for i := 0; i < repeat; i++ {
					values = append(values, value)
				}
			}
			repeat, _ := strconv.Atoi(row.RepeatRows)
			if repeat < 1 {
				repeat = 1
			}
			for i := 0; i < repeat; i++ {
				sheet.Rows = append(sheet.Rows, append([]string(nil), values...))
			}
		}
		workbook.Sheets = append(workbook.Sheets, sheet)
	}
	return workbook, nil
}

func TechnicalValue(workbook Workbook, key string) (string, bool) {
	for _, sheet := range workbook.Sheets {
		if sheet.Name != "Technical" {
			continue
		}
		for _, row := range sheet.Rows {
			if len(row) >= 2 && row[0] == key {
				return row[1], true
			}
		}
	}
	return "", false
}

func EncodeJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func DecodeJSON(value string, target any) error {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func buildContent(workbook Workbook) string {
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString(`<office:document-content office:version="1.2" xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0">`)
	body.WriteString(`<office:automatic-styles><style:style style:name="grid-cell" style:family="table-cell"><style:table-cell-properties fo:border="0.75pt solid #808080" style:vertical-align="top"/></style:style></office:automatic-styles>`)
	body.WriteString(`<office:body><office:spreadsheet>`)
	for _, sheet := range workbook.Sheets {
		body.WriteString(`<table:table table:name="`)
		escape(&body, sheet.Name)
		body.WriteString(`">`)
		for _, row := range sheet.Rows {
			body.WriteString(`<table:table-row>`)
			for _, cell := range row {
				body.WriteString(`<table:table-cell table:style-name="grid-cell" office:value-type="string"><text:p>`)
				writeParagraph(&body, cell)
				body.WriteString(`</text:p></table:table-cell>`)
			}
			body.WriteString(`</table:table-row>`)
		}
		body.WriteString(`</table:table>`)
	}
	body.WriteString(`</office:spreadsheet></office:body></office:document-content>`)
	return body.String()
}

func writeParagraph(builder *strings.Builder, value string) {
	parts := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for i, part := range parts {
		if i > 0 {
			builder.WriteString(`<text:line-break/>`)
		}
		escape(builder, part)
	}
}

func escape(builder *strings.Builder, value string) {
	_ = xml.EscapeText(builder, []byte(value))
}

func addStored(zw *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{
		Name: name, Method: zip.Store,
		CRC32:              crc32.ChecksumIEEE(data),
		CompressedSize64:   uint64(len(data)),
		UncompressedSize64: uint64(len(data)),
		Flags:              0,
	}
	writer, err := zw.CreateRaw(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func addDeflated(zw *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func replaceFile(source, destination string) error {
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(source, destination)
}

type contentDocument struct {
	Body struct {
		Spreadsheet struct {
			Tables []contentTable `xml:"table"`
		} `xml:"spreadsheet"`
	} `xml:"body"`
}

type contentTable struct {
	Name string       `xml:"name,attr"`
	Rows []contentRow `xml:"table-row"`
}

type contentRow struct {
	RepeatRows string        `xml:"number-rows-repeated,attr"`
	Cells      []contentCell `xml:"table-cell"`
}

type contentCell struct {
	RepeatColumns string   `xml:"number-columns-repeated,attr"`
	Paragraphs    []string `xml:"p"`
}

func (c *contentCell) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "number-columns-repeated" {
			c.RepeatColumns = attr.Value
		}
	}
	var paragraph strings.Builder
	inParagraph := false
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "p" {
				if inParagraph {
					paragraph.WriteByte('\n')
				}
				inParagraph = true
			}
			if value.Name.Local == "line-break" && inParagraph {
				paragraph.WriteByte('\n')
			}
		case xml.CharData:
			if inParagraph {
				paragraph.Write(value)
			}
		case xml.EndElement:
			if value.Name.Local == "p" && inParagraph {
				inParagraph = false
				c.Paragraphs = append(c.Paragraphs, paragraph.String())
				paragraph.Reset()
			}
			if value.Name == start.Name {
				return nil
			}
		}
	}
}
