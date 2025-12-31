// draw.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"strings"
	"math"

	"ab2e/loader"

	"github.com/ByteArena/box2d"
	"github.com/gonutz/wui/v2"
)

const (
	scale          = 30
	panSpeed       = 1.0
	panSensitivity = 0.125
)

// Config holds application configuration
type Config struct {
	DefaultInputMode      int     `json:"default_input_mode"`
	DefaultRenderStyle    int     `json:"default_render_style"`
	DefaultWindowWidth    int     `json:"default_window_width"`
	DefaultWindowHeight   int     `json:"default_window_height"`
	DefaultFPS            int     `json:"default_fps"`
	DefaultZoom           float64 `json:"default_zoom"`
	MinZoom               float64 `json:"min_zoom"`
	MaxZoom               float64 `json:"max_zoom"`
	ShowJointsByDefault   bool    `json:"show_joints_by_default"`
	PanSensitivity        float64 `json:"pan_sensitivity"`
	ZoomStep              float64 `json:"zoom_step"`
}

func NewConfig() *Config {
	return &Config{
		DefaultInputMode:      int(ModePanZoom),
		DefaultRenderStyle:    int(StyleClassic),
		DefaultWindowWidth:    600,
		DefaultWindowHeight:   450,
		DefaultFPS:            60,
		DefaultZoom:           1.0,
		MinZoom:               0.1,
		MaxZoom:               5.0,
		ShowJointsByDefault:   true,
		PanSensitivity:        0.125,
		ZoomStep:              0.1,
	}
}

func configPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "config.json" // safe fallback
	}

	exeDir := filepath.Dir(exePath)
	exeName := filepath.Base(exePath)
	exeBase := strings.TrimSuffix(exeName, filepath.Ext(exeName))

	return filepath.Join(exeDir, exeBase+".json")
}

func LoadConfig() *Config {
	config := NewConfig()

	path := configPath()

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist → save defaults
		_ = config.Save()
		return config
	}

	if err := json.Unmarshal(data, config); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return NewConfig()
	}

	return config
}

func (c *Config) Save() error {
	path := configPath()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}


