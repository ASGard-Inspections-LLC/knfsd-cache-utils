/*
 Copyright 2022 Google LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMetricTransformsMatch(t *testing.T) {
	type Transform struct {
		Action    string `yaml:"action"`
		Include   string `yaml:"include"`
		NewName   string `yaml:"new_name"`
		MatchType string `yaml:"match_type"`
	}

	type Config struct {
		Processors struct {
			MetricsTransform struct {
				Transforms []Transform `yaml:"transforms"`
			} `yaml:"metricstransform"`
		} `yaml:"processors"`
	}

	metrics, err := findAllMetricNames()
	require.NoError(t, err)

	var conf Config
	err = readYamlFile("config/common.yaml", &conf)
	require.NoError(t, err)

	var transformed []string
	for _, t := range conf.Processors.MetricsTransform.Transforms {
		if t.Action == "update" &&
			t.Include != "" &&
			t.NewName != "" &&
			(t.MatchType == "" || t.MatchType == "strict") {
			transformed = append(transformed, t.Include)
		}
	}

	// check that every metric has a transform, and that there are no transforms
	// without a matching metric (e.g. the metric has been removed/renamed).
	assert.ElementsMatch(t, transformed, metrics)
}

func TestNoDuplicateNames(t *testing.T) {
	names, err := findAllMetricNames()
	require.NoError(t, err)

	counts := make(map[string]int)
	for _, n := range names {
		counts[n]++
	}

	var dups []string
	for n, c := range counts {
		if c > 1 {
			dups = append(dups, n)
		}
	}

	assert.Empty(t, dups)
}

func findAllMetricNames() (names []string, err error) {
	files, err := findMetadata()
	if err != nil {
		return
	}

	for _, path := range files {
		var found []string
		found, err = readMetricNames(path)
		if err != nil {
			return
		}
		names = append(names, found...)
	}

	return
}

func findMetadata() ([]string, error) {
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if d.Name() == "metadata.yaml" && d.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func readMetricNames(path string) (names []string, err error) {
	type Metadata struct {
		Metrics map[string]struct{} `yaml:"metrics"`
	}

	var meta Metadata
	err = readYamlFile(path, &meta)
	if err != nil {
		return
	}

	for k := range meta.Metrics {
		names = append(names, k)
	}
	return
}

func readYamlFile(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	err = d.Decode(v)
	if err != nil {
		return err
	}

	return nil
}
