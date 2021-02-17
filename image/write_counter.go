package image

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type WriteCounter struct {
	total       uint64
	index       int
	layerDigest string
	progress    *Progress
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += uint64(n)
	wc.progress.PrintProgress(wc)
	return n, nil
}

func (wc *WriteCounter) done() {
	wc.progress.done(wc)
}

type Progress struct {
	mu          sync.Mutex
	MaxLine     int
	currentLine int
}

func (p *Progress) PrintProgress(wc *WriteCounter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//移动到当前行
	p.move(wc.index + 1)

	fmt.Printf("\r%s: 下载中... %s", wc.layerDigest[:13], humanSize(wc.total))

	//移动到最后一行
	p.move(p.MaxLine)
}

func (p *Progress) done(wc *WriteCounter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	//移动到当前行
	p.move(wc.index + 1)
	fmt.Printf("\r%s: 下载完成  %s", wc.layerDigest[:13], humanSize(wc.total))
	p.move(p.MaxLine)
}

func (p *Progress) skip(index int, layerDigest string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.move(index + 1)
	fmt.Printf("\r%s: 文件层已存在 ", layerDigest[:13])
	p.move(p.MaxLine)
}

func (p *Progress) move(line int) {
	fmt.Printf("\033[%dA\033[%dB", p.currentLine, line)
	p.currentLine = line
}

func DownloadFile(index int, layerDigest, filepath string, url string, progress *Progress) error {

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
	writeCounter := &WriteCounter{
		layerDigest: layerDigest,
		progress:    progress,
		index:       index,
	}
	if progress != nil {
		if _, err = io.Copy(out, io.TeeReader(resp.Body, writeCounter)); err != nil {
			out.Close()
			return err
		}

		writeCounter.done()
	} else {
		if _, err = io.Copy(out, resp.Body); err != nil {
			out.Close()
			return err
		}
	}

	// The progress use the same line so print a new line once it's finished downloading

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
