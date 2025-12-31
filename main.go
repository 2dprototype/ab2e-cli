// main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	panSpeed       = 1.0 // Increased pan speed
	panSensitivity = 0.125
)

// InputMode defines how mouse interaction works
type InputMode int

const (
	ModePanZoom InputMode = iota
	ModeGrab
)

func (m InputMode) String() string {
	switch m {
	case ModePanZoom:
		return "PAN/ZOOM"
	case ModeGrab:
		return "GRAB/PHYSICS"
	default:
		return "UNKNOWN"
	}
}

// RenderStyle defines the visual theme
type RenderStyle int

const (
	StyleFlat RenderStyle = iota
	StyleClassic
	StyleNeon
	StyleBlueprint
)

func (s RenderStyle) String() string {
	switch s {
	case StyleFlat:
		return "Flat"
	case StyleClassic:
		return "Classic Box2D"
	case StyleNeon:
		return "Neon"
	case StyleBlueprint:
		return "Blueprint"
	default:
		return "Unknown"
	}
}

// Viewport state
type Viewport struct {
	offsetX, offsetY       float64
	zoom                   float64
	panning                bool
	lastMouseX, lastMouseY int
	mutex                  sync.RWMutex
	windowWidth            int
	windowHeight           int
}

func NewViewport(width, height int) *Viewport {
	return &Viewport{
		zoom:         defaultZoom,
		windowWidth:  width,
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

	v.offsetX -= (dx * panSensitivity) / v.zoom
	v.offsetY -= (dy * panSensitivity) / v.zoom

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

// ----------------------------------------------------------------------------
// Interaction State
// ----------------------------------------------------------------------------

type InteractionState struct {
	Mode        InputMode
	Style       RenderStyle
	MouseJoint  *box2d.B2MouseJoint
	GroundBody  *box2d.B2Body // Static body for mouse joints
	MouseActive bool
}

// QueryCallback for finding bodies under mouse
type SimpleQueryCallback struct {
	World   *box2d.B2World
	Point   box2d.B2Vec2
	Fixture *box2d.B2Fixture
}

func (cb *SimpleQueryCallback) QueryCallback(proxyId int) bool {
	fixture := cb.World.GetContactManager().
		M_broadPhase.
		GetUserData(proxyId).(*box2d.B2Fixture)

	if fixture.GetBody().GetType() == box2d.B2BodyType.B2_dynamicBody {
		if fixture.TestPoint(cb.Point) {
			cb.Fixture = fixture
			return false // stop query
		}
	}

	return true // continue query
}


func (cb *SimpleQueryCallback) ReportFixture(fixture *box2d.B2Fixture) bool {
	body := fixture.GetBody()
	if body.GetType() == box2d.B2BodyType.B2_dynamicBody {
		if fixture.TestPoint(cb.Point) {
			cb.Fixture = fixture
			// We found a dynamic body, stop the query
			return false
		}
	}
	return true // Keep looking
}

// ----------------------------------------------------------------------------
// Main and Commands
// ----------------------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Check if command is a file (for "ab2e <filename>" syntax)
	if _, err := os.Stat(command); err == nil {
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
		drawFlags.Parse(os.Args[2:])
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

	if *fileType == "" {
		ext := strings.ToLower(filepath.Ext(sceneFile))
		if ext == ".bin" || ext == ".scn" {
			*fileType = "bin"
		} else if ext == ".json" {
			*fileType = "json"
		} else {
			data, err := os.ReadFile(sceneFile)
			if err == nil {
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

	launchPreview(scene)
}

// ----------------------------------------------------------------------------
// Preview Window & Rendering
// ----------------------------------------------------------------------------

func launchPreview(scene *loader.Scene) {
	windowFont, _ := wui.NewFont(wui.FontDesc{
		Name:   "Segoe UI", // Cleaner font
		Height: -12,
	})

	window := wui.NewWindow()
	window.SetFont(windowFont)
	window.SetTitle("Box2D Scene Preview")
	window.SetInnerSize(defaultWidth, defaultHeight)
	window.SetResizable(true)

	paintBox := wui.NewPaintBox()
	paintBox.SetBounds(0, 0, defaultWidth, defaultHeight)

	viewport := NewViewport(defaultWidth, defaultHeight)

	// Box2D World setup
	gravity := scene.World.Gravity
	world := box2d.MakeB2World(box2d.B2Vec2{X: gravity[0], Y: gravity[1]})
	loader.LoadScene(&world, *scene)

	// Interaction State
	iState := &InteractionState{
		Mode:  ModePanZoom,
		Style: StyleFlat,
	}

	// Create a static ground body for the mouse joint
	groundDef := box2d.MakeB2BodyDef()
	iState.GroundBody = world.CreateBody(&groundDef)

	var mutex sync.RWMutex
	var updated bool

	window.SetOnResize(func() {
		mutex.Lock()
		defer mutex.Unlock()

		w, h := window.InnerSize()
		paintBox.SetBounds(0, 0, w, h)
		viewport.SetWindowSize(w, h)
		updated = true
	})

	// --- Input Handling ---

	window.SetOnKeyDown(func(key int) {
		mutex.Lock()
		defer mutex.Unlock()

		switch key {
		case 39: // Right arrow
			viewport.Pan(-panSpeed, 0)
		case 37: // Left arrow
			viewport.Pan(panSpeed, 0)
		case 38: // Up arrow
			viewport.Pan(0, panSpeed)
		case 40: // Down arrow
			viewport.Pan(0, -panSpeed)
		case 187, 107: // +
			viewport.ZoomIn()
		case 189, 109: // -
			viewport.ZoomOut()
		case 82: // R - Reset
			viewport.Reset()
		case 71: // G - Toggle Mode
			if iState.Mode == ModePanZoom {
				iState.Mode = ModeGrab
			} else {
				iState.Mode = ModePanZoom
			}
		case 84: // T - Toggle Theme
			iState.Style = (iState.Style + 1) % 4
		}
		updated = true
	})

	window.SetOnMouseDown(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			if iState.Mode == ModePanZoom {
				viewport.StartPan(x, y)
			} else if iState.Mode == ModeGrab {
				wx, wy := viewport.ScreenToWorld(x, y)
				worldPoint := box2d.MakeB2Vec2(wx, wy)

				aabb := box2d.MakeB2AABB()
				d := 0.001
				aabb.LowerBound.Set(wx-d, wy-d)
				aabb.UpperBound.Set(wx+d, wy+d)

				mutex.Lock()
				
				// Fix: QueryAABB takes a function signature directly in this library
				var foundFixture *box2d.B2Fixture
				
				callback := func(fixture *box2d.B2Fixture) bool {
					body := fixture.GetBody()
					if body.GetType() == box2d.B2BodyType.B2_dynamicBody {
						if fixture.TestPoint(worldPoint) {
							foundFixture = fixture
							return false // Stop query
						}
					}
					return true // Continue query
				}

				world.QueryAABB(callback, aabb)

				if foundFixture != nil {
					body := foundFixture.GetBody()
					md := box2d.MakeB2MouseJointDef()
					md.BodyA = iState.GroundBody
					md.BodyB = body
					md.Target = worldPoint
					md.MaxForce = 1000.0 * body.GetMass()
					md.FrequencyHz = 5.0
					md.DampingRatio = 0.7

					iState.MouseJoint = world.CreateJoint(&md).(*box2d.B2MouseJoint)
					body.SetAwake(true)
					iState.MouseActive = true
				}
				mutex.Unlock()
			}
		}
	})

	paintBox.SetOnMouseMove(func(x, y int) {
		// Panning
		if iState.Mode == ModePanZoom && viewport.panning {
			viewport.UpdatePan(x, y)
			mutex.Lock()
			updated = true
			mutex.Unlock()
		}

		// Dragging Body
		if iState.Mode == ModeGrab && iState.MouseActive && iState.MouseJoint != nil {
			wx, wy := viewport.ScreenToWorld(x, y)
			target := box2d.MakeB2Vec2(wx, wy)

			mutex.Lock()
			iState.MouseJoint.SetTarget(target)
			updated = true
			mutex.Unlock()
		}
	})

	window.SetOnMouseUp(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			if iState.Mode == ModePanZoom {
				viewport.StopPan()
			} else if iState.Mode == ModeGrab {
				// Destroy mouse joint
				mutex.Lock()
				if iState.MouseJoint != nil {
					world.DestroyJoint(iState.MouseJoint)
					iState.MouseJoint = nil
				}
				iState.MouseActive = false
				mutex.Unlock()
			}
		}
	})

	window.SetOnMouseWheel(func(x int, y int, delta float64) {
		if delta > 0 {
			viewport.ZoomIn()
		} else {
			viewport.ZoomOut()
		}
		updated = true
	})

	// --- Rendering Loop ---

	paintBox.SetOnPaint(func(canvas *wui.Canvas) {
		mutex.Lock()
		defer mutex.Unlock()

		if !updated {
			return
		}
		updated = false

		_, _, width, height := paintBox.Bounds()

		clearCanvas(canvas, width, height, iState.Style)
		renderWorld(canvas, &world, viewport, width, height, iState)
		renderHUD(canvas, viewport, width, height, iState)
	})

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

// ----------------------------------------------------------------------------
// Rendering Logic
// ----------------------------------------------------------------------------

func getThemeColors(style RenderStyle) (bg, grid, text wui.Color) {
	switch style {
	case StyleFlat:
		return wui.RGB(240, 240, 245), wui.RGB(220, 220, 230), wui.RGB(50, 50, 60)
	case StyleClassic:
		return wui.RGB(20, 20, 20), wui.RGB(40, 40, 40), wui.RGB(220, 220, 220)
	case StyleNeon:
		return wui.RGB(5, 5, 10), wui.RGB(30, 20, 40), wui.RGB(0, 255, 255)
	case StyleBlueprint:
		return wui.RGB(10, 50, 100), wui.RGB(20, 70, 130), wui.RGB(220, 230, 255)
	default:
		return wui.RGB(0, 0, 0), wui.RGB(50, 50, 50), wui.RGB(255, 255, 255)
	}
}

func clearCanvas(canvas *wui.Canvas, width, height int, style RenderStyle) {
	bg, _, _ := getThemeColors(style)
	canvas.FillRect(0, 0, width, height, bg)
}

func renderWorld(canvas *wui.Canvas, world *box2d.B2World, viewport *Viewport, width, height int, iState *InteractionState) {
	_, gridColor, _ := getThemeColors(iState.Style)
	zoom := viewport.GetZoom()

	// 1. Grid
	gridSizeWorld := 1.0
	gridSizeScreen := int(gridSizeWorld * zoom * scale)

	if gridSizeScreen > 5 {
		worldLeft, worldTop := viewport.ScreenToWorld(0, 0)
		worldRight, worldBottom := viewport.ScreenToWorld(width, height)

		// Y flips in ScreenToWorld, so top > bottom in value effectively for loop
		minY := math.Min(worldTop, worldBottom)
		maxY := math.Max(worldTop, worldBottom)

		// Vertical lines
		startX := math.Floor(worldLeft/gridSizeWorld) * gridSizeWorld
		endX := math.Ceil(worldRight/gridSizeWorld) * gridSizeWorld
		for x := startX; x <= endX; x += gridSizeWorld {
			screenX, _ := viewport.WorldToScreen(x, 0)
			if int(math.Abs(x/gridSizeWorld))%10 == 0 {
				canvas.Line(screenX, 0, screenX, height, wui.RGB(100, 100, 100))
			} else {
				canvas.Line(screenX, 0, screenX, height, gridColor)
			}
		}

		// Horizontal lines
		startY := math.Floor(minY/gridSizeWorld) * gridSizeWorld
		endY := math.Ceil(maxY/gridSizeWorld) * gridSizeWorld
		for y := startY; y <= endY; y += gridSizeWorld {
			_, screenY := viewport.WorldToScreen(0, y)
			if int(math.Abs(y/gridSizeWorld))%10 == 0 {
				canvas.Line(0, screenY, width, screenY, wui.RGB(100, 100, 100))
			} else {
				canvas.Line(0, screenY, width, screenY, gridColor)
			}
		}
	}

	// 2. Bodies
	for body := world.GetBodyList(); body != nil; body = body.GetNext() {
		if body == iState.GroundBody {
			continue
		}
		renderBody(canvas, body, viewport, width, height, iState.Style)
	}

	// 3. Joints (specifically Mouse Joint Visualization)
	if iState.MouseJoint != nil {
		target := iState.MouseJoint.GetTarget()
		bodyB := iState.MouseJoint.GetBodyB()
		anchorB := bodyB.GetPosition()

		sx1, sy1 := viewport.WorldToScreen(target.X, target.Y)
		sx2, sy2 := viewport.WorldToScreen(anchorB.X, anchorB.Y)

		canvas.Line(sx1, sy1, sx2, sy2, wui.RGB(255, 255, 255))
		canvas.FillEllipse(sx1-3, sy1-3, 6, 6, wui.RGB(0, 255, 0))
	}
}

func renderBody(canvas *wui.Canvas, body *box2d.B2Body, viewport *Viewport, width, height int, style RenderStyle) {
	xf := body.GetTransform()

	var fill, outline wui.Color
	isStatic := body.GetType() == box2d.B2BodyType.B2_staticBody
	isAwake := body.IsAwake()

	// Determine Colors based on Style
	switch style {
	case StyleClassic:
		// Box2D C++ testbed colors
		if isStatic {
			fill = wui.RGB(127, 229, 127) // Light Green
		} else if !isAwake {
			fill = wui.RGB(127, 127, 127) // Gray
		} else {
			fill = wui.RGB(229, 178, 178) // Light Red/Pink
		}
		outline = wui.RGB(230, 230, 230) // Almost White outline for everything

	case StyleNeon:
		if isStatic {
			fill = wui.RGB(0, 0, 0)
			outline = wui.RGB(0, 255, 255) // Cyan
		} else {
			fill = wui.RGB(20, 10, 30)
			outline = wui.RGB(255, 0, 255) // Magenta
		}

	case StyleBlueprint:
		if isStatic {
			fill = wui.RGB(10, 50, 100) // Match bg (transparent-ish)
			outline = wui.RGB(255, 255, 255)
		} else {
			fill = wui.RGB(30, 90, 160)
			outline = wui.RGB(200, 220, 255)
		}

	case StyleFlat:
		// Modern flat look
		if isStatic {
			fill = wui.RGB(90, 99, 115) // Slate gray
		} else if !isAwake {
			fill = wui.RGB(180, 180, 180) // Light gray sleep
		} else {
			fill = wui.RGB(235, 87, 87) // Salmon red
		}
		outline = fill // No distinct outline
	}

	for f := body.GetFixtureList(); f != nil; f = f.GetNext() {
		shape := f.GetShape()

		if shape.GetType() == box2d.B2Shape_Type.E_circle {
			circle := shape.(*box2d.B2CircleShape)
			center := box2d.B2TransformVec2Mul(xf, circle.M_p)
			radius := circle.M_radius

			sx, sy := viewport.WorldToScreen(center.X, center.Y)
			sr := int(radius * viewport.zoom * scale)

			if sr < 1 {
				sr = 1
			}

			// Draw Fill
			if style != StyleClassic { // Classic is outline focused usually, but let's fill semi-transparently logic here
				canvas.FillEllipse(sx-sr, sy-sr, sr*2, sr*2, fill)
			}

			// Draw Outline
			if style != StyleFlat {
				canvas.DrawEllipse(sx-sr, sy-sr, sr*2, sr*2, outline)
			}

			// Rotation line
			ax := center.X + radius*xf.Q.C
			ay := center.Y + radius*xf.Q.S
			sAx, sAy := viewport.WorldToScreen(ax, ay)
			canvas.Line(sx, sy, sAx, sAy, wui.RGB(50, 50, 50))

		} else if shape.GetType() == box2d.B2Shape_Type.E_polygon {
			poly := shape.(*box2d.B2PolygonShape)
			vertexCount := poly.M_count
			points := make([]wui.Point, vertexCount)

			for i := 0; i < vertexCount; i++ {
				v := box2d.B2TransformVec2Mul(xf, poly.M_vertices[i])
				sx, sy := viewport.WorldToScreen(v.X, v.Y)
				points[i] = wui.Point{X: int32(sx), Y: int32(sy)}
			}

			if style != StyleClassic {
				canvas.Polygon(points, fill)
			}
			
			// For Classic/Neon/Blueprint we explicitly want the outline drawn
			if style != StyleFlat {
				canvas.Polyline(append(points, points[0]), outline)
			}
		}
	}
}

func renderHUD(canvas *wui.Canvas, viewport *Viewport, width, height int, iState *InteractionState) {
	_, _, textColor := getThemeColors(iState.Style)
	
	y := 10
	lineH := 20
	
	// Mode Indicator
	modeStr := fmt.Sprintf("MODE: %s (Press 'G')", iState.Mode)
	canvas.TextOut(10, y, modeStr, textColor)
	y += lineH

	// Style Indicator
	styleStr := fmt.Sprintf("STYLE: %s (Press 'T')", iState.Style)
	canvas.TextOut(10, y, styleStr, textColor)
	y += lineH * 2

	// Instructions
	strs := []string{
		"CONTROLS:",
		"----------------",
		"Arrow Keys : Pan",
		"+ / -      : Zoom",
		"G          : Toggle Grab/Pan",
		"T          : Toggle Theme",
		"R          : Reset View",
	}

	for _, s := range strs {
		canvas.TextOut(10, y, s, textColor)
		y += lineH
	}

	// Bottom Zoom status
	status := fmt.Sprintf("Zoom: %.2fx", viewport.GetZoom())
	canvas.TextOut(width-100, height-30, status, textColor)
}

func printUsage() {
	fmt.Println("AB2E Scene Tool - Encode, Decode, and Preview Box2D Scenes")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ab2e encode -i <input.json> -o <output.bin>")
	fmt.Println("  ab2e decode -i <input.bin> -o <output.json>")
	fmt.Println("  ab2e draw [-t bin|json] <scene_file>")
	fmt.Println("  ab2e <scene_file>")
	fmt.Println("  ab2e help")
}