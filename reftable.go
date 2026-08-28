/*
Copyright 2020 Google LLC

Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file or at
https://developers.google.com/open-source/licenses/bsd
*/

package reftable

// ReadRef reads a ref record by name.
func ReadRef(tab Table, name string) (*RefRecord, error) {
	it, err := tab.SeekRef(name)
	if err != nil {
		return nil, err
	}

	var ref RefRecord
	ok, err := it.NextRef(&ref)
	if err != nil || !ok {
		return nil, err
	}

	if ref.RefName != name {
		return nil, nil
	}

	return &ref, nil
}

// ReadRef reads the most recent log record of a certain ref.
func ReadLogAt(tab Table, name string, updateIndex uint64) (*LogRecord, error) {
	it, err := tab.SeekLog(name, updateIndex)
	if err != nil {
		return nil, err
	}

	var rec LogRecord
	ok, err := it.NextLog(&rec)
	if err != nil || !ok {
		return nil, err
	}

	if rec.RefName != name {
		return nil, nil
	}

	return &rec, nil
}
