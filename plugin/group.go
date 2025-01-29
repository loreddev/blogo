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

package plugin

const pluginGroupName = "blogo-plugingroup-group"

type Group interface {
	Plugin
	WithPlugins
	Plugins() []Plugin
}

type pluginGroup struct {
	plugins []Plugin
}

func NewGroup(plugins ...Plugin) Group {
	return &pluginGroup{plugins}
}

func (p *pluginGroup) Name() string {
	return pluginGroupName
}

func (p *pluginGroup) Use(plugin Plugin) {
	p.plugins = append(p.plugins, plugin)
}

func (p *pluginGroup) Plugins() []Plugin {
	return p.plugins
}
