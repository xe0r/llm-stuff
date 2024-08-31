//go:build windows

package llm

import (
	"encoding/base64"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func encryptData(data []byte) ([]byte, error) {
	var outBlob windows.DataBlob
	inBlob := windows.DataBlob{
		Data: &data[0],
		Size: uint32(len(data)),
	}

	err := windows.CryptProtectData(&inBlob, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &outBlob)
	if err != nil {
		return nil, err
	}

	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outBlob.Data)))

	res := make([]byte, outBlob.Size)
	copy(res, (*[1 << 30]byte)(unsafe.Pointer(outBlob.Data))[:outBlob.Size:outBlob.Size])

	return res, nil
}

func decryptData(data []byte) ([]byte, error) {
	var outBlob windows.DataBlob
	inBlob := windows.DataBlob{
		Data: &data[0],
		Size: uint32(len(data)),
	}

	err := windows.CryptUnprotectData(&inBlob, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &outBlob)
	if err != nil {
		return nil, err
	}

	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outBlob.Data)))

	res := make([]byte, outBlob.Size)
	copy(res, (*[1 << 30]byte)(unsafe.Pointer(outBlob.Data))[:outBlob.Size:outBlob.Size])

	return res, nil
}

const secureTokenFile = ".token.enc"

func secureGetToken() (string, error) {
	if content, err := os.ReadFile(secureTokenFile); err == nil {
		content, err := base64.StdEncoding.DecodeString(string(content))
		if err != nil {
			return "", err
		}

		data, err := decryptData(content)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	if content, err := os.ReadFile(".token"); err == nil {
		encData, err := encryptData(content)
		if err != nil {
			return "", err
		}

		encData = []byte(base64.StdEncoding.EncodeToString(encData))

		if err := os.WriteFile(secureTokenFile, encData, 0600); err != nil {
			return "", err
		}

		return string(content), nil
	}

	return "", os.ErrNotExist
}