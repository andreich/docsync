package mover

import (
	"os/exec"
)

func extractText(in string) ([]string, error) {
	out, err := exec.Command("/usr/bin/pdftotext", in, "-").Output()
	if err != nil {
		return nil, err
	}
	return []string{string(out)}, nil

}
