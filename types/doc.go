/*
Package types provides type definition of the moca-go-sdk client
*/
package types

import (
	"bytes"
	"os"
)

func CompareFiles(fileL string, fileR string) (bool, error) {
	finL, err := os.Open(fileL)
	if err != nil {
		return false, err
	}
	defer func() { _ = finL.Close() }()

	finR, err := os.Open(fileR)
	if err != nil {
		return false, err
	}
	defer func() { _ = finR.Close() }()

	statL, err := finL.Stat()
	if err != nil {
		return false, err
	}

	statR, err := finR.Stat()
	if err != nil {
		return false, err
	}

	if statL.Size() != statR.Size() {
		return false, nil
	}

	size := statL.Size()
	if size > 102400 {
		size = 102400
	}

	bufL := make([]byte, size)
	bufR := make([]byte, size)
	for {
		n, _ := finL.Read(bufL)
		if n == 0 {
			break
		}

		n, _ = finR.Read(bufR)
		if n == 0 {
			break
		}

		if !bytes.Equal(bufL, bufR) {
			return false, nil
		}
	}

	return true, nil
}
