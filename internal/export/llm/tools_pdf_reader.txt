package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ledongthuc/pdf"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"log"
	"os"
	"sort"
	"strings"
)

type PdfReader struct {
	File   string
	Result []schema.Document
}

func (pdf *PdfReader) Execute(context context.Context) error {
	docs, err := fetchDocumentsFromPdf(pdf.File, context)
	if err != nil {
		return err
	}
	pdf.Result = docs
	return nil
}

func fetchDocumentsFromPdf(doc string, context context.Context) ([]schema.Document, error) {
	file, err := os.Open(doc)
	if err != nil {
		log.Default().Println(err)
		return nil, errors.New("error opening file")
	}

	fileStat, err := file.Stat()
	if err != nil {
		log.Default().Println(err)
		return nil, errors.New("error to retrieve file stats")
	}

	pdf := documentloaders.NewPDF(file, fileStat.Size())

	docs, err := pdf.Load(context)
	if err != nil {
		log.Default().Println(err)
		return nil, errors.New("error to load pdf file")
	}

	a, _ := ReadPdf(doc)
	docs = []schema.Document{
		schema.Document{
			PageContent: a,
		},
	}

	return docs, nil

}

func ReadPdf(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()
	fileStat, err := f.Stat()
	reader, err := pdf.NewReader(f, fileStat.Size())
	if err != nil {
		log.Fatalf("Error creating PDF reader: %v", err)
	}

	numPages := reader.NumPage()

	docs := []schema.Document{}

	// fonts to be used when getting plain text from pages
	fonts := make(map[string]*pdf.Font)
	for i := 1; i < numPages+1; i++ {
		p := reader.Page(i)
		// add fonts to map
		for _, name := range p.Fonts() {
			// only add the font if we don't already have it
			if _, ok := fonts[name]; !ok {
				f := p.Font(name)
				fonts[name] = &f
			}
		}
		text := ""
		//text, _ = GetPlainText(p, fonts)

		rows, _ := GetTextByRow(p)
		for _, row := range rows {
			for _, line := range row.Content {
				//if line.S == string([]byte{239, 191, 189}) {
				//	text += " | "
				//} else {
				//for i = int(line.X); i > 0; {
				//	text += ""
				//	i -= 10
				//}
				text += line.S
				//}

				if strings.HasSuffix(text, "协议表:") {
					fmt.Println()
				}
			}
			text = text + "\n"
		}
		fmt.Println(text)
		// add the document to the doc list
		docs = append(docs, schema.Document{
			PageContent: text,
			Metadata: map[string]any{
				"page":        i,
				"total_pages": numPages,
			},
		})
	}
	return "", nil
}

type nopEncoder struct {
}

func (e *nopEncoder) Decode(raw string) (text string) {
	return raw
}
func GetTextByRow(p pdf.Page) (pdf.Rows, error) {
	result := pdf.Rows{}
	var err error

	defer func() {
		if r := recover(); r != nil {
			result = pdf.Rows{}
			err = errors.New(fmt.Sprint(r))
		}
	}()

	showText := func(enc pdf.TextEncoding, currentX, currentY float64, s string) {
		var textBuilder bytes.Buffer
		for _, ch := range enc.Decode(s) {
			_, err := textBuilder.WriteRune(ch)
			if err != nil {
				panic(err)
			}
		}

		// if DebugOn {
		// 	fmt.Println(textBuilder.String())
		// }

		text := pdf.Text{
			S: textBuilder.String() + " ",
			X: currentX,
			Y: currentY,
		}

		var currentRow *pdf.Row
		rowFound := false
		for _, row := range result {
			if int64(currentY) == row.Position {
				currentRow = row
				rowFound = true
				break
			}
		}

		if !rowFound {
			currentRow = &pdf.Row{
				Position: int64(currentY),
				Content:  pdf.TextHorizontal{},
			}
			result = append(result, currentRow)
		}

		currentRow.Content = append(currentRow.Content, text)
	}

	walkTextBlocks(p, showText)

	for _, row := range result {
		sort.Sort(row.Content)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Position > result[j].Position
	})

	return result, err
}

func walkTextBlocks(p pdf.Page, walker func(enc pdf.TextEncoding, x, y float64, s string)) {
	strm := p.V.Key("Contents")

	fonts := make(map[string]*pdf.Font)
	for _, font := range p.Fonts() {
		f := p.Font(font)
		fonts[font] = &f
	}

	var enc pdf.TextEncoding = &nopEncoder{}
	var currentX, currentY float64
	pdf.Interpret(strm, func(stk *pdf.Stack, op string) {
		n := stk.Len()
		args := make([]pdf.Value, n)
		for i := n - 1; i >= 0; i-- {
			args[i] = stk.Pop()
		}

		if op != "cm" && op != "c" && op != "Tm" {
			//fmt.Println(op, "->", args)
		}
		switch op {
		default:
			return
		case "BT":
		case "T*": // move to start of next line
		case "Tf": // set text font and size
			if len(args) != 2 {
				panic("bad TL")
			}

			if font, ok := fonts[args[0].Name()]; ok {
				enc = font.Encoder()
			} else {
				enc = &nopEncoder{}
			}
		case "\"": // set spacing, move to next line, and show text
			if len(args) != 3 {
				panic("bad \" operator")
			}
			fallthrough
		case "'": // move to next line and show text
			if len(args) != 1 {
				panic("bad ' operator")
			}
			fallthrough
		case "Tj": // show text
			if len(args) != 1 {
				panic("bad Tj operator")
			}

			walker(enc, currentX, currentY, args[0].RawString())
		case "TJ": // show text, allowing individual glyph positioning
			v := args[0]
			for i := 0; i < v.Len(); i++ {
				x := v.Index(i)
				if x.Kind() == pdf.String {
					walker(enc, currentX, currentY, x.RawString())
				}
			}
		case "Td":
			walker(enc, currentX, currentY, "")
		case "Tm":
			currentX = args[4].Float64()
			currentY = args[5].Float64()
		}
	})
}
