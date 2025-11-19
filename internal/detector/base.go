package detector

import (
	"image"
	"sync"
)

// Detector 是所有检测器的基础接口
type Detector interface {
	// Name 返回检测器的名称
	Name() string

	// Detect 对给定的图像执行检测
	// 返回检测结果和遇到的任何错误
	Detect(img image.Image) (interface{}, error)

	// Initialize 使用必要的资源初始化检测器
	Initialize() error

	// Cleanup 释放检测器使用的资源
	Cleanup() error

	// IsEnabled 返回检测器是否启用
	IsEnabled() bool

	// SetEnabled 设置检测器的启用状态
	SetEnabled(enabled bool)
}

// BaseDetector 为所有检测器提供通用功能
type BaseDetector struct {
	name    string
	enabled bool
	mu      sync.RWMutex
}

// NewBaseDetector 创建一个新的 BaseDetector
func NewBaseDetector(name string) *BaseDetector {
	return &BaseDetector{
		name:    name,
		enabled: true,
	}
}

// Name 返回检测器的名称
func (b *BaseDetector) Name() string {
	return b.name
}

// IsEnabled 返回检测器是否启用
func (b *BaseDetector) IsEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.enabled
}

// SetEnabled 设置检测器的启用状态
func (b *BaseDetector) SetEnabled(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = enabled
}

// DetectorRegistry 管理所有检测器
type DetectorRegistry struct {
	detectors map[string]Detector
	mu        sync.RWMutex
}

// NewDetectorRegistry 创建一个新的检测器注册表
func NewDetectorRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: make(map[string]Detector),
	}
}

// Register 注册一个检测器
func (r *DetectorRegistry) Register(detector Detector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors[detector.Name()] = detector
}

// Get 根据名称检索检测器
func (r *DetectorRegistry) Get(name string) (Detector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	detector, ok := r.detectors[name]
	return detector, ok
}

// GetAll 返回所有已注册的检测器
func (r *DetectorRegistry) GetAll() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	detectors := make([]Detector, 0, len(r.detectors))
	for _, detector := range r.detectors {
		detectors = append(detectors, detector)
	}
	return detectors
}

// InitializeAll 初始化所有已注册的检测器
func (r *DetectorRegistry) InitializeAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, detector := range r.detectors {
		if err := detector.Initialize(); err != nil {
			return err
		}
	}
	return nil
}

// CleanupAll 清理所有已注册的检测器
func (r *DetectorRegistry) CleanupAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, detector := range r.detectors {
		if err := detector.Cleanup(); err != nil {
			return err
		}
	}
	return nil
}
