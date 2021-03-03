package output

import (
	"encoding/json"
	"os"

	"github.com/cloudskiff/driftctl/pkg/output"

	"github.com/cloudskiff/driftctl/pkg/analyser"
)

const JSONOutputType = "json"
const JSONOutputExample = "json://PATH/TO/FILE.json"

type JSON struct {
	path string
}

func NewJSON(path string) *JSON {
	return &JSON{path}
}

func (c *JSON) GetInfoPrinter() output.Printer {
	if c.isStdOut() {
		return &output.VoidPrinter{}
	}
	return output.NewConsolePrinter()
}

func (c *JSON) Write(analysis *analyser.Analysis) error {
	file := os.Stdout
	if !c.isStdOut() {
		f, err := os.OpenFile(c.path, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		file = f
	}

	json, err := json.MarshalIndent(analysis, "", "\t")
	if err != nil {
		return err
	}
	if _, err := file.Write(json); err != nil {
		return err
	}
	return nil
}

func (c *JSON) isStdOut() bool {
	return c.path == "/dev/stdout" || c.path == "stdout"
}
