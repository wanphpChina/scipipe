package proclib

import (
	"fmt"
	"github.com/scipipe/scipipe"
)

// File splitter component

type FileSplitter struct {
	scipipe.Process
	InFile        *scipipe.InPort
	OutSplitFile  *scipipe.OutPort
	LinesPerSplit int
}

func NewFileSplitter(linesPerSplit int) *FileSplitter {
	return &FileSplitter{
		InFile:        scipipe.NewInPort(),
		OutSplitFile:  scipipe.NewOutPort(),
		LinesPerSplit: linesPerSplit,
	}
}

func (proc *FileSplitter) Run() {
	defer proc.OutSplitFile.Close()

	if !scipipe.LogExists {
		scipipe.InitLogAudit()
	}

	fileReader := NewFileReader()

	for ft := range proc.InFile.Chan {
		scipipe.Audit.Println("FileSplitter      Now processing input file ", ft.GetPath(), "...")

		go func() {
			defer close(fileReader.FilePath)
			fileReader.FilePath <- ft.GetPath()
		}()

		go fileReader.Run()

		i := 1
		splitIdx := 1
		splitFt := newSplitFileTargetFromIndex(ft.GetPath(), splitIdx)
		if !splitFt.Exists() {
			splitfile := splitFt.OpenWriteTemp()
			for line := range fileReader.OutLine {
				// If we have not yet reached the number of lines per split ...
				/// ... then just continue to write ...
				if i < splitIdx*proc.LinesPerSplit {
					splitfile.Write(line)
					i++
				} else {
					splitfile.Close()
					splitFt.Atomize()
					scipipe.Audit.Println("FileSplitter      Created split file", splitFt.GetPath())
					proc.OutSplitFile.Chan <- splitFt
					splitIdx++

					splitFt = newSplitFileTargetFromIndex(ft.GetPath(), splitIdx)
					splitfile = splitFt.OpenWriteTemp()
				}
			}
			splitfile.Close()
			splitFt.Atomize()
			scipipe.Audit.Println("FileSplitter      Created split file", splitFt.GetPath())
			proc.OutSplitFile.Chan <- splitFt
		} else {
			scipipe.Audit.Printf("Split file already exists: %s, so skipping.\n", splitFt.GetPath())
		}
	}
}

func (proc *FileSplitter) IsConnected() bool {
	return proc.InFile.IsConnected() &&
		proc.OutSplitFile.IsConnected()
}

func newSplitFileTargetFromIndex(basePath string, splitIdx int) *scipipe.FileTarget {
	return scipipe.NewFileTarget(basePath + fmt.Sprintf(".split_%v", splitIdx))
}
