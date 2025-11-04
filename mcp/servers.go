package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

type ServerConfig struct {
	Name    string
	Command string
	Args    []string
}

// 默认mcp服务器列表
//func DefaultServerConfigs() []ServerConfig {
//	return []ServerConfig{
//		//{Name: "filesystem", Command: "mcp-server-filesystem", Args: []string{"--root", "."}},
//	}
//}

// 从指定json文件路径加载
func LoadServerConfigFromJSON(path string) ([]ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取json配置失败：%w", err)
	}
	var configs []ServerConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("解析json配置失败：%w", err)
	}

	filtered := make([]ServerConfig, 0, len(configs))
	for _, c := range configs {
		if c.Name == "" || c.Command == "" {
			continue
		}
		filtered = append(filtered, c)
	}
	return filtered, nil
}
