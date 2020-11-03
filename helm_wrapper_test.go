package main

import (
	"os"
    "testing"
)

var g_hw *HelmWrapper

func init() {
    g_hw, _ = NewHelmWrapper()
}

func TestNewHelmWrapper(t *testing.T) {
	// TODO
}

func TestErrorf(t *testing.T) {
	g_hw.Errors = []error{}
	err := g_hw.errorf("test %s %d %t", "a", 1, true)
	if g_hw.Errors[0] != err {
		t.Errorf("errorf(test %%s %%d %%t, a, 1, true) = %s; want %s",  g_hw.Errors[0], err)
	}
}

func TestPipeWriter(t *testing.T) {
	// TODO
}

func TestValuesArg(t *testing.T) {
	res, _, err := g_hw.valuesArg([]string{"-f", "cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"-f\", \"cat.yaml\"}) = %s, %s; want cat.yaml, <nil>",  res, "cat.yaml")
	}

	res, _, err = g_hw.valuesArg([]string{"--values", "cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"--valuse\", \"cat.yaml\"}) = %s, %s; want cat.yaml, <nil>",  res, "cat.yaml")
	}

	res, _, err = g_hw.valuesArg([]string{"--values=cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"--values=cat.yaml\"}) = %s, %s; want cat.yaml, <nil>",  res, "cat.yaml")
	}
}

func TestReplaceValueFileArg(t *testing.T) {
	args := []string{"-f", "cat.yaml"}
	g_hw.replaceValueFileArg(args, "dog.yaml")
	if args[1] != "dog.yaml" {
		t.Errorf("args[1] = %s; want dog.yaml", args[1])
	}

	args = []string{"--values", "cat.yaml"}
	g_hw.replaceValueFileArg(args, "dog.yaml")
	if args[1] != "dog.yaml" {
		t.Errorf("args[1] = %s; want dog.yaml", args[1])
	}

	args = []string{"--values=cat.yaml"}
	g_hw.replaceValueFileArg(args, "dog.yaml")
	if args[0] != "--values=dog.yaml" {
		t.Errorf("args[1] = %s; want --values=dog.yaml", args[1])
	}
}

func TestMkTmpDir(t *testing.T) {
	// ensure no errors
	cleanFn, err := g_hw.mkTmpDir()
	if err != nil {
		t.Errorf("mkTmpDir error: %s", err)
	}

	// dir exists
	if _, err = os.Stat(g_hw.temporaryDirectory); err != nil {
		t.Errorf("mkTmpDir stat error: %s", err)
	}

	// ensure dir is deleted
	cleanFn()
	if _, err = os.Stat(g_hw.temporaryDirectory); err == nil {
		t.Errorf("mkTmpDir cleanup func did not work")
	} else if !os.IsNotExist(err) {
		t.Errorf("mkTmpDir cleanup something went wrong: %s", err)
	}
}

func TestMkPipe(t *testing.T) {
	// ensure no errors
	cleanFn, err := g_hw.mkPipe("cat.yaml")
	if err != nil {
		t.Errorf("mkPipe error: %s", err)
	}

	// file exists
	if _, err = os.Stat("cat.yaml"); err != nil {
		t.Errorf("mkPipe stat error: %s", err)
	}

	// ensure file is deleted
	cleanFn()
	if _, err = os.Stat("cat.yaml"); err == nil {
		t.Errorf("mkPipe cleanup func did not work")
	} else if !os.IsNotExist(err) {
		t.Errorf("mkPipe cleanup something went wrong: %s", err)
	}
}

func TestRunHelm(t *testing.T) {
	// TODO
}
