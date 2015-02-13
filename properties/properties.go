package properties

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func getPluginProperties(jsonPropertiesFile string) (map[string]interface{}, error) {
	pluginPropertiesJson, err := ioutil.ReadFile(jsonPropertiesFile)
	if err != nil {
		fmt.Printf("Could not read %s: %s\n", filepath.Base(jsonPropertiesFile), err)
		return nil, err
	}
	var pluginJson interface{}
	if err = json.Unmarshal([]byte(pluginPropertiesJson), &pluginJson); err != nil {
		fmt.Printf("Could not read %s: %s\n", filepath.Base(jsonPropertiesFile), err)
		return nil, err
	}
	return pluginJson.(map[string]interface{}), nil
}

func getGaugePluginVersion(pluginName string) (string, error) {
	pluginProperties, err := getPluginProperties(fmt.Sprintf("%s.json", pluginName))
	if err != nil {
		return nil, error(fmt.Sprintf("Failed to get gauge %s properties file. %s", pluginName, err))
	}
	return pluginProperties["version"].(string), nil
}
