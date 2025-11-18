# Day Detector Test Screenshots

This directory contains real game screenshots used for testing the Day detection functionality.

## Required Screenshots

Please upload game screenshots with the following filenames:

- `Day1.png` - A screenshot showing "DAY 1" text in the game
- `Day2.png` - A screenshot showing "DAY 2" text in the game
- `Day3.png` - A screenshot showing "DAY 3" text in the game

## Screenshot Requirements

1. **Resolution**: Recommended 1920x1080 or higher
2. **Format**: PNG (preferred for lossless quality)
3. **Content**: The screenshot should clearly show the Day number text as it appears in the game
4. **Language**: Match the language of the templates you're using (chs/cht/eng/jp)

## Running Tests

Once you've uploaded the screenshots, run:

```bash
go test ./internal/detector -v -run TestRealGameScreenshots
```

This will:
- Load each screenshot
- Attempt to detect the Day number using the intelligent detection system
- Report success/failure for each test case
- Provide detailed statistics on detection performance

## Expected Output

When tests pass, you should see output like:

```
=== RUN   TestRealGameScreenshots
=== RUN   TestRealGameScreenshots/Day1.png
    Loaded test image: Day1.png (1920x1080)
    ✅ Detection successful!
       Detected: Day 1
       Expected: Day 1
       Strategy: 2
       Time: 15ms
--- PASS: TestRealGameScreenshots/Day1.png (0.02s)

📊 Test Summary:
   Success Rate: 3/3 (100.0%)

📈 Detection Statistics:
   Total Detections: 3
   Cache Hits: 0
   Color Filter: 2
   Pyramid Search: 1
```

## Troubleshooting

If detection fails:
1. Ensure the screenshot shows the Day text clearly
2. Check that the language matches your template language setting
3. Verify the image quality is good (not too compressed)
4. The text should be visible and not obscured by other UI elements
