package version

const (
	// AppName 是应用程序的名称
	AppName = "nightreign-overlay-helper"
	// AppNameCHS 是应用程序的中文名称
	AppNameCHS = "黑夜君临悬浮助手"
	// Version 是当前版本
	Version = "0.9.0"
	// Author 是应用程序的作者
	Author = "NeuraXmy"
	// GameWindowTitle 是游戏窗口的标题
	GameWindowTitle = "ELDEN RING NIGHTREIGN"
)

// GetFullName 返回带版本的完整名称
func GetFullName() string {
	return AppNameCHS + "v" + Version
}
