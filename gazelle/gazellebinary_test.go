/* Copyright 2018 The Bazel Authors. All rights reserved.

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

package gazelle

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
)

var (
	gazellePath = flag.String("gazelle", "", "path to gazelle binary")
	baseFiles   = []testtools.FileSpec{
		{Path: "WORKSPACE"},
		{Path: "jest.config.js"},
		{Path: "hello_world/main.js", Content: `
import {format} from "date-fns";

import fonts from "~~/shared/fonts";
import colors from "@/shared/colors";

const date = format(new Date(2014, 0, 24), 'MM/DD/YYYY');
console.log(date + fonts + colors);
export default date;
`},
		{Path: "shared/fonts.js", Content: `
export default "Helvetica";
`},
		{Path: "shared/colors.js", Content: `
export default "Green";
`},
		{Path: "shared/colors.test.js", Content: `
import { something } from '@test/utils';

import colors from './colors';
import date from '../hello_world/main';

// Imagine some tests here
`},
	}
)

func TestMain(m *testing.M) {
	_, ok := os.LookupEnv("TEST_TARGET")
	if !ok {
		// Skip all tests if we aren't run by Bazel
		return
	}

	flag.Parse()
	if abs, err := filepath.Abs(*gazellePath); err != nil {
		log.Fatalf("unable to find absolute path for gazelle: %v\n", err)
		os.Exit(1)
	} else {
		*gazellePath = abs
	}
	os.Exit(m.Run())
}

func TestGazelleBinary(t *testing.T) {
	files := append(baseFiles)
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	cmd := exec.Command(*gazellePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{{
		Path: "BUILD.bazel",
		Content: `load("@ecosia_bazel_rules_nodejs_contrib//:defs.bzl", "js_library")

js_library(
    name = "jest.config",
    srcs = ["jest.config.js"],
    visibility = ["//visibility:public"],
)`,
	},{
		Path: "hello_world/BUILD.bazel",
		Content: `load("@ecosia_bazel_rules_nodejs_contrib//:defs.bzl", "js_library")

js_library(
    name = "main",
    srcs = ["main.js"],
    visibility = ["//visibility:public"],
    deps = [
        "//shared:colors",
        "//shared:fonts",
        "@npm//date-fns",
    ],
)`,
	}, {
		Path: "shared/BUILD.bazel",
		Content: `load("@ecosia_bazel_rules_nodejs_contrib//:defs.bzl", "jest_node_test", "js_library")

js_library(
    name = "colors",
    srcs = ["colors.js"],
    visibility = ["//visibility:public"],
)

jest_node_test(
    name = "colors.test",
    srcs = ["colors.test.js"],
    visibility = ["//visibility:public"],
    deps = [
        ":colors",
        "//hello_world:main",
        "@npm//@test/utils",
    ],
)

js_library(
    name = "fonts",
    srcs = ["fonts.js"],
    visibility = ["//visibility:public"],
)
`,
	}})
}