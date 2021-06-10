/*
Copyright 2020 Camptocamp

This file is part of Helm Sops.

Helm Sops is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Helm Sops is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Helm Sops. If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"os"
)

func main() {
	w, err := NewHelmWrapper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[helm-sops] Error: %s\n", err)
		os.Exit(1)
	}

	w.RunHelm()

	for _, err := range w.Errors {
		fmt.Fprintf(os.Stderr, "[helm-sops] Error: %s\n", err)
	}

	os.Exit(w.ExitCode)
}
