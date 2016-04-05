// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyze

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Step1 []struct {
		DataPathList      []string `yaml:"data_path_list"`
		DataBenchmarkPath string   `yaml:"data_benchmark_path"`
		OutputPath        string   `yaml:"output_path"`
	} `yaml:"step1"`

	Step2 []struct {
		DataList []struct {
			Path string `yaml:"path"`
			Name string `yaml:"name"`
		} `yaml:"data_list"`
		OutputPath string `yaml:"output_path"`
	} `yaml:"step2"`

	Step3 []struct {
		DataPath string `yaml:"data_path"`
		Title    string `yaml:"title"`
		PlotList []struct {
			Lines []struct {
				Column string `yaml:"column"`
				Legend string `yaml:"legend"`
			} `yaml:"lines"`
			XAxis          string   `yaml:"x_axis"`
			YAxis          string   `yaml:"y_axis"`
			OutputPathList []string `yaml:"output_path_list"`
		} `yaml:"plot_list"`
	} `yaml:"step3"`

	Step4 struct {
		Preface string `yaml:"preface"`
		Results []struct {
			Title  string `yaml:"title"`
			Images []struct {
				ImageTitle string `yaml:"image_title"`
				ImagePath  string `yaml:"image_path"`
				ImageType  string `yaml:"image_type"`
			} `yaml:"images"`
		} `yaml:"results"`
		OutputPath string `yaml:"output_path"`
	} `yaml:"step4"`
}

func ReadConfig(fpath string) (Config, error) {
	bts, err := ioutil.ReadFile(fpath)
	if err != nil {
		return Config{}, err
	}
	rs := Config{}
	if err := yaml.Unmarshal(bts, &rs); err != nil {
		return Config{}, err
	}
	return rs, nil
}
