/*
Copyright 2019-2020 vChain, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package immudb

import (
	"github.com/spf13/cobra/doc"
	"os"
	"path/filepath"
)

type ManpageService interface {
	InstallManPages(dir string) error
	UninstallManPages(dir string) error
}

type ManpageServiceImmudb struct{}

// InstallManPages installs man pages
func (ms ManpageServiceImmudb) InstallManPages(dir string) error {
	header := &doc.GenManHeader{
		Title:   "immuadmin service",
		Section: "1",
		Source:  "Generated by immuadmin installer",
	}
	_ = os.Mkdir(dir, os.ModePerm)
	err := doc.GenManTree(NewCmd(), header, dir)
	if err != nil {
		return err
	}
	return nil
}

// UninstallManPages uninstalls man pages
func (ms ManpageServiceImmudb) UninstallManPages(dir string) error {
	err1 := os.Remove(filepath.Join(dir, "immudb-version.1"))
	err2 := os.Remove(filepath.Join(dir, "immudb.1"))
	switch {
	case err1 != nil:
		return err1
	case err2 != nil:
		return err2
	default:
		return nil
	}
}