// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"math"

	"ab2e/loader"
	"github.com/ByteArena/box2d"
	"github.com/gonutz/wui/v2"
)

const (
	scale          = 30
	defaultWidth   = 600
	defaultHeight  = 450
	fps            = 60
	stepTime       = 1.0 / fps
	defaultZoom    = 1.0
	minZoom        = 0.1
	maxZoom        = 5.0
	zoomStep       = 0.1
	panSpeed       = 1.0
)

// Viewport state
type Viewport struct {
	offsetX, offsetY float64
	zoom             float64
	panning          bool
	lastMouseX, lastMouseY int
	mutex            sync.RWMutex
	windowWidth      int
	windowHeight     int
}

func NewViewport(width, height int) *Viewport {
	return &Viewport{
		zoom: defaultZoom,
		windowWidth: width,
		windowHeight: height,
	}
}

func (v *Viewport) SetWindowSize(width, height int) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.windowWidth = width
	v.windowHeight = height
}

func (v *Viewport) SetOffset(x, y float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.offsetX = x
	v.offsetY = y
}

func (v *Viewport) GetOffset() (float64, float64) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.offsetX, v.offsetY
}

func (v *Viewport) SetZoom(z float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Max(minZoom, math.Min(maxZoom, z))
}

func (v *Viewport) GetZoom() float64 {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.zoom
}

func (v *Viewport) ZoomIn() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Min(maxZoom, v.zoom+zoomStep)
}

func (v *Viewport) ZoomOut() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Max(minZoom, v.zoom-zoomStep)
}

func (v *Viewport) Pan(dx, dy float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.offsetX += dx / v.zoom
	v.offsetY += dy / v.zoom
}

func (v *Viewport) Reset() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.offsetX = 0
	v.offsetY = 0
	v.zoom = defaultZoom
}


func (v *Viewport) StartPan(x, y int) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.panning = true
	v.lastMouseX = x
	v.lastMouseY = y
}

func (v *Viewport) UpdatePan(x, y int) {
	if !v.panning {
		return
	}
	
	v.mutex.Lock()
	defer v.mutex.Unlock()
	
	dx := float64(x - v.lastMouseX)
	dy := float64(y - v.lastMouseY)
	
	v.offsetX -= dx / v.zoom
	v.offsetY -= dy / v.zoom
	
	v.lastMouseX = x
	v.lastMouseY = y
}

func (v *Viewport) StopPan() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.panning = false
}

func (v *Viewport) WorldToScreen(worldX, worldY float64) (screenX, screenY int) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	// Apply viewport transform: (world + offset) * zoom * scale + center
	screenX = v.windowWidth/2 + int((worldX+v.offsetX)*v.zoom*scale)
	screenY = v.windowHeight/2 + int((worldY+v.offsetY)*v.zoom*scale)
	return
}

