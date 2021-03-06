/*
Copyright (C) 2017 Red Hat, Inc.

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

package addon

import (
	"fmt"
	"reflect"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/provision"
	"github.com/minishift/minishift/cmd/minishift/state"
	"github.com/minishift/minishift/pkg/minikube/constants"
	addOnConfig "github.com/minishift/minishift/pkg/minishift/addon/config"
	"github.com/minishift/minishift/pkg/minishift/addon/manager"
	minishiftConfig "github.com/minishift/minishift/pkg/minishift/config"
	"github.com/minishift/minishift/pkg/minishift/docker"
	"github.com/minishift/minishift/pkg/minishift/openshift"
	"github.com/minishift/minishift/pkg/util/os/atexit"
	"gopkg.in/yaml.v2"
)

// GetAddOnManager returns the addon manager
func GetAddOnManager() *manager.AddOnManager {
	addOnConfigs := minishiftConfig.InstanceConfig.AddonConfig
	m, err := manager.NewAddOnManager(state.InstanceDirs.Addons, addOnConfigs)
	if err != nil {
		atexit.ExitWithMessage(1, fmt.Sprintf("Cannot initialize the add-on manager: %s", err.Error()))
	}

	return m
}

// GetAddOnConfiguration reads the Minishift configuration in $MINISHIFT_HOME/config/config.json related to addons and returns
// a map of addon names to AddOnConfig
func GetAddOnConfiguration() map[string]*addOnConfig.AddOnConfig {
	c, err := minishiftConfig.ReadViperConfig(constants.ConfigFile)
	if err != nil {
		atexit.ExitWithMessage(1, fmt.Sprintf("Cannot read the Minishift configuration: %s", err.Error()))
	}

	var configSlice map[string]interface{}
	if c[addOnConfigKey] != nil {
		configSlice = c[addOnConfigKey].(map[string]interface{})
	} else {
		configSlice = make(map[string]interface{})
	}

	addOnConfigs := make(map[string]*addOnConfig.AddOnConfig)
	for _, entry := range configSlice {
		addOnConfig := &addOnConfig.AddOnConfig{}
		addOnMap := entry.(map[string]interface{})
		fillStruct(addOnMap, addOnConfig)
		addOnConfigs[addOnConfig.Name] = addOnConfig
	}

	return addOnConfigs
}

// fillStruct populates the specified result struct with the data provided in the data map
func fillStruct(data map[string]interface{}, result interface{}) {
	t := reflect.ValueOf(result).Elem()
	for k, v := range data {
		val := t.FieldByName(k)
		val.Set(reflect.ValueOf(v))
	}
}

func determineRoutingSuffix(driver drivers.Driver) string {
	defer func() {
		if r := recover(); r != nil {
			atexit.ExitWithMessage(1, "Cannot determine the routing suffix from the OpenShift master configuration.")
		}
	}()

	sshCommander := provision.GenericSSHCommander{Driver: driver}
	dockerCommander := docker.NewVmDockerCommander(sshCommander)

	raw, err := openshift.ViewConfig(openshift.GetOpenShiftPatchTarget("master"), dockerCommander)
	if err != nil {
		atexit.ExitWithMessage(1, fmt.Sprintf("Cannot get the OpenShift master configuration: %s", err.Error()))
	}

	var config map[interface{}]interface{}
	err = yaml.Unmarshal([]byte(raw), &config)
	if err != nil {
		atexit.ExitWithMessage(1, fmt.Sprintf("Cannot parse the OpenShift master configuration: %s", err.Error()))
	}

	// making assumptions about the master config here. In case the config structure changes, the code might panic here
	return config["routingConfig"].(map[interface{}]interface{})["subdomain"].(string)
}
