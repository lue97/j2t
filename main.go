package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/valyala/fastjson"
)

const (
	formatList = "list"
	formatJson = "json"
	formatCsv  = "csv"

	headerField   = "field"
	headerType    = "type"
	headerContent = "content"
)

func main() {
	parser := argparse.NewParser("j2t", "Lists the fields in a JSON string")
	outputFile := parser.String("o", "output", &argparse.Options{Help: "Sets the output file. Reads from STDIN by default"})
	inputFile := parser.String("i", "input", &argparse.Options{Help: "Sets the input file. Writes to STDOUT by default"})
	format := parser.Selector("f", "format", []string{formatList, formatJson, formatCsv}, &argparse.Options{Help: "Output format", Default: "list"})
	prefix := parser.String("P", "prefix", &argparse.Options{Help: "Field prefix"})
	prettyPrint := parser.Flag("p", "pretty-print", &argparse.Options{Help: "Pretty print. Only applicable for `json` format"})
	requireHeader := parser.Flag("H", "headers", &argparse.Options{Help: "If headers should be printed. Only applicable for `list` and `csv` format"})
	merge := parser.Flag("m", "merge", &argparse.Options{Help: "Merges type and content for fields with multiple types. Only applicable for `list` and `csv` format"})
	categorizeNumeric := parser.Flag("n", "numeric", &argparse.Options{Help: "Categorize `number` into `number_int` and `number_float`"})
	if err := parser.Parse(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v", parser.Usage(err))
		os.Exit(1)
	}

	input, err := getReader(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", parser.Usage(err))
		os.Exit(1)
	}

	content, err := io.ReadAll(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	m := make(map[string]typeMap)
	b, err := fastjson.ParseBytes(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	Parse(*prefix, b, m, *categorizeNumeric)
	sortedKeys := make([]string, 0, len(m))
	for k := range m {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] < sortedKeys[j]
	})

	writer, closeWriter, err := getWriter(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", parser.Usage(err))
		os.Exit(1)
	}
	defer closeWriter()

	switch *format {
	case "list":
		err = writeList(writer, sortedKeys, m, *merge, *requireHeader)
	case "json":
		err = writeJson(writer, sortedKeys, m, *prettyPrint)
	case "csv":
		err = writeCsv(writer, sortedKeys, m, *merge, *requireHeader)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func getReader(inputFile *string) (io.Reader, error) {
	if inputFile != nil && *inputFile != "" {
		f, err := os.Open(*inputFile)
		if err != nil {
			return nil, err
		}
		return bufio.NewReader(f), nil
	}
	return bufio.NewReader(os.Stdin), nil
}

func getWriter(outputFile *string) (io.Writer, func() error, error) {
	if outputFile != nil && *outputFile != "" {
		f, err := os.OpenFile(*outputFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, err
		}
		return bufio.NewWriter(f), func() error {
			return f.Close()
		}, nil
	}
	return os.Stdout, func() error {
		return nil
	}, nil
}

func writeCsv(w io.Writer, sortedKeys []string, maps map[string]typeMap, merge, requireHeaders bool) error {
	writer := csv.NewWriter(w)
	if requireHeaders {
		if err := writer.Write([]string{headerField, headerType, headerContent}); err != nil {
			return err
		}
	}
	for _, field := range sortedKeys {
		m := maps[field]
		typeArr := make([]string, 0, len(m))
		contentArr := make([]string, 0, len(m))
		for t, c := range m {
			if !merge {
				if err := writer.Write([]string{field, t, c}); err != nil {
					return err
				}
			}
			typeArr = append(typeArr, t)
			contentArr = append(contentArr, c)
		}
		if !merge {
			continue
		}
		if err := writer.Write([]string{field, strings.Join(typeArr, ";"), strings.Join(contentArr, ";")}); err != nil {
			return err
		}
		continue
	}
	writer.Flush()
	return nil
}

func writeJson(w io.Writer, sortedKeys []string, maps map[string]typeMap, prettyPrint bool) error {
	content := make([]map[string]any, 0, len(maps))
	for _, field := range sortedKeys {
		typeContents := make([]map[string]string, 0, len(maps[field]))
		for t, c := range maps[field] {
			typeContents = append(typeContents, map[string]string{
				headerType:    t,
				headerContent: c,
			})
		}
		content = append(content, map[string]any{
			"types":     typeContents,
			headerField: field,
		})
	}
	enc := json.NewEncoder(w)
	if prettyPrint {
		enc.SetIndent("", "    ")
	}
	return enc.Encode(content)
}

func writeList(w io.Writer, sortedFields []string, maps map[string]typeMap, merge, requireHeaders bool) error {
	fieldLen := len(headerField)
	for _, field := range sortedFields {
		fieldLen = max(fieldLen, len(field))
	}
	fieldLen++
	typeLen := len(headerType)
	for _, fieldMeta := range maps {
		mergeLen := 0
		for t, _ := range fieldMeta {
			if merge {
				mergeLen += len(t) + 1
			} else {
				typeLen = max(typeLen, len(t))
			}
		}
		if merge {
			mergeLen--
			typeLen = max(mergeLen, typeLen)
		}
	}
	typeLen++
	if requireHeaders {
		if _, err := w.Write([]byte(fmt.Sprintf("%-*s%-*s%-s\n", fieldLen, headerField, typeLen, headerType, headerContent))); err != nil {
			return err
		}
	}
	for _, field := range sortedFields {
		m := maps[field]
		typeArr := make([]string, 0, len(m))
		contentArr := make([]string, 0, len(m))
		for t, c := range m {
			if !merge {
				if _, err := w.Write([]byte(fmt.Sprintf("%-*s%-*s%-s\n", fieldLen, field, typeLen, t, c))); err != nil {
					return err
				}
				continue
			}
			typeArr = append(typeArr, t)
			contentArr = append(contentArr, c)
		}
		if !merge {
			continue
		}
		if _, err := w.Write([]byte(fmt.Sprintf(
			"%-*s%-*s%-s\n",
			fieldLen,
			field,
			typeLen,
			strings.Join(typeArr, ";"),
			strings.Join(contentArr, ";"),
		))); err != nil {
			return err
		}
	}
	return nil
}
