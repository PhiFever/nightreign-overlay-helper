package updater

import (
	"context"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/detector"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
)

// DetectorResult 表示检测器的结果
type DetectorResult struct {
	DetectorName string
	Result       interface{}
	Error        error
	Timestamp    time.Time
}

// ScreenCapture 是用于捕获屏幕的函数类型
type ScreenCapture func() (image.Image, error)

// Updater 协调所有检测器并管理检测循环
type Updater struct {
	config   *config.Config
	registry *detector.DetectorRegistry

	// 通道
	resultChan chan DetectorResult
	stopChan   chan struct{}
	doneChan   chan struct{}

	// 屏幕捕获函数
	captureFunc ScreenCapture

	// 状态
	running bool
	mu      sync.RWMutex

	// 统计信息
	updateCount    uint64
	lastUpdateTime time.Time

	// 缓存最后的结果以避免重复日志
	lastResults map[string]string
	resultsMu   sync.Mutex
}

// NewUpdater 创建一个新的更新器
func NewUpdater(cfg *config.Config, registry *detector.DetectorRegistry) *Updater {
	return &Updater{
		config:      cfg,
		registry:    registry,
		resultChan:  make(chan DetectorResult, 100),
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
		captureFunc: mockCapture, // 暂时使用模拟捕获
		lastResults: make(map[string]string),
	}
}

// SetCaptureFunc 设置屏幕捕获函数
func (u *Updater) SetCaptureFunc(fn ScreenCapture) {
	u.captureFunc = fn
}

// Start 启动更新器循环
func (u *Updater) Start(ctx context.Context) error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return fmt.Errorf("updater is already running")
	}
	u.running = true
	u.mu.Unlock()

	logger.Info("[Updater] Starting...")

	// 启动结果处理器
	go u.processResults(ctx)

	// 启动检测循环
	go u.detectionLoop(ctx)

	logger.Info("[Updater] Started successfully")
	return nil
}

// Stop 停止更新器
func (u *Updater) Stop() error {
	u.mu.Lock()
	if !u.running {
		u.mu.Unlock()
		return fmt.Errorf("updater is not running")
	}
	u.running = false
	u.mu.Unlock()

	logger.Info("[Updater] Stopping...")

	// 发送停止信号
	close(u.stopChan)

	// 等待完成
	<-u.doneChan

	logger.Info("[Updater] Stopped successfully")
	return nil
}

// IsRunning 返回更新器是否正在运行
func (u *Updater) IsRunning() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.running
}

// GetResultChan 返回结果通道
func (u *Updater) GetResultChan() <-chan DetectorResult {
	return u.resultChan
}

// detectionLoop 运行主检测循环
func (u *Updater) detectionLoop(ctx context.Context) {
	defer close(u.doneChan)

	interval := time.Duration(u.config.UpdateInterval * float64(time.Second))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Infof("[Updater] Detection loop started (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("[Updater] Context cancelled, stopping detection loop")
			return

		case <-u.stopChan:
			logger.Info("[Updater] Stop signal received, stopping detection loop")
			return

		case <-ticker.C:
			u.runDetection()
		}
	}
}

// runDetection 运行所有启用的检测器
func (u *Updater) runDetection() {
	// 捕获屏幕
	img, err := u.captureFunc()
	if err != nil {
		logger.Errorf("[Updater] Failed to capture screen: %v", err)
		return
	}

	// 获取所有检测器
	detectors := u.registry.GetAll()

	// 并发运行检测器
	var wg sync.WaitGroup
	for _, d := range detectors {
		if !d.IsEnabled() {
			continue
		}

		wg.Add(1)
		go func(det detector.Detector) {
			defer wg.Done()

			result, err := det.Detect(img)

			// 发送结果
			select {
			case u.resultChan <- DetectorResult{
				DetectorName: det.Name(),
				Result:       result,
				Error:        err,
				Timestamp:    time.Now(),
			}:
			default:
				// 通道已满，跳过
				logger.Warningf("[Updater] Result channel full, dropping result from %s", det.Name())
			}
		}(d)
	}

	// 等待所有检测器完成
	wg.Wait()

	// 更新统计信息
	u.updateCount++
	u.lastUpdateTime = time.Now()
}

// processResults 处理检测器结果
func (u *Updater) processResults(ctx context.Context) {
	logger.Info("[Updater] Result processor started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("[Updater] Context cancelled, stopping result processor")
			return

		case <-u.stopChan:
			logger.Info("[Updater] Stop signal received, stopping result processor")
			return

		case result := <-u.resultChan:
			u.handleResult(result)
		}
	}
}

// handleResult 处理单个检测器结果
func (u *Updater) handleResult(result DetectorResult) {
	if result.Error != nil {
		logger.Errorf("[Updater] Detector %s error: %v", result.DetectorName, result.Error)
		return
	}

	// 将结果转换为字符串以进行比较
	resultStr := fmt.Sprintf("%v", result.Result)

	// 检查结果是否已更改
	u.resultsMu.Lock()
	lastResult, exists := u.lastResults[result.DetectorName]
	shouldLog := !exists || lastResult != resultStr
	if shouldLog {
		u.lastResults[result.DetectorName] = resultStr
	}
	u.resultsMu.Unlock()

	// 仅在结果更改时记录日志
	if shouldLog {
		logger.Infof("[Updater] %s: %v", result.DetectorName, result.Result)
	}

	// TODO: 使用结果更新 UI
}

// GetStatistics 返回更新器统计信息
func (u *Updater) GetStatistics() map[string]interface{} {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return map[string]interface{}{
		"running":          u.running,
		"update_count":     u.updateCount,
		"last_update_time": u.lastUpdateTime,
	}
}

// mockCapture 是用于测试的模拟屏幕捕获函数
func mockCapture() (image.Image, error) {
	// 返回一个虚拟的 1x1 图像
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	return img, nil
}
