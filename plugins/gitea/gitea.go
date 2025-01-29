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
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"forge.capytal.company/loreddev/blogo/plugin"
)

const pluginName = "blogo-gitea-sourcer"

type p struct {
	client *client

	owner string
	repo  string
	ref   string
}

type Opts struct {
	HTTPClient *http.Client
	Ref        string
}

func New(owner, repo, apiUrl string, opts ...Opts) plugin.Plugin {
	opt := Opts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.HTTPClient == nil {
		opt.HTTPClient = http.DefaultClient
	}

	u, err := url.Parse(apiUrl)
	if err != nil {
		panic(
			fmt.Sprintf(
				"%s: %q is not a valid URL. Err: %q",
				pluginName,
				apiUrl,
				err.Error(),
			),
		)
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/api/v1"
	} else {
		u.Path = strings.TrimSuffix(u.Path, "/api/v1")
	}

	client := newClient(u.String(), opt.HTTPClient)

	return &p{
		client: client,

		owner: owner,
		repo:  repo,
		ref:   opt.Ref,
	}
}

func (p *p) Name() string {
	return pluginName
}

func (p *p) Source() (fs.FS, error) {
	return newRepositoryFS(p.owner, p.repo, p.ref, p.client), nil
}
