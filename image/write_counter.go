package image

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type WriteCounter struct {
	Total       uint64
	LayerDigest string
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {

	fmt.Printf("\r%s: 下载中... %s", wc.LayerDigest[:13], humanSize(wc.Total))
}

func DownloadFile(layerDigest, filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{LayerDigest: layerDigest}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		out.Close()
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(filepath+".tmp", filepath); err != nil {
		return err
	}
	return nil
}

func humanSize(b uint64) string {
	kb := float64(b) / 1024
	size := fmt.Sprintf("%.2f Kb", kb)

	if mb := kb / 1024; mb > 1 {
		size = fmt.Sprintf("%.2f Mb   ", mb)
	}
	return size
}
