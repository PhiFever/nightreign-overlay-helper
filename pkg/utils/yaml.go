package utils

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAML 从 YAML 文件加载数据到 map
func LoadYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SaveYAML 将 map 保存到 YAML 文件
// 使用原子写入（写入临时文件然后替换）以防止损坏
func SaveYAML(path string, data interface{}) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 写入临时文件
	tmpPath := path + ".tmp"
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, yamlData, 0644); err != nil {
		return err
	}

	// 原子替换
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // 出错时清理
		return err
	}

	return nil
}
