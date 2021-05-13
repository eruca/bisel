package manager

import (
	"os"

	"github.com/eruca/bisel/btypes"
	"github.com/eruca/bisel/utils"
	"github.com/pelletier/go-toml"
)

const configFile = "config.toml"

// 否则默认使用responseType作为ConfigResponseType
func defaultResponseType(reqType string, successed bool) string {
	if successed {
		return reqType + "_success"
	}
	return reqType + "_failure"
}

func LoadConfigFile() btypes.Config {
	config := btypes.Config{}
	if utils.IsNotExist(configFile) {
		config.SetDefault()
		return config
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	err = toml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	config.SetDefault()
	return config
}
