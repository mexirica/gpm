package model

type PackagesLoadedMsg struct {
	Packages []Package
	Err      error
}

type OperationFinishedMsg struct {
	Output string
	Err    error
}

type StatusMsg string
