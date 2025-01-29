// Copyright 2025-present Gustavo "Guz" L. de Mello
// Copyright 2025-present The Lored.dev Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitea

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"slices"
	"syscall"
	"time"

	"forge.capytal.company/loreddev/blogo/metadata"
)

type repositoryFS struct {
	metadata map[string]any

	owner string
	repo  string
	ref   string

	client *client
}

func newRepositoryFS(owner, repo, ref string, client *client) fs.FS {
	return &repositoryFS{
		owner:  owner,
		repo:   repo,
		ref:    ref,
		client: client,
	}
}

func (fsys *repositoryFS) Metadata() metadata.Metadata {
	// TODO: Properly implement metadata with contents from the API
	if fsys.metadata == nil || (fsys.metadata != nil && len(fsys.metadata) == 0) {
		m := map[string]any{}
		m["gitea.owner"] = fsys.owner
		m["gitea.repository"] = fsys.repo

		if fsys.ref != "" {
			m["gitea.ref"] = fsys.ref
		}

		fsys.metadata = m
	}
	return metadata.Map(fsys.metadata)
}

func (fsys *repositoryFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	file, _, err := fsys.client.GetContents(fsys.owner, fsys.repo, fsys.ref, name)
	if err == nil {
		return &repositoryFile{
			contentsResponse: *file,

			owner:  fsys.owner,
			repo:   fsys.repo,
			ref:    fsys.ref,
			client: fsys.client,

			contents: nil,
		}, nil
	}

	// If previous call returned a error, it may be because the file is a directory,
	// so we will call from it's parent directory to be able to get it's metadata.
	path := path.Dir(name)
	if path == "." {
		path = ""
	}

	list, res, err := fsys.client.ListContents(fsys.owner, fsys.repo, fsys.ref, path)
	if err != nil {
		if res.StatusCode == http.StatusUnauthorized {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
		} else if res.StatusCode == http.StatusNotFound {
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
		}
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	// If the function is being called to open the root directory, return the
	// repository as a root directory. We are returning it here since we can get
	// a SHA of the past returned files.
	if name == "." {
		sha := ""
		if len(list) > 0 {
			sha = list[0].LastCommitSha
		}

		return &repositoryDirFile{repositoryFile{
			contentsResponse: contentsResponse{
				Name:          fsys.repo,
				Path:          ".",
				SHA:           sha,
				LastCommitSha: sha,
				Type:          "dir",
			},

			owner: fsys.owner,
			repo:  fsys.repo,
			ref:   fsys.ref,

			client: fsys.client,

			contents: nil,
		}, 0}, nil
	}

	i := slices.IndexFunc(list, func(i *contentsResponse) bool {
		return i.Path == name
	})
	if i == -1 {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	dir := list[i]
	if dir.Type != "dir" {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  errors.New("unexpected, directory found is not of type 'dir'"),
		}
	}

	f := &repositoryFile{
		contentsResponse: *dir,

		owner:  fsys.owner,
		repo:   fsys.repo,
		ref:    fsys.ref,
		client: fsys.client,

		contents: nil,
	}

	return &repositoryDirFile{*f, 0}, nil
}

// Implements fs.File to represent a remote file in the repository. The contents of
// the file are filled on the first Read call, reusing the base64-encoded
// *contentsResponse.Content if available, if not, the file calls the API to retrieve
// the raw contents.
//
// To prevent possible content changes after this object has been initialized, if none
// ref is provided, it uses the *contentsResponse.LastCommitSha as a ref.
type repositoryFile struct {
	contentsResponse

	metadata map[string]any

	owner string
	repo  string
	ref   string

	client *client

	contents io.ReadCloser
}

func (f *repositoryFile) Metadata() metadata.Metadata {
	// TODO: Properly implement metadata with contents from the API
	if f.metadata == nil || (f.metadata != nil && len(f.metadata) == 0) {
		m := map[string]any{}
		m["gitea.owner"] = f.owner
		m["gitea.repository"] = f.repo

		if f.ref != "" {
			m["gitea.ref"] = f.ref
		}

		f.metadata = m
	}
	return metadata.Map(f.metadata)
}

func (f *repositoryFile) Stat() (fs.FileInfo, error) {
	return &repositoryFileInfo{*f}, nil
}

func (f *repositoryFile) Read(p []byte) (int, error) {
	var err error

	if f.contents == nil && f.Type == "file" {
		f.contents, err = f.getFileContents()
	}

	if err != nil {
		return 0, errors.Join(errors.New("failed to fetch file contents from API"), err)
	}

	return f.contents.Read(p)
}

func (f *repositoryFile) Close() error {
	return f.contents.Close()
}

func (f *repositoryFile) getFileContents() (io.ReadCloser, error) {
	if *f.Content != "" && f.Encoding != nil && *f.Encoding == "base64" {
		b, err := base64.StdEncoding.DecodeString(*f.Content)
		if err == nil {
			return io.NopCloser(bytes.NewReader(b)), nil
		}
	}

	ref := f.ref
	if ref == "" {
		ref = f.contentsResponse.LastCommitSha
	}

	r, _, err := f.client.GetFileReader(f.owner, f.repo, ref, f.Path, true)
	return r, err
}

// Implements fs.ReadDirFile for the underlying 'repositoryFile'.
// 'repositoryFile' should be of type "dir", and not a list of said directory
// content.
type repositoryDirFile struct {
	repositoryFile
	n int
}

func (f *repositoryDirFile) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (f *repositoryDirFile) Close() error {
	return nil
}

func (f *repositoryDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	list, _, err := f.client.ListContents(f.owner, f.repo, f.ref, f.Path)
	if err != nil {
		return []fs.DirEntry{}, err
	}

	start, end := f.n, f.n+n
	if n <= 0 {
		start, end = 0, len(list)
	} else if end > len(list) {
		end = len(list)
		err = io.EOF
	}

	list = list[start:end]
	entries := make([]fs.DirEntry, len(list))
	for i, v := range list {
		entries[i] = &repositoryDirEntry{repositoryFile{
			contentsResponse: *v,

			owner:  f.owner,
			repo:   f.repo,
			ref:    f.ref,
			client: f.client,
		}}
	}

	f.n = end

	return entries, err
}

// Implements fs.DirEntry for the embedded 'repositoryFile'
type repositoryDirEntry struct {
	repositoryFile
}

func (e *repositoryDirEntry) Name() string {
	i, _ := e.Info()
	return i.Name()
}

func (e *repositoryDirEntry) IsDir() bool {
	i, _ := e.Info()
	return i.IsDir()
}

func (e *repositoryDirEntry) Type() fs.FileMode {
	i, _ := e.Info()
	return i.Mode().Type()
}

func (e *repositoryDirEntry) Info() (fs.FileInfo, error) {
	return &repositoryFileInfo{e.repositoryFile}, nil
}

// Implements fs.FileInfo, getting information from the embedded 'repositoryFile'
type repositoryFileInfo struct {
	repositoryFile
}

func (fi *repositoryFileInfo) Name() string {
	return fi.contentsResponse.Name
}

func (fi *repositoryFileInfo) Size() int64 {
	return fi.contentsResponse.Size
}

func (fi *repositoryFileInfo) Mode() fs.FileMode {
	if fi.Type == "symlink" {
		return os.FileMode(fs.ModeSymlink | syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH)
	} else if fi.IsDir() {
		return os.FileMode(fs.ModeDir | syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH)
	}
	return os.FileMode(syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH)
}

func (fi *repositoryFileInfo) ModTime() time.Time {
	commit, _, err := fi.client.GetSingleCommit(fi.owner, fi.repo, fi.LastCommitSha)
	if err != nil {
		return time.Time{}
	}

	return commit.Created
}

func (fi *repositoryFileInfo) IsDir() bool {
	return fi.Type == "dir"
}

func (fi *repositoryFileInfo) Sys() any {
	return nil
}
