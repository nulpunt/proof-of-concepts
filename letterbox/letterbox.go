package main

import (
	"github.com/GeertJohan/go.leptonica"
	"github.com/GeertJohan/go.tesseract"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// const imageName = "differentFonts.png"
const imageName = "pkiTaskforce.png"
const languageType = "nld" // eng

var pageHTML = `<!DOCTYPE html>
<html>
	<head>
		<title>Nulpunt proof-of-concept: letterbox</title>
		<style>
			html {
				-webkit-user-select: none;  /* Chrome all / Safari all */
				-moz-user-select: none;     /* Firefox all */
				-ms-user-select: none;      /* IE 10+ */

				/* No support for these yet, use at own risk */
				-o-user-select: none;
				user-select: none;     
			}

			.selectable, .selectable div {
				-webkit-user-select: all;
				-moz-user-select: all;
				-ms-user-select: all;
				-o-user-select: all;
				user-select: all;
			}

			#imageBase {
				position: relative;
			}
			.character {
				position: absolute;
				display: inline;
				color: red;
				font-weight: bold;
				background-color: rgba(255, 255, 255, 0.8);
			}
		</style>
	</head>
	<body>
		<div id="info" >
			Tesseract version: {{.TesseractVersion}} <br/>
			Displaying image: {{.ImageName}} <br/>
		</div>
		<div id="imageBase" >
			<img src="/files/{{.ImageName}}" />
			<div id="lines" class="selectable" >
				{{range .Lines}}
					<div class="line" >
						{{range .Characters}}
							<div class="character character-{{.Character}}" style="bottom: {{.StartY}}px; left: {{.StartX}}px;">{{.Character}}</div>
						{{end}}
					</div>
				{{end}}
			</div>
		</div>
	</body>
</html>
`

var pageTmpl *template.Template

func init() {
	var err error
	pageTmpl, err = template.New("page").Parse(pageHTML)
	if err != nil {
		log.Fatalf("Error parsing template for page: %s\n", err)
	}
}

type pageData struct {
	TesseractVersion string
	ImageName        string
	FullText         string
	Lines            []*pageDataLine
}

type pageDataLine struct {
	Characters []*pageDataCharacter
}

type pageDataCharacter struct {
	Character string
	StartX    uint32
	StartY    uint32
}

func main() {

	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("./files/"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// create new tess instance and point it to the tessdata location. Set language to english.
		t, err := tesseract.NewTess("/usr/local/share/tessdata", languageType)
		if err != nil {
			log.Printf("Error while initializing Tess: %s\n", err)
			return
		}
		defer t.Close()

		// open a new Pix from file with leptonica
		pix, err := leptonica.NewPixFromFile("./files/" + imageName)
		if err != nil {
			log.Printf("Error while getting pix from file: %s\n", err)
			return
		}

		// set the image to the tesseract instance
		t.SetImagePix(pix)

		// retrieve boxed text from the tesseract instance
		bt, err := t.BoxText(0)
		if err != nil {
			log.Printf("Error could not get boxtext from tesseract: %s\n", err)
			return
		}

		// get full text from tesseract
		fullText := t.Text()

		// get individual lines from full text
		lines := strings.Split(fullText, "\n")

		// create page data instance
		pd := &pageData{
			ImageName:        imageName,
			TesseractVersion: tesseract.Version(),
			FullText:         fullText,
			Lines:            make([]*pageDataLine, 0, len(lines)),
		}

		// keep a letter count
		letterCount := -1

		// range over text lines
		for _, line := range lines {
			// create new instance to keep line characters and metadata
			pdl := &pageDataLine{
				Characters: make([]*pageDataCharacter, 0),
			}

			// range over characters in line
			for _, c := range line {

				// check for space
				if c == ' ' {
					if len(pdl.Characters) > 0 {
						pdl.Characters[len(pdl.Characters)-1].Character += " "
					}
					continue
				}

				// increment letter count
				letterCount++

				// get boxText character
				btc := bt.Characters[letterCount]

				// compare to text character
				if c != btc.Character {
					log.Printf("Error. Character mismatch. Ommiting character. '%s' != '%s'\n", string(c), string(btc.Character))
					continue
				}

				// create new pdc and store it onto pdl
				pdc := &pageDataCharacter{
					Character: string(c),
					StartX:    btc.StartX,
					StartY:    btc.StartY,
				}
				pdl.Characters = append(pdl.Characters, pdc)

			}

			pd.Lines = append(pd.Lines, pdl)
		}

		// execute template with page data
		err = pageTmpl.Execute(w, pd)
		if err != nil {
			log.Printf("Could not execute template: %s\n", err)
		}
	})

	// listen and serve
	err := http.ListenAndServe(":1234", nil)
	if err != nil {
		log.Fatalf("Error serving HTTP: %s\n", err)
	}
}
