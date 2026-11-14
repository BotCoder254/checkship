/*
   Copyright 2020 Docker Compose CLI authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// checkship patch: BuildKit progress used to be printed to os.Stdout, whatever output the
// docker CLI was configured with. Programs embedding compose (checkship writes each deployment
// to its own log) could not capture it.

package compose

import (
	"io"
	"os"

	"github.com/containerd/console"
)

// Returns where BuildKit progress is printed: the process standard output when the CLI output
// is that terminal (the docker compose CLI behaviour), the CLI output otherwise.
func (s *composeService) buildProgressOutput() console.File {
	out := s.stdout()

	if out.IsTerminal() {
		return os.Stdout
	}

	return writerFile{out}
}

// Satisfies console.File for an output which is not a file. BuildKit only uses it as an
// io.Writer when it is not a console, which is the case here.
type writerFile struct {
	io.Writer
}

func (writerFile) Read([]byte) (int, error) { return 0, io.EOF }
func (writerFile) Close() error             { return nil }
func (writerFile) Fd() uintptr              { return ^uintptr(0) }
func (writerFile) Name() string             { return "" }
