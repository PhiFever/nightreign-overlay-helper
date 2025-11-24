#!/usr/bin/env python3
"""
地图检测器测试脚本
用于测试 Python 版本的地形检测功能，并与 Go 版本对比
"""

import os
import sys
from pathlib import Path

import cv2
import numpy as np

# 直接导入需要的常量和函数
sys.path.insert(0, str(Path(__file__).parent))

# 从 src/detector/map_detector.py 直接导入
from src.common import get_data_path

# 加载地形背景图
CV2_RESIZE_METHOD = cv2.INTER_CUBIC
MAP_BGS = {}
for i in [0, 1, 2, 3, 5]:  # Skip 4
    try:
        path = get_data_path(f"maps/{i}.jpg")
        if not os.path.exists(path):
            # 尝试中文命名
            earth_names = {0: "普通", 1: "雪山", 2: "火山", 3: "腐败", 5: "隐城"}
            path = get_data_path(f"maps/{i}_{earth_names[i]}.jpg")
        img = cv2.cvtColor(cv2.imread(path), cv2.COLOR_BGR2RGB)
        MAP_BGS[i] = img
    except Exception as e:
        print(f"Warning: Could not load earth map {i}: {e}")

# 检测参数
PREDICT_EARTH_SHIFTING_SIZE = (100, 100)
PREDICT_EARTH_SHIFTING_SIZE_REGION = (
    int(PREDICT_EARTH_SHIFTING_SIZE[0] * 0.2),
    int(PREDICT_EARTH_SHIFTING_SIZE[1] * 0.2),
    int(PREDICT_EARTH_SHIFTING_SIZE[0] * 0.6),
    int(PREDICT_EARTH_SHIFTING_SIZE[1] * 0.6),
)
PREDICT_EARTH_SHIFTING_OFFSET_AND_STRIDE = (5, 1)
PREDICT_EARTH_SHIFTING_SCALES = (0.95, 1.05, 7)

def match_earth_shifting(img):
    """地形检测函数（复制自 Python map_detector.py 的实现）"""
    img_resized = cv2.resize(img, PREDICT_EARTH_SHIFTING_SIZE, interpolation=CV2_RESIZE_METHOD)
    x, y, w, h = PREDICT_EARTH_SHIFTING_SIZE_REGION
    img_roi = img_resized[y:y+h, x:x+w].astype(int)

    best_map_id, best_score = None, float('inf')
    offset, stride = PREDICT_EARTH_SHIFTING_OFFSET_AND_STRIDE
    min_scale, max_scale, scale_num = PREDICT_EARTH_SHIFTING_SCALES

    for map_id, map_img in MAP_BGS.items():
        score = float('inf')
        for scale in np.linspace(min_scale, max_scale, scale_num, endpoint=True):
            size = (int(PREDICT_EARTH_SHIFTING_SIZE[0] * scale),
                   int(PREDICT_EARTH_SHIFTING_SIZE[1] * scale))
            map_resized = cv2.resize(map_img, size, interpolation=CV2_RESIZE_METHOD).astype(int)

            for dx in range(-offset, offset+1, stride):
                for dy in range(-offset, offset+1, stride):
                    map_shifted = map_resized[y+dy:y+h+dy, x+dx:x+w+dx]
                    diff = np.abs((img_roi - map_shifted))
                    diff[diff > 100] = 0
                    diff = np.linalg.norm(diff, axis=2)
                    cur_score = np.median(diff)
                    score = min(score, cur_score)

        if score < best_score:
            best_score = score
            best_map_id = map_id

    return best_map_id, best_score

# 地形名称到 ID 的映射
EARTH_SHIFTING_NAME_MAP = {
    "普通": 0,
    "雪山": 1,
    "火山": 2,
    "腐败": 3,
    "隐城": 5,
}

def extract_earth_shifting_from_filename(filename):
    """从文件名提取地形类型"""
    for name, earth_id in EARTH_SHIFTING_NAME_MAP.items():
        if name in filename:
            return earth_id, True
    return -1, False

def is_full_screen_image(filename):
    """判断是否为全屏截图"""
    return "全屏" in filename or "fullscreen" in filename or "full" in filename

