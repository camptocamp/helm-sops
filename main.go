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

var (
	valuesArgRegexp      *regexp.Regexp
	secretFilenameRegexp *regexp.Regexp
)

func init() {
	valuesArgRegexp = regexp.MustCompile("^(-f|--values)(?:=(.+))?$")
	secretFilenameRegexp = regexp.MustCompile("^((?:.*/)?secrets(?:(?:-|\\.|_).+)?.yaml)$")
}

func runHelm() (errs []error) {
	var helmPath string
	var err error

	switch executableName := path.Base(os.Args[0]); executableName {
	case "helm", "helm2", "helm3":
		executableName = fmt.Sprintf("_%s", executableName)

		helmPath, err = exec.LookPath(executableName)

		if err != nil {
			return append(errs, fmt.Errorf("failed to find Helm binary '%s'", executableName))
		}
	default:
		helmPath, err = exec.LookPath("helm")

		if err != nil {
			return append(errs, fmt.Errorf("failed to find Helm binary 'helm'"))
		}
	}

	temporaryDirectory, err := ioutil.TempDir("", fmt.Sprintf("%s.", path.Base(os.Args[0])))

	if err != nil {
		return append(errs, fmt.Errorf("failed to create temporary directory: %s", err))
	}

	defer func() {
		err := os.RemoveAll(temporaryDirectory)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to remove temporary directory '%s': %s", temporaryDirectory, err))

			return
		}
	}()

loop:
	for args := os.Args[1:]; len(args) > 0; args = args[1:] {
		arg := args[0]

		if valuesArgRegexpMatches := valuesArgRegexp.FindStringSubmatch(arg); valuesArgRegexpMatches != nil {
			var filename string

			switch {
			case len(valuesArgRegexpMatches[2]) > 0:
				filename = valuesArgRegexpMatches[2]
			case len(args) > 1:
				filename = args[1]
			default:
				break loop
			}

			if secretFilenameRegexpMatches := secretFilenameRegexp.FindStringSubmatch(filename); secretFilenameRegexpMatches != nil {
				secretFilename := secretFilenameRegexpMatches[0]
				cleartextSecretFilename := fmt.Sprintf("%s/%x", temporaryDirectory, sha256.Sum256([]byte(secretFilename)))

				cleartextSecrets, err := decrypt.File(secretFilename, "yaml")

				if err != nil {
					return append(errs, fmt.Errorf("failed to decrypt secret file '%s': %s", secretFilename, err))
				}

				err = syscall.Mkfifo(cleartextSecretFilename, 0600)

				if err != nil {
					return append(errs, fmt.Errorf("failed to create cleartext secret pipe '%s': %s", cleartextSecretFilename, err))
				}

				defer func(cleartextSecretFilename string) {
					err := os.Remove(cleartextSecretFilename)

					if err != nil {
						errs = append(errs, fmt.Errorf("failed to remove cleartext secret pipe '%s': %s", cleartextSecretFilename, err))

						return
					}
				}(cleartextSecretFilename)

				var errs1 []error
				var errs2 []error

				pipeWriterWaitGroup := sync.WaitGroup{}
				pipeCloseChannel := make(chan struct{})

				defer func(errs1 *[]error, errs2 *[]error, pipeWriterWaitGroup *sync.WaitGroup, pipeCloseChannel chan struct{}) {
					close(pipeCloseChannel)

					pipeWriterWaitGroup.Wait()

					errs = append(errs, *errs1...)
					errs = append(errs, *errs2...)
				}(&errs1, &errs2, &pipeWriterWaitGroup, pipeCloseChannel)

				pipeWriterWaitGroup.Add(2)

				pipeWriterUnlockedChannel := make(chan struct{}, 1)

				go func(cleartextSecretFilename string, cleartextSecrets []byte, errs *[]error, pipeWriterUnlockedChannel chan struct{}, pipeWriterWaitGroup *sync.WaitGroup) {
					defer pipeWriterWaitGroup.Done()

					cleartextSecretFile, err := os.OpenFile(cleartextSecretFilename, os.O_WRONLY, 0)

					pipeWriterUnlockedChannel <- struct{}{}

					if err != nil {
						*errs = append(*errs, fmt.Errorf("failed to open cleartext secret pipe '%s' in pipe writer: %s", cleartextSecretFilename, err))

						return
					}

					defer func() {
						err := cleartextSecretFile.Close()

						if err != nil {
							*errs = append(*errs, fmt.Errorf("failed to close cleartext secret pipe '%s' in pipe writer: %s", cleartextSecretFilename, err))

							return
						}
					}()

					_, err = cleartextSecretFile.Write(cleartextSecrets)

					if err != nil {
						*errs = append(*errs, fmt.Errorf("failed to write cleartext secret to pipe '%s': %s", cleartextSecretFilename, err))

						return
					}
				}(cleartextSecretFilename, cleartextSecrets, &errs1, pipeWriterUnlockedChannel, &pipeWriterWaitGroup)

				go func(cleartextSecretFilename string, errs *[]error, pipeCloseChannel chan struct{}, pipeWriterUnlockedChannel chan struct{}, pipeWriterWaitGroup *sync.WaitGroup) {
					defer pipeWriterWaitGroup.Done()

					<-pipeCloseChannel

					select {
					case <-pipeWriterUnlockedChannel:
						return
					default:
					}

					cleartextSecretFile, err := os.OpenFile(cleartextSecretFilename, os.O_RDWR, 0)

					if err != nil {
						*errs = append(*errs, fmt.Errorf("failed to open cleartext secret pipe '%s' in pipe closer: %s", cleartextSecretFilename, err))

						return
					}

					<-pipeWriterUnlockedChannel

					defer func() {
						err := cleartextSecretFile.Close()

						if err != nil {
							*errs = append(*errs, fmt.Errorf("failed to close cleartext secret pipe '%s' in pipe closer: %s", cleartextSecretFilename, err))

							return
						}
					}()
				}(cleartextSecretFilename, &errs2, pipeCloseChannel, pipeWriterUnlockedChannel, &pipeWriterWaitGroup)

				if len(valuesArgRegexpMatches[2]) > 0 {
					args[0] = fmt.Sprintf("%s=%s", valuesArgRegexpMatches[1], cleartextSecretFilename)
				} else {
					args[1] = cleartextSecretFilename
					args = args[1:]
				}
			}
		}
	}

	cmd := exec.Command(helmPath, os.Args[1:]...)

	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		return append(errs, fmt.Errorf("failed to run Helm: %s", err))
	}

	return
}

func main() {
	errs := runHelm()

	exitCode := 0

	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "[helm-sops] Error: %s\n", err)

		exitCode = 1
	}

	os.Exit(exitCode)
}
