// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func DockerStopDevsim(t *testing.T) {
	cmd := exec.Command("docker")
	cmd.Args = []string{"docker", "kill", DockerDevsimName}
	err := cmd.Run()
	require.NoError(t, err)
}

func DockerStartDevsim(t *testing.T) {
	cmd := exec.Command("docker")
	cmd.Args = []string{"docker", "start", DockerDevsimName}
	err := cmd.Run()
	require.NoError(t, err)
}
