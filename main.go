package main

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/otiai10/gosseract/v2"
)

type OCRConfig struct {
	Language       string
	DPI            float64
	OutputFile     string
	PreserveLayout bool
}

// ExtractTextFromPDF extracts text from PDF files, including scanned PDFs using OCR
func ExtractTextFromPDF(pdfPath string, config OCRConfig) (string, error) {
	// Open the PDF document
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return "", fmt.Errorf("error opening PDF: %w", err)
	}
	defer doc.Close()

	numPages := doc.NumPage()
	fmt.Printf("Processing %d pages from %s\n", numPages, pdfPath)

	var fullText strings.Builder

	// Process each page
	for pageNum := 0; pageNum < numPages; pageNum++ {
		fmt.Printf("Processing page %d/%d...\n", pageNum+1, numPages)

		// First, try to extract text directly (for text-based PDFs)
		text, err := doc.Text(pageNum)
		if err != nil {
			return "", fmt.Errorf("error extracting text from page %d: %w", pageNum+1, err)
		}

		// If text extraction yields substantial text, use it
		cleanText := strings.TrimSpace(text)
		if len(cleanText) > 50 { // Threshold for "substantial" text
			fullText.WriteString(fmt.Sprintf("--- Page %d ---\n", pageNum+1))
			fullText.WriteString(cleanText)
			fullText.WriteString("\n\n")
		} else {
			// If no text or minimal text, perform OCR on the page image
			fmt.Printf("Page %d has minimal text, performing OCR...\n", pageNum+1)

			ocrText, err := ocrPage(doc, pageNum, config)
			if err != nil {
				log.Printf("Warning: OCR failed for page %d: %v\n", pageNum+1, err)
				continue
			}

			fullText.WriteString(fmt.Sprintf("--- Page %d (OCR) ---\n", pageNum+1))
			fullText.WriteString(ocrText)
			fullText.WriteString("\n\n")
		}
	}

	return fullText.String(), nil
}

// ocrPage performs OCR on a single PDF page
func ocrPage(doc *fitz.Document, pageNum int, config OCRConfig) (string, error) {
	// Render page as image
	img, err := doc.Image(pageNum)
	if err != nil {
		return "", fmt.Errorf("error rendering page image: %w", err)
	}

	// Save image temporarily
	tmpFile := fmt.Sprintf("/tmp/page_%d.png", pageNum)
	defer os.Remove(tmpFile)

	f, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %w", err)
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()
		return "", fmt.Errorf("error encoding image: %w", err)
	}
	f.Close()

	// Perform OCR using Tesseract
	client := gosseract.NewClient()
	defer client.Close()

	client.SetImage(tmpFile)
	client.SetLanguage(config.Language)

	if config.PreserveLayout {
		client.SetPageSegMode(gosseract.PSM_AUTO)
	}

	text, err := client.Text()
	if err != nil {
		return "", fmt.Errorf("error performing OCR: %w", err)
	}

	return text, nil
}

// ExtractImagesFromPDF extracts all images from a PDF
func ExtractImagesFromPDF(pdfPath, outputDir string) error {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return fmt.Errorf("error opening PDF: %w", err)
	}
	defer doc.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	numPages := doc.NumPage()
	imageCount := 0

	for pageNum := 0; pageNum < numPages; pageNum++ {
		img, err := doc.Image(pageNum)
		if err != nil {
			log.Printf("Warning: could not extract image from page %d: %v\n", pageNum+1, err)
			continue
		}

		filename := filepath.Join(outputDir, fmt.Sprintf("page_%d.jpg", pageNum+1))
		f, err := os.Create(filename)
		if err != nil {
			log.Printf("Warning: could not create file %s: %v\n", filename, err)
			continue
		}

		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
			f.Close()
			log.Printf("Warning: could not encode image: %v\n", err)
			continue
		}
		f.Close()

		imageCount++
		fmt.Printf("Extracted image from page %d to %s\n", pageNum+1, filename)
	}

	fmt.Printf("Total images extracted: %d\n", imageCount)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("PDF OCR Text Extraction Tool")
		fmt.Println("\nUsage:")
		fmt.Println("  pdf-ocr-tool <pdf-file> [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  -o <output-file>    Save extracted text to file")
		fmt.Println("  -lang <language>    OCR language (default: eng)")
		fmt.Println("  -layout             Preserve layout during OCR")
		fmt.Println("  -extract-images     Extract all images to a directory")
		fmt.Println("\nExamples:")
		fmt.Println("  pdf-ocr-tool document.pdf")
		fmt.Println("  pdf-ocr-tool scanned.pdf -o output.txt -lang eng")
		fmt.Println("  pdf-ocr-tool document.pdf -extract-images")
		os.Exit(1)
	}

	pdfPath := os.Args[1]

	// Check if file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		log.Fatalf("Error: File %s does not exist\n", pdfPath)
	}

	// Parse command line options
	config := OCRConfig{
		Language: "eng",
		DPI:      300,
	}

	extractImages := false

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-o":
			if i+1 < len(os.Args) {
				config.OutputFile = os.Args[i+1]
				i++
			}
		case "-lang":
			if i+1 < len(os.Args) {
				config.Language = os.Args[i+1]
				i++
			}
		case "-layout":
			config.PreserveLayout = true
		case "-extract-images":
			extractImages = true
		}
	}

	// Extract images if requested
	if extractImages {
		outputDir := strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath)) + "_images"
		fmt.Printf("Extracting images to: %s\n", outputDir)
		if err := ExtractImagesFromPDF(pdfPath, outputDir); err != nil {
			log.Fatalf("Error extracting images: %v\n", err)
		}
		return
	}

	// Extract text from PDF
	text, err := ExtractTextFromPDF(pdfPath, config)
	if err != nil {
		log.Fatalf("Error extracting text: %v\n", err)
	}

	// Output the result
	if config.OutputFile != "" {
		if err := os.WriteFile(config.OutputFile, []byte(text), 0644); err != nil {
			log.Fatalf("Error writing to file: %v\n", err)
		}
		fmt.Printf("Text extracted successfully and saved to: %s\n", config.OutputFile)
	} else {
		fmt.Println("\n=== Extracted Text ===\n")
		fmt.Println(text)
	}
}