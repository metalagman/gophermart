package accrual

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

// readJSON into interface
func readJSON(in io.ReadCloser, v interface{}) error {
	body, err := ioutil.ReadAll(in)
	_ = in.Close()
	if err != nil {
		return fmt.Errorf("io read: %w", err)
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	return nil
}

// readString into struct
func readString(in io.ReadCloser) string {
	body, err := ioutil.ReadAll(in)
	_ = in.Close()
	if err != nil {
		return ""
	}

	return string(body)
}
