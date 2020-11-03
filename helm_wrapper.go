package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sync"
	"syscall"

	"go.mozilla.org/sops/v3/decrypt"
)

type HelmWrapper struct {
	Errors []error
	errMutex sync.Mutex

	ExitCode int

	helmBinPath string
	pipeWriterWaitGroup sync.WaitGroup
	valuesArgRegexp *regexp.Regexp
	temporaryDirectory string
}

func NewHelmWrapper() (*HelmWrapper, error) {
	c := HelmWrapper{}

	c.Errors = []error{}
	c.pipeWriterWaitGroup = sync.WaitGroup{}
	c.valuesArgRegexp = regexp.MustCompile("^(-f|--values)(?:=(.+))?$")

	// Determine the name of the helm binary by examining our binary name
	helmBinName := "helm"
	ourBinName := path.Base(os.Args[0])
	if ourBinName == "helm" || ourBinName == "helm2" || ourBinName == "helm3" {
		helmBinName = fmt.Sprintf("_%s", ourBinName)
	}

	var err error
	c.helmBinPath, err = exec.LookPath(helmBinName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Helm binary '%s': %s", helmBinName, err)
	}

	return &c, nil
}

func (c *HelmWrapper) errorf(msg string, a ...interface{}) error {
	e := fmt.Errorf(msg, a...)
	c.errMutex.Lock()
	c.Errors = append(c.Errors, e)
	c.errMutex.Unlock()
	return e
}

func (c *HelmWrapper) pipeWriter(outPipeName string, data []byte) {
	c.pipeWriterWaitGroup.Add(1)
	defer c.pipeWriterWaitGroup.Done()

	cleartextSecretFile, err := os.OpenFile(outPipeName, os.O_WRONLY, 0)
	if err != nil {
		c.errorf("failed to open cleartext secret pipe '%s' in pipe writer: %s", outPipeName, err)
		return
	}
	defer func() {
		err := cleartextSecretFile.Close()
		if err != nil {
			c.errorf("failed to close cleartext secret pipe '%s' in pipe writer: %s", outPipeName, err)
		}
	}()

	_, err = cleartextSecretFile.Write(data)
	if err != nil {
		c.errorf("failed to write cleartext secret to pipe '%s': %s", outPipeName, err)
	}
}

func (c *HelmWrapper) valuesArg(args []string) (string, string, error) {
	valuesArgRegexpMatches := c.valuesArgRegexp.FindStringSubmatch(args[0])
	if valuesArgRegexpMatches == nil {
		return "", "", nil
	}

	var filename string
	if len(valuesArgRegexpMatches[2]) > 0 {
		// current arg is in the format --values=filename
		filename = valuesArgRegexpMatches[2]
	} else if len(args) > 1 {
		// arg is in the format "-f filename"
		filename = args[1]
	} else {
		return "", "", c.errorf("missing filename after -f or --values")
	}

	cleartextSecretFilename := fmt.Sprintf("%s/%x", c.temporaryDirectory, sha256.Sum256([]byte(filename)))

	return filename, cleartextSecretFilename, nil
}

func (c *HelmWrapper) replaceValueFileArg(args []string, cleartextSecretFilename string) {
	valuesArgRegexpMatches := c.valuesArgRegexp.FindStringSubmatch(args[0])

	// replace the filename with our pipe
	if len(valuesArgRegexpMatches[2]) > 0 {
		args[0] = fmt.Sprintf("%s=%s", valuesArgRegexpMatches[1], cleartextSecretFilename)
	} else {
		args[1] = cleartextSecretFilename
	}
}

func (c *HelmWrapper) mkTmpDir() (func(), error) {
	var err error
	c.temporaryDirectory, err = ioutil.TempDir("", fmt.Sprintf("%s.", path.Base(os.Args[0])))
	if err != nil {
		return nil, c.errorf("failed to create temporary directory: %s", err)
	}
	return func() {
		err := os.RemoveAll(c.temporaryDirectory)
		if err != nil {
			c.errorf("failed to remove temporary directory '%s': %s", c.temporaryDirectory, err)
		}
	}, nil
}

func (c *HelmWrapper) mkPipe(cleartextSecretFilename string) (func(), error) {
	err := syscall.Mkfifo(cleartextSecretFilename, 0600)
	if err != nil {
		return nil, c.errorf("failed to create cleartext secret pipe '%s': %s", cleartextSecretFilename, err)
	}
	return func() {
		err := os.Remove(cleartextSecretFilename)
		if err != nil {
			c.errorf("failed to remove cleartext secret pipe '%s': %s", cleartextSecretFilename, err)
		}
	}, nil
}

func (c *HelmWrapper) RunHelm() {
	var err error
	// Setup temporary directory and defer cleanup
	cleanFn, err := c.mkTmpDir()
	if err != nil {
		return
	}
	defer cleanFn()

	// Loop through arguments looking for --values or -f.
	// If we find a values argument, check if file has a sops section indicating it is encrypted.
	// Setup a named pipe and write the decrypted data into that for helm.
	for i := range os.Args {
		args := os.Args[i:]

		filename, cleartextSecretFilename, err := c.valuesArg(args)
		if err != nil {
			return
		}
		if filename == "" {
			continue
		}

		encrypted, err := DetectSopsYaml(filename)
		if err != nil {
			c.errorf("error checking if file is encrypted: %s", err)
			return
		}
		if !encrypted {
			continue
		}

		c.replaceValueFileArg(args, cleartextSecretFilename)

		cleartextSecrets, err := decrypt.File(filename, "yaml")
		if err != nil {
			c.errorf("failed to decrypt secret file '%s': %s", filename, err)
			return
		}

		cleanFn, err := c.mkPipe(cleartextSecretFilename)
		if err != nil {
			return
		}
		defer cleanFn()

		go c.pipeWriter(cleartextSecretFilename, cleartextSecrets)
	}
	defer c.pipeWriterWaitGroup.Wait()

	cmd := exec.Command(c.helmBinPath, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		c.ExitCode = cmd.ProcessState.ExitCode()
		c.errorf("failed to run Helm: %s", err)
	}
}
