# Helm Sops

Helm Sops is a Helm wrapper which decrypts Sops encrypted value files before invoking Helm. It does so by using named pipes to pass cleartext value files to Helm in order to avoid secrets being written to disk in clear.

## Installation

### Prerequisites

Helm is needed for Helm Sops to work. Follow the instructions [here](https://helm.sh/docs/intro/install/) to install it.

### Getting Helm Sops binary

#### Helm Sops releases

Helm Sops released binaries can be downloaded from [GitHub](https://github.com/camptocamp/helm-sops/releases).

#### Building from sources

Helm Sops can be built using the `go build` command.

### Deploying Helm Sops

Deploy Helm Sops executable (`helm-sops`) in a directory present in the *PATH*. When invoking Helm Sops, it will look for a Helm executable named `helm` in the *PATH*.

Alternatively, Helm Sops executable can be renamed `helm` before deploying it. When invoked as `helm`, Helm Sops will look for a Helm executable named `_helm` in the *PATH*.

## Usage

Create encrypted value files using [Sops](https://github.com/mozilla/sops) and name them using the `^secrets((-|\.|_).+)?\.yaml$` pattern (for example `secrets.yaml`, `secrets-production.yaml` or `secrets.lab.yaml`).

To pass these encrypted value files to Helm, just invoke Helm Sops with the same arguments which would be used for the Helm invocation (for example  
`helm-sops template . --values secrets.yaml --values secrets-production.yaml` or  
`helm template . --values secrets.yaml --values secrets-production.yaml`  
depending on how Helm Sops was deployed).

## Git diff helper

The following script (`sops-git-diff-helper`) can be placed in the *PATH* to be used as a Git diff helper for Sops encrypted value files:

```sh
#! /bin/sh

if [ $# -ne 1 ]
then
	exit 1
fi

if [ -n "${SOPS_ENCRYPTED_DIFF}" ]
then
	cat "$1"
else
	sops -d "$1" 2>&1 || cat "$1"
fi
```

To enable it, run `git config --global diff.sops.textconv sops-git-diff-helper`
