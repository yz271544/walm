/*
Copyright The Helm Authors.

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

package rules // import "k8s.io/helm/pkg/lint/rules"

import (
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	"k8s.io/helm/pkg/chart"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/lint/support"
)

// Chartfile runs a set of linter rules related to Chart.yaml file
func Chartfile(linter *support.Linter) {
	chartFileName := "Chart.yaml"
	chartPath := filepath.Join(linter.ChartDir, chartFileName)

	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartYamlNotDirectory(chartPath))

	chartFile, err := chartutil.LoadChartfile(chartPath)
	validChartFile := linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartYamlFormat(err))

	// Guard clause. Following linter rules require a parseable ChartFile
	if !validChartFile {
		return
	}

	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartName(chartFile))
	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartNameDirMatch(linter.ChartDir, chartFile))

	// Chart metadata
	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartVersion(chartFile))
	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartMaintainer(chartFile))
	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartSources(chartFile))
	linter.RunLinterRule(support.InfoSev, chartFileName, validateChartIconPresence(chartFile))
	linter.RunLinterRule(support.ErrorSev, chartFileName, validateChartIconURL(chartFile))
}

func validateChartYamlNotDirectory(chartPath string) error {
	fi, err := os.Stat(chartPath)

	if err == nil && fi.IsDir() {
		return errors.New("should be a file, not a directory")
	}
	return nil
}

func validateChartYamlFormat(chartFileError error) error {
	if chartFileError != nil {
		return errors.Errorf("unable to parse YAML\n\t%s", chartFileError.Error())
	}
	return nil
}

func validateChartName(cf *chart.Metadata) error {
	if cf.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func validateChartNameDirMatch(chartDir string, cf *chart.Metadata) error {
	if cf.Name != filepath.Base(chartDir) {
		return errors.Errorf("directory name (%s) and chart name (%s) must be the same", filepath.Base(chartDir), cf.Name)
	}
	return nil
}

func validateChartVersion(cf *chart.Metadata) error {
	if cf.Version == "" {
		return errors.New("version is required")
	}

	version, err := semver.NewVersion(cf.Version)

	if err != nil {
		return errors.Errorf("version '%s' is not a valid SemVer", cf.Version)
	}

	c, err := semver.NewConstraint("> 0")
	if err != nil {
		return err
	}
	valid, msg := c.Validate(version)

	if !valid && len(msg) > 0 {
		return errors.Errorf("version %v", msg[0])
	}

	return nil
}

func validateChartMaintainer(cf *chart.Metadata) error {
	for _, maintainer := range cf.Maintainers {
		if maintainer.Name == "" {
			return errors.New("each maintainer requires a name")
		} else if maintainer.Email != "" && !govalidator.IsEmail(maintainer.Email) {
			return errors.Errorf("invalid email '%s' for maintainer '%s'", maintainer.Email, maintainer.Name)
		} else if maintainer.URL != "" && !govalidator.IsURL(maintainer.URL) {
			return errors.Errorf("invalid url '%s' for maintainer '%s'", maintainer.URL, maintainer.Name)
		}
	}
	return nil
}

func validateChartSources(cf *chart.Metadata) error {
	for _, source := range cf.Sources {
		if source == "" || !govalidator.IsRequestURL(source) {
			return errors.Errorf("invalid source URL '%s'", source)
		}
	}
	return nil
}

func validateChartIconPresence(cf *chart.Metadata) error {
	if cf.Icon == "" {
		return errors.New("icon is recommended")
	}
	return nil
}

func validateChartIconURL(cf *chart.Metadata) error {
	if cf.Icon != "" && !govalidator.IsRequestURL(cf.Icon) {
		return errors.Errorf("invalid icon URL '%s'", cf.Icon)
	}
	return nil
}
