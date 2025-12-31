// main.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Version = "v0.0.4"
)

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
	case "config":
		configCmd()
	case "version":
		versionCmd()
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

	encodeFile(*input, *output)
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

	decodeFile(*input, *output)
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

	// Auto-detect file type if not specified
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

	launchScenePreview(sceneFile, *fileType)
}

func configCmd() {
	configFlags := flag.NewFlagSet("config", flag.ExitOnError)
	reset := configFlags.Bool("reset", false, "Reset to default configuration")
	show := configFlags.Bool("show", false, "Show current configuration")
	
	if err := configFlags.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *reset {
		config := NewConfig()
		if err := config.Save(); err != nil {
			fmt.Printf("Error resetting config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Configuration reset to defaults")
	} else if *show {
		config := LoadConfig()
		fmt.Println("Current Configuration:")
		fmt.Printf("  Default Input Mode: %s\n", InputMode(config.DefaultInputMode))
		fmt.Printf("  Default Render Style: %s\n", RenderStyle(config.DefaultRenderStyle))
		fmt.Printf("  Default Window Width: %d\n", config.DefaultWindowWidth)
		fmt.Printf("  Default Window Height: %d\n", config.DefaultWindowHeight)
		fmt.Printf("  Default FPS: %d\n", config.DefaultFPS)
		fmt.Printf("  Default Zoom: %.2f\n", config.DefaultZoom)
		fmt.Printf("  Show Joints by Default: %v\n", config.ShowJointsByDefault)
		fmt.Printf("  Pan Sensitivity: %.3f\n", config.PanSensitivity)
		fmt.Printf("  Zoom Step: %.2f\n", config.ZoomStep)
	} else {
		fmt.Println("Configuration command options:")
		fmt.Println("  ab2e config -reset     Reset to default configuration")
		fmt.Println("  ab2e config -show      Show current configuration")
	}
}

func versionCmd() {
	fmt.Println(Version)
}

func printUsage() {
	fmt.Println("AB2E Scene Tool - Encode, Decode, and Preview Box2D Scenes")
	fmt.Println("Version:", Version)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ab2e encode -i <input.json> -o <output.scn>")
	fmt.Println("  ab2e decode -i <input.scn> -o <output.json>")
	fmt.Println("  ab2e draw [-t scn|json] <scene_file>")
	fmt.Println("  ab2e <scene_file>")
	fmt.Println("  ab2e config [-reset|-show]")
	fmt.Println("  ab2e version")
	fmt.Println("  ab2e help")
}