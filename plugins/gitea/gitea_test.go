package gitea_test

import (
	"io"
	"testing"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/blogo/plugins/gitea"
)

func TestSource(t *testing.T) {
	p := gitea.New("loreddev", "x", "https://forge.capytal.company")

	s := p.(plugin.Sourcer)

	fs, err := s.Source()
	if err != nil {
		t.Fatalf("Failed to source file system: %s %v", err.Error(), err)
	}

	file, err := fs.Open("blogo/LICENSE")
	if err != nil {
		t.Fatalf("Failed to open file: %s %v", err.Error(), err)
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read contents of file: %s %v", err.Error(), err)
	}

	t.Logf("Successfully read contents of file: %s", string(contents))
}