def test_earth_shifting_detection():
    """测试地形检测准确率"""
    test_dir = Path("data/test/map_detector")

    if not test_dir.exists():
        print(f"测试目录不存在: {test_dir}")
        return

    # 查找所有测试图片
    image_files = []
    for ext in ["*.png", "*.jpg", "*.jpeg"]:
        image_files.extend(test_dir.glob(ext))

    # 构建已知地形的测试用例
    known_earth_shifting = {}
    for img_file in image_files:
        filename = img_file.name
        earth_id, found = extract_earth_shifting_from_filename(filename)
        if found:
            known_earth_shifting[filename] = earth_id

    if not known_earth_shifting:
        print("没有找到包含地形名称的测试图片")
        return

    print(f"找到 {len(known_earth_shifting)} 个包含已知地形标签的测试图片\n")

    correct_count = 0
    total_count = 0
    earth_stats = {}  # {earth_id: {correct: int, total: int}}

    # 存储所有检测结果以便输出
    results = []

    for filename, expected_earth in sorted(known_earth_shifting.items()):
        file_path = test_dir / filename

        # 加载图片
        img = cv2.imread(str(file_path))
        if img is None:
            print(f"✗ 加载图片失败: {filename}")
            continue

        img = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)

        # 执行检测（使用我们自己的实现）
        try:
            detected_earth, score = match_earth_shifting(img)

            total_count += 1

            # 更新统计
            if expected_earth not in earth_stats:
                earth_stats[expected_earth] = {"correct": 0, "total": 0}
            earth_stats[expected_earth]["total"] += 1

            # 检查是否正确
            is_correct = detected_earth == expected_earth
            if is_correct:
                correct_count += 1
                earth_stats[expected_earth]["correct"] += 1
                status = "✓ 正确"
            else:
                status = "✗ 错误"

            results.append({
                "filename": filename,
                "expected": expected_earth,
                "detected": detected_earth,
                "score": score,
                "correct": is_correct,
            })

            print(f"{status}: {filename}")
            print(f"  期望: {expected_earth}, 检测: {detected_earth}, 分数: {score:.4f}")

        except Exception as e:
            print(f"✗ 检测失败: {filename} - {e}")

    # 输出总体统计
    print("\n" + "="*60)
    print("=== 地形检测准确率统计 (Python 版本) ===")
    print("="*60)

    if total_count > 0:
        accuracy = (correct_count / total_count) * 100
        print(f"\n总体准确率: {accuracy:.1f}% ({correct_count}/{total_count})\n")

        # 各地形准确率
        earth_names = {0: "普通", 1: "雪山", 2: "火山", 3: "腐败", 5: "隐城"}
        for earth_id in sorted(earth_names.keys()):
            if earth_id in earth_stats:
                stats = earth_stats[earth_id]
                if stats["total"] > 0:
                    acc = (stats["correct"] / stats["total"]) * 100
                    print(f"  {earth_names[earth_id]}(ID={earth_id}): {acc:.1f}% "
                          f"({stats['correct']}/{stats['total']})")

    # 输出详细对比表
    print("\n" + "="*60)
    print("详细检测结果:")
    print("="*60)
    print(f"{'文件名':<30} {'期望':<6} {'检测':<6} {'分数':<10} {'结果'}")
    print("-"*60)
    for r in results:
        result_str = "✓" if r["correct"] else "✗"
        print(f"{r['filename']:<30} {r['expected']:<6} {r['detected']:<6} "
              f"{r['score']:<10.4f} {result_str}")

    return {
        "accuracy": accuracy if total_count > 0 else 0,
        "correct": correct_count,
        "total": total_count,
        "earth_stats": earth_stats,
        "results": results,
    }

def test_single_image_detailed(filename):
    """详细测试单个图片，输出所有地形的分数"""
    test_dir = Path("data/test/map_detector")
    file_path = test_dir / filename

    if not file_path.exists():
        print(f"文件不存在: {file_path}")
        return

    # 加载图片
    img = cv2.imread(str(file_path))
    if img is None:
        print(f"加载图片失败: {filename}")
        return

    img = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)

    # 缩放到标准大小
    img_resized = cv2.resize(img, PREDICT_EARTH_SHIFTING_SIZE, interpolation=CV2_RESIZE_METHOD)
    x, y, w, h = PREDICT_EARTH_SHIFTING_SIZE_REGION
    img_roi = img_resized[y:y+h, x:x+w].astype(int)

    # 测试每个地形
    print(f"\n测试图片: {filename}")
    print(f"图片尺寸: {img.shape[1]}x{img.shape[0]}")
    print(f"\n各地形匹配分数:")
    print("-"*40)

    scores = {}
    offset, stride = PREDICT_EARTH_SHIFTING_OFFSET_AND_STRIDE
    min_scale, max_scale, scale_num = PREDICT_EARTH_SHIFTING_SCALES

    for map_id in sorted(MAP_BGS.keys()):
        map_img = MAP_BGS[map_id]
        score = float('inf')

        for scale in np.linspace(min_scale, max_scale, scale_num, endpoint=True):
            size = (int(PREDICT_EARTH_SHIFTING_SIZE[0] * scale),
                   int(PREDICT_EARTH_SHIFTING_SIZE[1] * scale))
            map_resized = cv2.resize(map_img, size, interpolation=CV2_RESIZE_METHOD).astype(int)

            for dx in range(-offset, offset+1, stride):
                for dy in range(-offset, offset+1, stride):
                    map_shifted = map_resized[y+dy:y+h+dy, x+dx:x+w+dx]
                    diff = np.abs((img_roi - map_shifted))
                    diff[diff > 100] = 0
                    diff = np.linalg.norm(diff, axis=2)
                    cur_score = np.median(diff)
                    score = min(score, cur_score)

        scores[map_id] = score
        print(f"  地形 {map_id}: {score:.4f}")

    # 找出最佳匹配
    best_map_id = min(scores, key=scores.get)
    print(f"\n最佳匹配: 地形 {best_map_id} (分数: {scores[best_map_id]:.4f})")

    # 提取期望地形
    expected_earth, has_expected = extract_earth_shifting_from_filename(filename)
    if has_expected:
        print(f"期望地形: {expected_earth}")
        if best_map_id == expected_earth:
            print("✓ 检测正确！")
        else:
            print(f"✗ 检测错误！")

if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="测试 Python 版本的地图检测器")
    parser.add_argument("--detailed", metavar="FILENAME",
                       help="详细测试单个图片")

    args = parser.parse_args()

    if args.detailed:
        test_single_image_detailed(args.detailed)
    else:
        test_earth_shifting_detection()