func (v *Viewport) ScreenToWorld(screenX, screenY int) (worldX, worldY float64) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
	// Inverse transform: (screen - center) / (zoom * scale) - offset
	worldX = (float64(screenX-v.windowWidth/2)/(v.zoom*scale)) - v.offsetX
	worldY = (float64(screenY-v.windowHeight/2)/(v.zoom*scale)) - v.offsetY
	return
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	// Check if command is a file (for "ab2e <filename>" syntax)
	if _, err := os.Stat(command); err == nil {
		// It's a file, treat as draw command
		os.Args = append([]string{os.Args[0], "draw"}, os.Args[1:]...)
		command = "draw"
	}
	
	switch command {
	case "encode":
		encodeCmd()
	case "decode":
		decodeCmd()
	case "draw", "preview":
		drawCmd()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func encodeCmd() {
	encodeFlags := flag.NewFlagSet("encode", flag.ExitOnError)
	input := encodeFlags.String("i", "", "Input JSON file")
	output := encodeFlags.String("o", "", "Output binary file")
	
	if err := encodeFlags.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *input == "" || *output == "" {
		fmt.Println("Error: Both -i and -o flags are required")
		encodeFlags.Usage()
		os.Exit(1)
	}

	// Read JSON
	raw, err := os.ReadFile(*input)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}
	
	var scene loader.Scene
	if err := json.Unmarshal(raw, &scene); err != nil {
		fmt.Printf("JSON parse error: %v\n", err)
		os.Exit(1)
	}

	// Write Binary
	f, err := os.Create(*output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := loader.Encode(&scene, f); err != nil {
		fmt.Printf("Encode error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully encoded to %s\n", *output)
}

func decodeCmd() {
	decodeFlags := flag.NewFlagSet("decode", flag.ExitOnError)
	input := decodeFlags.String("i", "", "Input binary file")
	output := decodeFlags.String("o", "", "Output JSON file")
	
	if err := decodeFlags.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *input == "" || *output == "" {
		fmt.Println("Error: Both -i and -o flags are required")
		decodeFlags.Usage()
		os.Exit(1)
	}

	// Read Binary
	f, err := os.Open(*input)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	scene, err := loader.Decode(f)
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		os.Exit(1)
	}

	// Write JSON
	outBytes, err := json.MarshalIndent(scene, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.WriteFile(*output, outBytes, 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully decoded to %s\n", *output)
}

func drawCmd() {
	drawFlags := flag.NewFlagSet("draw", flag.ExitOnError)
	fileType := drawFlags.String("t", "", "File type: bin or json (default: autodetect)")
	sceneFile := ""
	
	if len(os.Args) > 2 {
		// Parse flags first
		drawFlags.Parse(os.Args[2:])
		
		// Get non-flag arguments (the scene file)
		args := drawFlags.Args()
		if len(args) > 0 {
			sceneFile = args[0]
		}
	}

	if sceneFile == "" {
		fmt.Println("Error: Scene file path is required")
		fmt.Println("Usage: ab2e draw [-t bin|json] <scene_file>")
		os.Exit(1)
	}

	// Auto-detect file type if not specified
	if *fileType == "" {
		ext := strings.ToLower(filepath.Ext(sceneFile))
		if ext == ".bin" || ext == ".scn" {
			*fileType = "bin"
		} else if ext == ".json" {
			*fileType = "json"
		} else {
			// Try to detect by content
			data, err := os.ReadFile(sceneFile)
			if err == nil {
				// Check if it looks like JSON
				str := string(data)
				if len(str) > 0 && (str[0] == '{' || str[0] == '[') {
					*fileType = "json"
				} else {
					fmt.Println("Error: Cannot auto-detect file type. Please specify with -t flag")
					os.Exit(1)
				}
			} else {
				fmt.Println("Error: Cannot auto-detect file type. Please specify with -t flag")
				os.Exit(1)
			}
		}
	}

	// Load scene
	var scene *loader.Scene
	
	if *fileType == "bin" {
		f, err := os.Open(sceneFile)
		if err != nil {
			fmt.Printf("Error reading binary file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		
		scene, err = loader.Decode(f)
		if err != nil {
			fmt.Printf("Error decoding binary: %v\n", err)
			os.Exit(1)
		}
	} else if *fileType == "json" {
		raw, err := os.ReadFile(sceneFile)
		if err != nil {
			fmt.Printf("Error reading JSON file: %v\n", err)
			os.Exit(1)
		}
		
		scene, err = loader.DecodeScene(string(raw))
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Error: Invalid file type. Use 'bin' or 'json'")
		os.Exit(1)
	}

	// Launch preview window
	launchPreview(scene)
}

func launchPreview(scene *loader.Scene) {
	windowFont, _ := wui.NewFont(wui.FontDesc{
		Name:   "Tahoma",
		Height: -11,
	})

	window := wui.NewWindow()
	window.SetFont(windowFont)
	window.SetTitle("Box2D Scene Preview - Use Arrow Keys to Pan, +/- to Zoom, Mouse Drag to Pan, R to Reset, F to Fit View")
	window.SetInnerSize(defaultWidth, defaultHeight)
	window.SetResizable(true)

	paintBox := wui.NewPaintBox()
	paintBox.SetBounds(0, 0, defaultWidth, defaultHeight)

	// Create viewport
	viewport := NewViewport(defaultWidth, defaultHeight)

	// Create a new Box2D world using scene's gravity
	gravity := scene.World.Gravity
	world := box2d.MakeB2World(box2d.B2Vec2{X: gravity[0], Y: gravity[1]})

	// Load scene into Box2D
	loader.LoadScene(&world, *scene)

	var mutex sync.RWMutex
	var updated bool

	// Handle window resize
	window.SetOnResize(func() {
		mutex.Lock()
		defer mutex.Unlock()
		
		w, h := window.InnerSize()
		paintBox.SetBounds(0, 0, w, h)
		viewport.SetWindowSize(w, h)
		updated = true
	})

	// Handle keyboard input
	window.SetOnKeyDown(func(key int) {
		switch key {
		case 39: // Right arrow
			viewport.Pan(-panSpeed, 0)
			updated = true
		case 37: // Left arrow
			viewport.Pan(panSpeed, 0)
			updated = true
		case 38: // Up arrow
			viewport.Pan(0, panSpeed)
			updated = true
		case 40: // Down arrow
			viewport.Pan(0, -panSpeed)
			updated = true
		case 187: // +
			viewport.ZoomIn()
			updated = true
		case 189: // -
			viewport.ZoomOut()
			updated = true
		case 82: // R
			viewport.Reset()
			updated = true
		}
	})

	// Handle mouse input for panning
	window.SetOnMouseDown(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			viewport.StartPan(x, y)
		}
	})

	paintBox.SetOnMouseMove(func(x, y int) {
		if viewport.panning {
			viewport.UpdatePan(x, y)
			mutex.Lock()
			updated = true
			mutex.Unlock()
		}
	})

	window.SetOnMouseUp(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			viewport.StopPan()
		}
	})

	// Handle mouse wheel for zoom
	window.SetOnMouseWheel(func(x int, y int, delta float64) {
		if delta > 0 {
			viewport.ZoomIn()
		} else {
			viewport.ZoomOut()
		}
		updated = true
	})

	paintBox.SetOnPaint(func(canvas *wui.Canvas) {
		mutex.Lock()
		defer mutex.Unlock()

		if !updated {
			return
		}
		updated = false

		_, _, width, height := paintBox.Bounds()
		
		clearCanvas(canvas, width, height)
		renderWorld(canvas, &world, viewport, width, height)
		renderHUD(canvas, viewport, width, height)
	})

	// Start physics simulation in goroutine
	go func() {
		ticker := time.NewTicker(time.Second / fps)
		defer ticker.Stop()

		for range ticker.C {
			mutex.Lock()
			world.Step(stepTime, 8, 3)
			updated = true
			paintBox.Paint()
			mutex.Unlock()
		}
	}()

	window.Add(paintBox)
	window.Show()
}

func clearCanvas(canvas *wui.Canvas, width, height int) {
	canvas.FillRect(0, 0, width, height, wui.RGB(20, 20, 30))
}

func renderWorld(canvas *wui.Canvas, world *box2d.B2World, viewport *Viewport, width, height int) {
	zoom := viewport.GetZoom()
	
	// Draw coordinate grid
	gridSizeWorld := 1.0 // 1 meter in world coordinates
	gridSizeScreen := int(gridSizeWorld * zoom * scale)
	gridColor := wui.RGB(40, 40, 60)
	
	if gridSizeScreen > 5 { // Only draw grid if it's visible
		
		// Find visible range
		worldLeft, worldTop := viewport.ScreenToWorld(0, 0)
		worldRight, worldBottom := viewport.ScreenToWorld(width, height)
		
		// Vertical lines
		startX := math.Floor(worldLeft/gridSizeWorld) * gridSizeWorld
		endX := math.Ceil(worldRight/gridSizeWorld) * gridSizeWorld
		
		for x := startX; x <= endX; x += gridSizeWorld {
			screenX1, screenY1 := viewport.WorldToScreen(x, worldTop)
			screenX2, screenY2 := viewport.WorldToScreen(x, worldBottom)
			
			// Make every 5th line brighter
			lineColor := gridColor
			if int(math.Abs(x/gridSizeWorld))%5 == 0 {
				lineColor = wui.RGB(60, 60, 80)
			}
			
			canvas.Line(screenX1, screenY1, screenX2, screenY2, lineColor)
		}
		
		// Horizontal lines
		startY := math.Floor(worldTop/gridSizeWorld) * gridSizeWorld
		endY := math.Ceil(worldBottom/gridSizeWorld) * gridSizeWorld
		
		for y := startY; y <= endY; y += gridSizeWorld {
			screenX1, screenY1 := viewport.WorldToScreen(worldLeft, y)
			screenX2, screenY2 := viewport.WorldToScreen(worldRight, y)
			
			// Make every 5th line brighter
			lineColor := gridColor
			if int(math.Abs(y/gridSizeWorld))%5 == 0 {
				lineColor = wui.RGB(60, 60, 80)
			}
			
			canvas.Line(screenX1, screenY1, screenX2, screenY2, lineColor)
		}
	}

	// Draw origin (0,0)
	originX, originY := viewport.WorldToScreen(0, 0)
	if originX >= 0 && originX < width && originY >= 0 && originY < height {
		canvas.FillEllipse(originX-3, originY-3, 6, 6, wui.RGB(255, 100, 100))
		canvas.TextOut(originX+8, originY-6, "(0,0)", wui.RGB(255, 150, 150))
	}
	
	// Draw bodies
	bodyList := world.GetBodyList()
	bodyColors := []wui.Color{
		wui.RGB(255, 100, 100),   // Red
		wui.RGB(100, 255, 100),   // Green
		wui.RGB(100, 100, 255),   // Blue
		wui.RGB(255, 255, 100),   // Yellow
		wui.RGB(255, 100, 255),   // Magenta
		wui.RGB(100, 255, 255),   // Cyan
	}

	bodyIdx := 0
	for body := bodyList; body != nil; body = body.GetNext() {
		color := bodyColors[bodyIdx%len(bodyColors)]
		bodyType := body.GetType()
		
		// Different colors for different body types
		if bodyType == box2d.B2BodyType.B2_staticBody {
			color = wui.RGB(150, 150, 150) // Gray for static
		} else if bodyType == box2d.B2BodyType.B2_kinematicBody {
			color = wui.RGB(200, 150, 50) // Orange for kinematic
		}

		for fixture := body.GetFixtureList(); fixture != nil; fixture = fixture.GetNext() {
			shape := fixture.GetShape()
			shapeType := shape.GetType()

			if shapeType == box2d.B2Shape_Type.E_circle {
				// Circle shape
				position := body.GetPosition()
				radius := shape.GetRadius()
				screenX, screenY := viewport.WorldToScreen(position.X, position.Y)
				screenRadius := int(radius * zoom * scale)
				
				if screenRadius > 1 { // Only draw if visible
					canvas.DrawEllipse(
						screenX-screenRadius,
						screenY-screenRadius,
						screenRadius*2,
						screenRadius*2,
						color,
					)
					
					// Draw rotation indicator
					angle := body.GetAngle()
					indicatorX := screenX + int(float64(screenRadius)*0.7*math.Cos(angle))
					indicatorY := screenY + int(float64(screenRadius)*0.7*math.Sin(angle))
					canvas.FillEllipse(indicatorX-2, indicatorY-2, 4, 4, wui.RGB(255, 255, 255))
				}
				
			} else if shapeType == box2d.B2Shape_Type.E_polygon {
				// Polygon shape
				polygonShape := shape.(*box2d.B2PolygonShape)
				points := make([]wui.Point, polygonShape.M_count)
				visible := false
				
				for i := 0; i < polygonShape.M_count; i++ {
					vertex := body.GetWorldPoint(polygonShape.M_vertices[i])
					screenX, screenY := viewport.WorldToScreen(vertex.X, vertex.Y)
					points[i] = wui.Point{
						X: int32(screenX),
						Y: int32(screenY),
					}
					
					// Check if at least one vertex is visible
					if screenX >= 0 && screenX < width && screenY >= 0 && screenY < height {
						visible = true
					}
				}
				
				if visible {
					canvas.Polyline(append(points, points[0]), color)
					
					// Draw center point
					position := body.GetPosition()
					centerX, centerY := viewport.WorldToScreen(position.X, position.Y)
					if centerX >= 0 && centerX < width && centerY >= 0 && centerY < height {
						canvas.FillEllipse(centerX-2, centerY-2, 4, 4, wui.RGB(255, 255, 255))
					}
				}
				
			} else if shapeType == box2d.B2Shape_Type.E_edge {
				// Edge shape (line segment)
				edgeShape := shape.(*box2d.B2EdgeShape)
				v1 := body.GetWorldPoint(edgeShape.M_vertex1)
				v2 := body.GetWorldPoint(edgeShape.M_vertex2)
				
				screenX1, screenY1 := viewport.WorldToScreen(v1.X, v1.Y)
				screenX2, screenY2 := viewport.WorldToScreen(v2.X, v2.Y)
				
				canvas.Line(
					screenX1,
					screenY1,
					screenX2,
					screenY2,
					wui.RGB(200, 200, 100), // Yellow for edges
				)
			} else if shapeType == box2d.B2Shape_Type.E_chain {
				// Chain shape
				chainShape := shape.(*box2d.B2ChainShape)
				vertices := chainShape.M_vertices
				count := chainShape.M_count
				
				for i := 0; i < count-1; i++ {
					v1 := body.GetWorldPoint(vertices[i])
					v2 := body.GetWorldPoint(vertices[i+1])
					
					screenX1, screenY1 := viewport.WorldToScreen(v1.X, v1.Y)
					screenX2, screenY2 := viewport.WorldToScreen(v2.X, v2.Y)
					
					canvas.Line(
						screenX1,
						screenY1,
						screenX2,
						screenY2,
						wui.RGB(100, 200, 200), // Cyan for chains
					)
				}
			}
		}
		
		// Draw velocity vector for dynamic bodies
		if body.GetType() == box2d.B2BodyType.B2_dynamicBody {
			position := body.GetPosition()
			velocity := body.GetLinearVelocity()
			
			if velocity.Length() > 0.1 {
				startX, startY := viewport.WorldToScreen(position.X, position.Y)
				endX, endY := viewport.WorldToScreen(
					position.X+velocity.X*0.2, // Scale for visualization
					position.Y+velocity.Y*0.2,
				)
				
				canvas.Line(startX, startY, endX, endY, wui.RGB(255, 100, 100))
				canvas.FillEllipse(endX-3, endY-3, 6, 6, wui.RGB(255, 50, 50))
			}
		}
		
		bodyIdx++
	}
}

func renderHUD(canvas *wui.Canvas, viewport *Viewport, width, height int) {
	zoom := viewport.GetZoom()
	offsetX, offsetY := viewport.GetOffset()
	
	// Draw viewport info
	info := fmt.Sprintf("Zoom: %.2fx\nOffset: (%.2f, %.2f)\n\nControls:\nArrow Keys => Pan\n+/- => Zoom\nMouse Drag => Pan\nMouse Wheel => Zoom\nR => Reset View\nF => Fit View to Scene",
		zoom, offsetX, offsetY)
	
	lines := strings.Split(info, "\n")
	for i, line := range lines {
		canvas.TextOut(10, 10+i*15, line, wui.RGB(220, 220, 220))
	}
	
	// Draw zoom indicator in bottom right
	zoomBarWidth := 100
	zoomBarHeight := 10
	zoomBarX := width - zoomBarWidth - 10
	zoomBarY := height - 30
	
	// Background
	canvas.FillRect(zoomBarX, zoomBarY, zoomBarWidth, zoomBarHeight, wui.RGB(80, 80, 80))
	
	// Fill based on zoom level (logarithmic scale for better visualization)
	zoomRatio := (math.Log10(zoom) - math.Log10(minZoom)) / (math.Log10(maxZoom) - math.Log10(minZoom))
	fillWidth := int(float64(zoomBarWidth) * math.Max(0, math.Min(1, zoomRatio)))
	canvas.FillRect(zoomBarX, zoomBarY, fillWidth, zoomBarHeight, wui.RGB(100, 200, 100))
	
	// Zoom labels
	canvas.TextOut(zoomBarX, zoomBarY-15, fmt.Sprintf("Zoom: %.2fx", zoom), wui.RGB(220, 220, 220))
	canvas.TextOut(zoomBarX, zoomBarY+15, fmt.Sprintf("Min: %.1fx", minZoom), wui.RGB(150, 150, 150))
	canvas.TextOut(zoomBarX+zoomBarWidth-45, zoomBarY+15, fmt.Sprintf("Max: %.1fx", maxZoom), wui.RGB(150, 150, 150))
}

func printUsage() {
	fmt.Println("AB2E Scene Tool - Encode, Decode, and Preview Box2D Scenes")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ab2e encode -i <input.json> -o <output.bin>")
	fmt.Println("  ab2e decode -i <input.bin> -o <output.json>")
	fmt.Println("  ab2e draw [-t bin|json] <scene_file>")
	fmt.Println("  ab2e <scene_file>             (auto-detect format)")
	fmt.Println("  ab2e help")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  encode    Convert JSON scene to binary format")
	fmt.Println("  decode    Convert binary scene to JSON format")
	fmt.Println("  draw      Preview a scene (JSON or binary) in a window")
	fmt.Println("  help      Show this help message")
	fmt.Println()
	fmt.Println("Preview Controls:")
	fmt.Println("  Arrow Keys  - Pan viewport")
	fmt.Println("  +/-         - Zoom in/out")
	fmt.Println("  Mouse Drag  - Pan viewport")
	fmt.Println("  Mouse Wheel - Zoom in/out")
	fmt.Println("  R           - Reset view to default")
	fmt.Println("  F           - Fit view to scene")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ab2e encode -i scene.json -o scene.bin")
	fmt.Println("  ab2e decode -i scene.bin -o scene.json")
	fmt.Println("  ab2e draw scene.json")
	fmt.Println("  ab2e draw -t bin scene.bin")
	fmt.Println("  ab2e scene.json")
}