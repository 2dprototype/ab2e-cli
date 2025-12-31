## AB2E CLI Tool (Go)

AB2E Scene Tool - Encode, Decode, and Preview Box2D Scenes

```bash
AB2E Scene Tool - Encode, Decode, and Preview Box2D Scenes

Usage:
  ab2e encode -i <input.json> -o <output.bin>
  ab2e decode -i <input.bin> -o <output.json>
  ab2e draw [-t bin|json] <scene_file>
  ab2e <scene_file>
  ab2e config [-reset|-show]
  ab2e version
  ab2e help

Commands:
  encode    Convert JSON scene to binary format
  decode    Convert binary scene to JSON format
  draw      Preview a scene (JSON or binary) in a window
  help      Show this help message

Preview Controls:
  Arrow Keys  - Pan viewport
  +/-         - Zoom in/out
  Mouse Drag  - Pan viewport
  Mouse Wheel - Zoom in/out
  R           - Reset view to default
  F           - Fit view to scene

Examples:
  ab2e encode -i scene.json -o scene.bin
  ab2e decode -i scene.bin -o scene.json
  ab2e draw scene.json
  ab2e draw -t bin scene.bin
  ab2e scene.json
```