package utils

import (
	"os"
	"path/filepath"

	"github.com/PhiFever/nightreign-overlay-helper/pkg/version"
)

// GetAssetPath 返回资源文件的路径
func GetAssetPath(path string) string {
	return filepath.Join("assets", path)
}

// GetDataPath 返回数据文件的路径
func GetDataPath(path string) string {
	return filepath.Join("data", path)
}

// GetAppDataPath 返回应用程序数据文件的路径
func GetAppDataPath(filename string) (string, error) {
	var appDataDir string

	if appData := os.Getenv("APPDATA"); appData != "" {
		// Windows 系统
		appDataDir = filepath.Join(appData, version.AppName)
	} else if home := os.Getenv("HOME"); home != "" {
		// Linux/macOS 系统
		appDataDir = filepath.Join(home, ".local", "share", version.AppName)
	} else {
		// 后备方案
		appDataDir = version.AppName
	}

	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appDataDir, filename), nil
}

// GetDesktopPath 返回桌面文件的路径
func GetDesktopPath(filename string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	desktop := filepath.Join(home, "Desktop")
	if err := os.MkdirAll(desktop, 0755); err != nil {
		return "", err
	}

	if filename != "" {
		return filepath.Join(desktop, filename), nil
	}
	return desktop, nil
}

// GetIconPath 返回图标文件的路径
func GetIconPath() string {
	return GetAssetPath("icon.ico")
}
