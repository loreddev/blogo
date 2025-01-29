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

package plugins

import (
	"cmp"
	"slices"

	"forge.capytal.company/loreddev/blogo/plugin"
)

const priorityGroupName = "blogo-prioritygroup-group"

func NewPriorityGroup(plugins ...plugin.Plugin) PriorityGroup {
	return &priorityGroup{plugins}
}

type PriorityGroup interface {
	plugin.WithPlugins
}

type priorityGroup struct {
	plugins []plugin.Plugin
}

func (p *priorityGroup) Name() string {
	return priorityGroupName
}

func (p *priorityGroup) Use(plugin plugin.Plugin) {
	p.plugins = append(p.plugins, plugin)
}

func (p *priorityGroup) Plugins() []plugin.Plugin {
	slices.SortStableFunc(p.plugins, func(a plugin.Plugin, b plugin.Plugin) int {
		return cmp.Compare(p.getPriority(a, b), p.getPriority(b, a))
	})
	return p.plugins
}

func (p *priorityGroup) getPriority(plugin plugin.Plugin, cmp plugin.Plugin) int {
	if plg, ok := plugin.(PluginWithDynamicPriority); ok {
		return plg.Priority(cmp)
	} else if plg, ok := plugin.(PluginWithPriority); ok {
		return plg.Priority()
	} else {
		return 0
	}
}

type PluginWithPriority interface {
	plugin.Plugin
	Priority() int
}

type PluginWithDynamicPriority interface {
	plugin.Plugin
	Priority(plugin.Plugin) int
}
