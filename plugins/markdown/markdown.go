package markdown

import (
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"

	"forge.capytal.company/loreddev/blogo/plugin"
)

const pluginName = "blogo-markdown-renderer"

type p struct {
	parser   parser.Parser
	renderer renderer.Renderer
}

func New() plugin.Plugin {
	m := goldmark.New(
		goldmark.WithExtensions(
			extension.NewLinkify(),
			meta.Meta,
		),
	)

	return &p{
		parser:   m.Parser(),
		renderer: m.Renderer(),
	}
}

func (p *p) Name() string {
	return pluginName
}

func (p *p) Render(f fs.File, w io.Writer) error {
	stat, err := f.Stat()
	if err != nil || !strings.HasSuffix(stat.Name(), ".md") {
		return errors.New("does not support file")
	}

	src, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	txt := text.NewReader(src)

	ast := p.parser.Parse(txt)

	return p.renderer.Render(w, src, ast)
}
