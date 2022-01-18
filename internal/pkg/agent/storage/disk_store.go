// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/hectane/go-acl"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-poc/internal/pkg/agent/errors"
)

// NewDiskStore creates an unencrypted disk store.
func NewDiskStore(target string) *DiskStore {
	return &DiskStore{target: target}
}

// Exists check if the store file exists on the disk
func (d *DiskStore) Exists() (bool, error) {
	_, err := os.Stat(d.target)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Delete deletes the store file on the disk
func (d *DiskStore) Delete() error {
	return os.Remove(d.target)
}

// Save accepts a persistedConfig and saved it to a target file, to do so we will
// make a temporary files if the write is successful we are replacing the target file with the
// original content.
func (d *DiskStore) Save(in io.Reader) error {
	tmpFile := d.target + ".tmp"

	fd, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perms)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not save to %s", tmpFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	// Always clean up the temporary file and ignore errors.
	defer os.Remove(tmpFile)

	if _, err := io.Copy(fd, in); err != nil {
		_ = fd.Close()

		return errors.New(err, "could not save content on disk",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	_ = fd.Sync()

	if err := fd.Close(); err != nil {
		return errors.New(err, "could not close temporary file",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, tmpFile))
	}

	if err := file.SafeFileRotate(d.target, tmpFile); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not replace target file %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	if err := acl.Chmod(d.target, perms); err != nil {
		return errors.New(err,
			fmt.Sprintf("could not set permissions target file %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}

	return nil
}

// Load return a io.ReadCloser for the target file.
func (d *DiskStore) Load() (io.ReadCloser, error) {
	fd, err := os.OpenFile(d.target, os.O_RDONLY|os.O_CREATE, perms)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not open %s", d.target),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, d.target))
	}
	return fd, nil
}