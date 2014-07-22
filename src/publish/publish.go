package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

/*
	html structure with specific format.
*/
type html struct {
	Head HtmlHead `xml:"head"`
	Body HtmlBody `xml:"body"`
}

type HtmlHead struct {
	Title string `xml:"title"`
	Stl   Style  `xml:"style"`
	Met   []Meta `xml:"meta"`
}

type Style struct {
	StyleBodies string `xml:",chardata"`
	Type        string `xml:"type,attr"`
}

type Meta struct {
	Http_equiv string `xml:"http-equiv,attr"`
	Content    string `xml:"content,attr"`
}

type HtmlBody struct {
	PLines []PLine `xml:"p"`
}

type PLine struct {
	// this unit is minimum of line. includes link, string, and other contents.
	Ln string `xml:",innerxml"`
	Cs string `xml:"class,attr"`
}

func main() {
	/*
		拡張子が.rtfdか.rtfのものがあれば、記事に変換する。
	*/
	targetExtensions := []string{".rtfd", ".rtf"}

	/*
		実行箇所のパスを取得。
	*/
	currentPath, _ := os.Getwd()

	fileInfos, _ := ioutil.ReadDir(currentPath)

	listUp := func() []string {
		filePaths := []string{}

		for _, fileInfo := range fileInfos {
			fileName := fileInfo.Name()
			fileExt := path.Ext(fileName)

			for _, targetExtension := range targetExtensions {
				if fileExt == targetExtension {
					filePaths = append(filePaths, fileName)
				}
			}
		}

		return filePaths
	}
	targetFilePaths := listUp()

	readHtmlFile := func(htmlFilePath string) string {
		input, _ := os.Open(htmlFilePath)

		var lines string
		scanner := bufio.NewScanner(input)
		defer input.Close()

		for scanner.Scan() {
			line := scanner.Text()

			/**
			fix unclosed xml literals.
			diiiiiirrrrrrrttttttyyyyy hack,,,,
			*/
			if strings.Contains(line, "<br>") {
				line = strings.Replace(line, "<br>", "<br/>", -1)
			}
			if strings.Contains(line, "<meta ") {
				line = strings.TrimSuffix(line, ">")
				line += "/>"
			}
			if strings.Contains(line, "<img ") {
				line = strings.TrimSuffix(line, "></p>")
				line += "/></p>"
			}
			lines += line
		}

		return lines
	}

	// TODO: goroutineで並列にして良い気がする
	for _, basePath := range targetFilePaths {

		// ファイル名だけを取得
		fileNameWithoutExt := strings.TrimSuffix(basePath, path.Ext(basePath))
		folderName := fileNameWithoutExt

		// 同名のディレクトリを作り、basePathファイルを移動する
		os.Mkdir(folderName, 0777)
		os.Rename(currentPath+"/"+basePath, folderName+"/"+basePath)

		targetPath := currentPath + "/" + folderName + "/" + basePath

		/*
			html化
			Macでしか動かないので移植するとしたらめっちゃ詰む箇所
		*/
		htmlize := exec.Command("textutil", "-convert", "html", targetPath)
		htmlize.Output()

		htmlFilePath := currentPath + "/" + folderName + "/" + fileNameWithoutExt + ".html"

		var source html
		data := readHtmlFile(htmlFilePath)

		/*
			read data as xml
		*/
		if err := xml.Unmarshal([]byte(data), &source); err != nil {
			fmt.Println("err", err)
		}

		// fmt.Printf("%#v\n", source)
		{
			style := source.Head.Stl.StyleBodies

			// remove ' from style. for avoiding misencoding.
			source.Head.Stl.StyleBodies = strings.Replace(style, "'", "", -1)
		}

		body := source.Body
		pLines := body.PLines

		// 編集して、完了したら書き直す。
		var replaceLines = make([]PLine, len(pLines))

		for i, ln := range pLines {
			lineContents := ln.Ln

			// set title from contents
			if i == 0 {
				title := strings.Replace(lineContents, "<b>", "", -1)
				title = strings.Replace(title, "</b>", "", -1)
				source.Head.Title = title
			}

			// if file:/// contains in line,
			if strings.Contains(lineContents, "file:///") {
				lineContents = strings.Replace(lineContents, "file:///", "", -1)
			}

			replaceLines[i].Ln = lineContents
			replaceLines[i].Cs = ln.Cs
		}

		source.Body.PLines = replaceLines

		output, _ := os.Create(htmlFilePath)
		enc := xml.NewEncoder(output)
		enc.Indent("  ", "    ")
		enc.Encode(source)

		// デバッグ、変形したファイルを戻す
		if true {
			os.Rename(folderName+"/"+basePath, currentPath+"/"+basePath)
		}
	}
}
