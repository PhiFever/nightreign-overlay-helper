package config

import (
	"os"
	"sync"
	"time"
)

// Config 表示应用程序配置
type Config struct {
	// 天数/缩圈相关
	DayPeriodSeconds      []int   `yaml:"day_period_seconds"`
	DeadlyNightrainSeconds int     `yaml:"deadly_nightrain_seconds"`
	ForwardDaySeconds     int     `yaml:"foward_day_seconds"`
	BackDaySeconds        int     `yaml:"back_day_seconds"`
	TimeScale             float64 `yaml:"time_scale"`

	// 更新间隔
	UpdateInterval   float64            `yaml:"update_interval"`
	DetectIntervals  map[string]float64 `yaml:"detect_intervals"`

	// CSS 样式
	DayProgressCSS    string `yaml:"day_progress_css"`
	DayTextCSS        string `yaml:"day_text_css"`
	InRainProgressCSS string `yaml:"in_rain_progress_css"`
	InRainTextCSS     string `yaml:"in_rain_text_css"`
	ArtProgressCSS    string `yaml:"art_progress_css"`
	ArtTextCSS        string `yaml:"art_text_css"`

	// 天数检测器设置
	TemplateStandardHeight int       `yaml:"template_standard_height"`
	MaskLowerWhite         []int     `yaml:"mask_lower_white"`
	MaskUpperWhite         []int     `yaml:"mask_upper_white"`
	ScaleRange             []float64 `yaml:"scale_range"`
	DayxScoreThreshold     float64   `yaml:"dayx_score_threshold"`
	DayxDetectLangs        map[string]string `yaml:"dayx_detect_langs"`

	// HP 检测器设置
	LowerHLSNotInRain []int   `yaml:"lower_hls_not_in_rain"`
	UpperHLSNotInRain []int   `yaml:"upper_hls_not_in_rain"`
	LowerHLSInRain    []int   `yaml:"lower_hls_in_rain"`
	UpperHLSInRain    []int   `yaml:"upper_hls_in_rain"`
	HTolerance        int     `yaml:"h_tolerance"`
	LTolerance        int     `yaml:"l_tolerance"`
	STolerance        int     `yaml:"s_tolerance"`
	HpColorMinAreaRatio float64 `yaml:"hp_color_min_area_ratio"`
	HpColorMaxAreaRatio float64 `yaml:"hp_color_max_area_ratio"`

	// HP 条检测器设置
	HpbarRegionAspectRatio   float64 `yaml:"hpbar_region_aspect_ratio"`
	HpbarDetectStdHeight     int     `yaml:"hpbar_detect_std_height"`
	HpbarBorderVPeakStart    int     `yaml:"hpbar_border_v_peak_start"`
	HpbarBorderVPeakLower    int     `yaml:"hpbar_border_v_peak_lower"`
	HpbarBorderVPeakThreshold int    `yaml:"hpbar_border_v_peak_threshold"`
	HpbarBorderVPeakInterval int     `yaml:"hpbar_border_v_peak_interval"`
	HpbarRecentLengthCount   int     `yaml:"hpbar_recent_length_count"`

	// 地图检测器设置
	FixedMapOverlayDrawSize   []int   `yaml:"fixed_map_overlay_draw_size"`
	MapOverlayDrawSizeRatio   float64 `yaml:"map_overlay_draw_size_ratio"`
	FullMapHoughCircleThres   []int   `yaml:"full_map_hough_circle_thres"`
	FullMapErrorThreshold     float64 `yaml:"full_map_error_threshold"`
	EarthShiftingErrorThreshold float64 `yaml:"earth_shifting_error_threshold"`
	MapPatternMatchInterval   float64 `yaml:"map_pattern_match_interval"`

	// 技能检测器设置
	ArtDetectStandardSize  int       `yaml:"art_detect_standard_size"`
	ArtDetectMatchScales   []float64 `yaml:"art_detect_match_scales"`
	ArtDetectThreshold     float64   `yaml:"art_detect_threshold"`
	ArtDetectDelaySeconds  float64   `yaml:"art_detect_delay_seconds"`
	ArtInfo                map[string]ArtInfo `yaml:"art_info"`

	// Bug 报告
	BugReportEmail string `yaml:"bug_report_email"`
}

// ArtInfo 包含技能能力的信息
type ArtInfo struct {
	Delay    float64 `yaml:"delay"`
	Duration float64 `yaml:"duration"`
	Text     string  `yaml:"text"`
	Color    string  `yaml:"color"`
}

var (
	globalConfig *Config
	configMu     sync.RWMutex
	configPath   = "config.yaml"
	lastModTime  time.Time
)

// Get 返回全局配置，如果文件已被修改则重新加载
func Get() (*Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	info, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	modTime := info.ModTime()
	if globalConfig == nil || modTime != lastModTime {
		cfg, err := Load(configPath)
		if err != nil {
			return nil, err
		}
		globalConfig = cfg
		lastModTime = modTime
	}

	return globalConfig, nil
}

// SetConfigPath 设置配置文件的路径
func SetConfigPath(path string) {
	configMu.Lock()
	defer configMu.Unlock()
	configPath = path
	globalConfig = nil
}
