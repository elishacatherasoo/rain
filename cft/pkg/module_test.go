package pkg_test

import (
	"testing"

	"github.com/elishacatherasoo/rain/cft/diff"
	"github.com/elishacatherasoo/rain/cft/parse"
	"github.com/elishacatherasoo/rain/cft/pkg"
)

func TestModule(t *testing.T) {

	path := "../../test/modules/expect.yaml"

	expectedTemplate, err := parse.File(path)
	if err != nil {
		t.Error(err)
		return
	}

	pkg.Experimental = true

	packaged, err := pkg.File("../../test/templates/module.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	d := diff.New(packaged, expectedTemplate)
	if d.Mode() != "=" {
		t.Errorf("Output does not match expected: %v", d.Format(true))
	}
}
