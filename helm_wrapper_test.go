package main

import (
	"os"
	"os/exec"
	"io"
	"bytes"
	"sync"
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
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = writer
	wg := new(sync.WaitGroup)
	var out1 bytes.Buffer
	go func() {
		wg.Add(1)
		defer wg.Done()
		_, err = io.Copy(&out1, reader)
		if err != nil {
			t.Errorf("io.Copy error: %s", err)
		}
	}()

	os.Args = []string{
		"./helm-sops",
		"template",
		"./test/charts/test",
		"--values=test/charts/test/values-enc.yaml",
	}
	g_hw.RunHelm()
	writer.Close()
	wg.Wait()


	reader, writer, err = os.Pipe()
	if err != nil {
		panic(err)
	}
	var out2 bytes.Buffer
	go func() {
		wg.Add(1)
		defer wg.Done()
		_, err = io.Copy(&out2, reader)
		if err != nil {
			t.Errorf("io.Copy error: %s", err)
		}
	}()

	args := []string{
		g_hw.helmBinPath,
		"template",
		"./test/charts/test",
		"--values=test/charts/test/values-dec.yaml",
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	cmd.Run()
	writer.Close()
	wg.Wait()

	if !bytes.Equal(out1.Bytes(), out2.Bytes()) {
		t.Errorf("unexpected RunHelm output: \n%s", out1.String())
	}
}