func encodeFile(input, output string) {
	raw, err := os.ReadFile(input)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	var scene loader.Scene
	if err := json.Unmarshal(raw, &scene); err != nil {
		fmt.Printf("JSON parse error: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := loader.Encode(&scene, f); err != nil {
		fmt.Printf("Encode error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully encoded to %s\n", output)
}

func decodeFile(input, output string) {
	f, err := os.Open(input)
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

	if err := os.WriteFile(output, outBytes, 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully decoded to %s\n", output)
}

func launchScenePreview(sceneFile, fileType string) {
	config := LoadConfig()
	
	var scene *loader.Scene

	if fileType == "bin" {
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
	} else if fileType == "json" {
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

	launchPreview(scene, config)
}

func launchPreview(scene *loader.Scene, config *Config) {
	windowFont, _ := wui.NewFont(wui.FontDesc{
		Name:   "Segoe UI",
		Height: -12,
	})

	window := wui.NewWindow()
	window.SetFont(windowFont)
	window.SetTitle("Box2D Scene Preview")
	window.SetInnerSize(config.DefaultWindowWidth, config.DefaultWindowHeight)
	window.SetResizable(true)

	paintBox := wui.NewPaintBox()
	paintBox.SetBounds(0, 0, config.DefaultWindowWidth, config.DefaultWindowHeight)

	viewport := NewViewport(config.DefaultWindowWidth, config.DefaultWindowHeight)
	viewport.SetZoom(config.DefaultZoom)

	// Box2D World setup
	gravity := scene.World.Gravity
	world := box2d.MakeB2World(box2d.B2Vec2{X: gravity[0], Y: gravity[1]})
	loader.LoadScene(&world, *scene)

	// Interaction State
	iState := &InteractionState{
		Mode:       InputMode(config.DefaultInputMode),
		Style:      RenderStyle(config.DefaultRenderStyle),
		ShowJoints: config.ShowJointsByDefault,
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
		case 74: // J - Toggle Joints
			iState.ShowJoints = !iState.ShowJoints
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
				
				var foundFixture *box2d.B2Fixture
				
				callback := func(fixture *box2d.B2Fixture) bool {
					body := fixture.GetBody()
					if body.GetType() == box2d.B2BodyType.B2_dynamicBody {
						if fixture.TestPoint(worldPoint) {
							foundFixture = fixture
							return false
						}
					}
					return true
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
		if iState.Mode == ModePanZoom && viewport.panning {
			viewport.UpdatePan(x, y)
			mutex.Lock()
			updated = true
			mutex.Unlock()
		}

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
		stepTime := 1.0 / float64(config.DefaultFPS)
		ticker := time.NewTicker(time.Duration(stepTime * float64(time.Second)))
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
// Helper functions (these need to be in main.go or a shared file)
// ----------------------------------------------------------------------------

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
		zoom:         1.0,
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
	config := LoadConfig()
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Max(config.MinZoom, math.Min(config.MaxZoom, z))
}

func (v *Viewport) GetZoom() float64 {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.zoom
}

func (v *Viewport) ZoomIn() {
	config := LoadConfig()
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Min(config.MaxZoom, v.zoom+config.ZoomStep)
}

func (v *Viewport) ZoomOut() {
	config := LoadConfig()
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.zoom = math.Max(config.MinZoom, v.zoom-config.ZoomStep)
}

func (v *Viewport) Pan(dx, dy float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.offsetX += dx / v.zoom
	v.offsetY += dy / v.zoom
}

func (v *Viewport) Reset() {
	config := LoadConfig()
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.offsetX = 0
	v.offsetY = 0
	v.zoom = config.DefaultZoom
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

	config := LoadConfig()
	v.offsetX -= (dx * config.PanSensitivity) / v.zoom
	v.offsetY -= (dy * config.PanSensitivity) / v.zoom

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
	
	screenX = v.windowWidth/2 + int((worldX+v.offsetX)*v.zoom*scale)
	screenY = v.windowHeight/2 + int((worldY+v.offsetY)*v.zoom*scale)
	return
}

func (v *Viewport) ScreenToWorld(screenX, screenY int) (worldX, worldY float64) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	
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
	GroundBody  *box2d.B2Body
	MouseActive bool
	ShowJoints  bool
}

// ----------------------------------------------------------------------------
// Rendering Logic
// ----------------------------------------------------------------------------

func getThemeColors(style RenderStyle) (bg, grid, text, jointColor, mouseJointColor wui.Color) {
	switch style {
	case StyleFlat:
		return wui.RGB(240, 240, 245), wui.RGB(220, 220, 230), wui.RGB(50, 50, 60), 
			wui.RGB(255, 100, 100), wui.RGB(0, 200, 0)
	case StyleClassic:
		return wui.RGB(20, 20, 20), wui.RGB(40, 40, 40), wui.RGB(220, 220, 220),
			wui.RGB(0, 191, 255), wui.RGB(50, 205, 50)
	case StyleNeon:
		return wui.RGB(5, 5, 10), wui.RGB(30, 20, 40), wui.RGB(0, 255, 255),
			wui.RGB(255, 20, 147), wui.RGB(0, 255, 127)
	case StyleBlueprint:
		return wui.RGB(10, 50, 100), wui.RGB(20, 70, 130), wui.RGB(220, 230, 255),
			wui.RGB(255, 215, 0), wui.RGB(144, 238, 144)
	default:
		return wui.RGB(0, 0, 0), wui.RGB(50, 50, 50), wui.RGB(255, 255, 255),
			wui.RGB(100, 100, 100), wui.RGB(0, 200, 0)
	}
}

func clearCanvas(canvas *wui.Canvas, width, height int, style RenderStyle) {
	bg, _, _, _, _ := getThemeColors(style)
	canvas.FillRect(0, 0, width, height, bg)
}

func renderWorld(canvas *wui.Canvas, world *box2d.B2World, viewport *Viewport, width, height int, iState *InteractionState) {
	_, gridColor, _, jointColor, mouseJointColor := getThemeColors(iState.Style)
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
	
	// 3. Joints 
	if iState.ShowJoints {
		for joint := world.GetJointList(); joint != nil; joint = joint.GetNext() {
			if joint == iState.MouseJoint {
				renderMouseJoint(canvas, joint.(*box2d.B2MouseJoint), viewport, mouseJointColor)
			} else {
				renderJoint(canvas, joint, viewport, jointColor, iState.Style)
			}
		}
	}

	// 4. Mouse joint (if not already rendered in joints section)
	if iState.MouseJoint != nil && !iState.ShowJoints {
		renderMouseJoint(canvas, iState.MouseJoint, viewport, mouseJointColor)
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
		} else if shape.GetType() == box2d.B2Shape_Type.E_edge {
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
				outline,
			)
		} else if shape.GetType() == box2d.B2Shape_Type.E_chain {
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
					outline,
				)
			}
		}
	}
}

// renderJoint renders a Box2D joint
func renderJoint(canvas *wui.Canvas, joint box2d.B2JointInterface, viewport *Viewport, color wui.Color, style RenderStyle) {
	b1 := joint.GetBodyA()
	b2 := joint.GetBodyB()
	
	if b1 == nil || b2 == nil {
		return
	}
	
	x1 := b1.GetPosition()
	x2 := b2.GetPosition()
	
	// Get joint type
	jointType := joint.GetType()
	
	// For all joint types, get anchor points
	var p1, p2 box2d.B2Vec2

	switch j := joint.(type) {

	case *box2d.B2RevoluteJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2DistanceJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2PrismaticJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2WeldJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2WheelJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2FrictionJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2MotorJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	case *box2d.B2PulleyJoint:
		p1 = j.GetAnchorA()
		p2 = j.GetAnchorB()

	default:
		// fallback: connect body centers
		p1 = joint.GetBodyA().GetPosition()
		p2 = joint.GetBodyB().GetPosition()
	}

	
	sx1, sy1 := viewport.WorldToScreen(x1.X, x1.Y)
	sx2, sy2 := viewport.WorldToScreen(x2.X, x2.Y)
	sp1x, sp1y := viewport.WorldToScreen(p1.X, p1.Y)
	sp2x, sp2y := viewport.WorldToScreen(p2.X, p2.Y)
	
	switch jointType {
	case box2d.B2JointType.E_revoluteJoint:
		// Revolute joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
		// Draw joint center
		canvas.FillEllipse(sp1x-3, sp1y-3, 6, 6, color)
		
	case box2d.B2JointType.E_distanceJoint:
		// Distance joint: anchor1 -> anchor2
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		
	case box2d.B2JointType.E_prismaticJoint:
		// Prismatic joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_weldJoint:
		// Weld joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_pulleyJoint:
		// Pulley joint
		if pulleyJoint, ok := joint.(*box2d.B2PulleyJoint); ok {
			groundAnchor1 := pulleyJoint.GetGroundAnchorA()
			groundAnchor2 := pulleyJoint.GetGroundAnchorB()
			
			sg1x, sg1y := viewport.WorldToScreen(groundAnchor1.X, groundAnchor1.Y)
			sg2x, sg2y := viewport.WorldToScreen(groundAnchor2.X, groundAnchor2.Y)
			
			// Draw the pulley lines
			canvas.Line(sp1x, sp1y, sg1x, sg1y, color)
			canvas.Line(sp2x, sp2y, sg2x, sg2y, color)
			canvas.Line(sg1x, sg1y, sg2x, sg2y, color)
			
			// Draw ground anchors
			canvas.FillEllipse(sg1x-2, sg1y-2, 4, 4, color)
			canvas.FillEllipse(sg2x-2, sg2y-2, 4, 4, color)
		}
		
	case box2d.B2JointType.E_gearJoint:
		// Gear joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_wheelJoint:
		// Wheel joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_ropeJoint:
		// Rope joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_frictionJoint:
		// Friction joint: body1 -> anchor1 -> anchor2 -> body2
		canvas.Line(sx1, sy1, sp1x, sp1y, color)
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
		canvas.Line(sp2x, sp2y, sx2, sy2, color)
		
	case box2d.B2JointType.E_motorJoint:
		// Motor joint: body1 -> body2
		canvas.Line(sx1, sy1, sx2, sy2, color)
		
	default:
		// Default: draw line between anchors
		canvas.Line(sp1x, sp1y, sp2x, sp2y, color)
	}
}

// renderMouseJoint renders a mouse joint (special case)
func renderMouseJoint(canvas *wui.Canvas, joint *box2d.B2MouseJoint, viewport *Viewport, color wui.Color) {
	target := joint.GetTarget()
	bodyB := joint.GetBodyB()
	if bodyB == nil {
		return
	}
	anchorB := bodyB.GetPosition()

	sx1, sy1 := viewport.WorldToScreen(target.X, target.Y)
	sx2, sy2 := viewport.WorldToScreen(anchorB.X, anchorB.Y)

	canvas.Line(sx1, sy1, sx2, sy2, color)
	canvas.FillEllipse(sx1-3, sy1-3, 6, 6, color)
}

func renderHUD(canvas *wui.Canvas, viewport *Viewport, width, height int, iState *InteractionState) {
	_, _, textColor, _, _ := getThemeColors(iState.Style)
	
	y := 10
	lineH := 20
	
	// Mode Indicator
	modeStr := fmt.Sprintf("MODE: %s (Press 'G')", iState.Mode)
	canvas.TextOut(10, y, modeStr, textColor)
	y += lineH

	// Style Indicator
	styleStr := fmt.Sprintf("STYLE: %s (Press 'T')", iState.Style)
	canvas.TextOut(10, y, styleStr, textColor)
	y += lineH

	// Joints Indicator
	jointsStr := fmt.Sprintf("JOINTS: %v (Press 'J')", iState.ShowJoints)
	canvas.TextOut(10, y, jointsStr, textColor)
	y += lineH * 2

	// Instructions
	strs := []string{
		"CONTROLS:",
		"Arrow Keys : Pan",
		"+ / -      : Zoom",
		"G          : Toggle Grab/Pan",
		"T          : Toggle Theme",
		"J          : Toggle Joints",
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