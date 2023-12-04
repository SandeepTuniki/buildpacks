// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package publisher

import (
	"io/ioutil"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

var (
	appHostingCompleteYAMLPath string = testdata.MustGetPath("testdata/apphosting_complete.yaml")
	appHostingInvalidYAMLPath  string = testdata.MustGetPath("testdata/apphosting_invalid.yaml")
)

func TestPublish(t *testing.T) {
	testDir := t.TempDir()
	outputFilePath := testDir + "/output"

	testCasesHappy := []struct {
		desc            string
		filePath        string
		wantBuildSchema buildSchema
	}{{
		desc:     "Correctly parse in provided apphosting.yaml",
		filePath: appHostingCompleteYAMLPath,
		wantBuildSchema: buildSchema{
			BackendResources: backendResources{
				CPU:          3,
				Memory:       512,
				Concurrency:  100,
				MinInstances: 2,
				MaxInstances: 3,
			},
		}},
	}

	testCasesError := []struct {
		desc      string
		filePath  string
		wantError string
	}{{
		desc:      "Throw an error parsing apphosting.yaml when values are invalid",
		filePath:  appHostingInvalidYAMLPath,
		wantError: "apphosting.yaml fields are not valid: concurrency field is not in valid range of '1 <= concurrency <= 1000'",
	}}

	// Testing happy paths
	for _, test := range testCasesHappy {
		err := Publish(test.filePath, "", outputFilePath)
		if err != nil {
			t.Errorf("Error in test '%v'. Error was %v", test.desc, err)
		}

		actualBuildSchemaData, err := ioutil.ReadFile(outputFilePath)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		var actualBuildSchema buildSchema
		err = yaml.Unmarshal(actualBuildSchemaData, &actualBuildSchema)
		if err != nil {
			t.Errorf("error unmarshalling %q as YAML: %v", actualBuildSchemaData, err)
		}

		if diff := cmp.Diff(test.wantBuildSchema, actualBuildSchema); diff != "" {
			t.Errorf("Unexpected YAML for test %v (+got, -want): \n %v", test.desc, diff)
		}
	}

	// Testing error paths
	for _, test := range testCasesError {
		err := Publish(test.filePath, "", outputFilePath)
		if err == nil {
			t.Errorf("Calling Publish did not produce an error for test '%v'", test.desc)
		}

		if diff := cmp.Diff(test.wantError, err.Error()); diff != "" {
			t.Errorf("Error not of expected format for test '%v' (+got, -want): \n %v", test.desc, diff)
		}
	}
}

func TestValidateAppHostingYAMLFields(t *testing.T) {
	testCasesError := []struct {
		desc             string
		appHostingSchema *appHostingSchema
		wantError        bool
	}{{
		desc: "Throw no error when schema is valid",
		appHostingSchema: &appHostingSchema{
			BackendResources: backendResources{
				CPU:          7,
				Memory:       1024,
				Concurrency:  500,
				MinInstances: 2,
				MaxInstances: 4,
			},
		},
		wantError: false,
	},
		{
			desc: "Throw an error when CPU value is invalid",
			appHostingSchema: &appHostingSchema{
				BackendResources: backendResources{
					CPU:          50, // Invalid CPU value
					Memory:       1024,
					Concurrency:  500,
					MinInstances: 2,
					MaxInstances: 4,
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when Memory value is invalid",
			appHostingSchema: &appHostingSchema{
				BackendResources: backendResources{
					CPU:          4,
					Memory:       40000, // Invalid Memory value
					Concurrency:  500,
					MinInstances: 2,
					MaxInstances: 4,
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when concurrency value is invalid",
			appHostingSchema: &appHostingSchema{
				BackendResources: backendResources{
					CPU:          4,
					Memory:       1024,
					Concurrency:  2000, // Invalid Concurrency value
					MinInstances: 2,
					MaxInstances: 4,
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when minInstances value is invalid",
			appHostingSchema: &appHostingSchema{
				BackendResources: backendResources{
					CPU:          4,
					Memory:       1024,
					Concurrency:  500,
					MinInstances: 0, // Invalid minInstances value
					MaxInstances: 4,
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when maxInstances value is invalid",
			appHostingSchema: &appHostingSchema{
				BackendResources: backendResources{
					CPU:          3,
					Memory:       1024,
					Concurrency:  500,
					MinInstances: 2,
					MaxInstances: 101, // Invalid maxInstances value
				},
			},
			wantError: true,
		}}

	for _, test := range testCasesError {
		err := validateAppHostingYAMLFields(test.appHostingSchema)
		gotErr := err != nil

		if gotErr != test.wantError {
			t.Errorf("For test '%v' got %v, want error presence = %v", test.desc, err, test.wantError)
		}
	}
}