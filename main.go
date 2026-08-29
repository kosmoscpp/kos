package main

import (
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
)

const (
 KosDir       = ".kos"
 ManifestFile = "manifest.json"
 IgnoreFile   = ".kosignore"
)

// Snapshot represents a single commit/save point
// NOTE: The json tags MUST be wrapped in backticks (), not single quotes!
type Snapshot struct {
 ID        string            `json:"id"`
 Timestamp int64             `json:"timestamp"`
 Message   string            `json:"message"`
 Filename  string            `json:"filename"`
 Files     map[string]string `json:"files"`
}

// Manifest represents the entire history
type Manifest struct {
 Snapshots []Snapshot `json:"snapshots"`
}

func main() {
 if len(os.Args) < 2 {
  printUsage()
  os.Exit(1)
 }

 command := os.Args[1]

 switch command {
 case "init":
  cmdInit()
 case "ignore":
  if len(os.Args) < 3 {
   fmt.Println("Error: missing pattern. Usage: kos ignore <pattern>")
   os.Exit(1)
  }
  cmdIgnore(os.Args[2])
 default:
  fmt.Printf("Unknown command: %s\n", command)
  printUsage()
  os.Exit(1)
 }
}

func printUsage() {
 fmt.Println("KOS - Zero-Dependency Portable Version Control")
 fmt.Println("Usage: kos <command> [arguments]")
 fmt.Println("\nCommands:")
 fmt.Println("  init              Initialize a new KOS repository")
 fmt.Println("  ignore <pattern>  Add a pattern to .kosignore")
 fmt.Println("  save <message>    Create a new snapshot")
 fmt.Println("  view              View the ASCII timeline")
 fmt.Println("  status            Show modified files")
 fmt.Println("  diff <id1> <id2>  Compare two snapshots")
 fmt.Println("  export <id> <dir> Extract a snapshot")
 fmt.Println("  load <file.tar.gz> Restore from a portable snapshot")
}

func cmdInit() {
 if _, err := os.Stat(KosDir); err == nil {
  fmt.Println("KOS repository already initialized in this directory.")
  return
 }

 if err := os.Mkdir(KosDir, 0755); err != nil {
  fmt.Printf("Error creating %s: %v\n", KosDir, err)
  os.Exit(1)
 }

 manifest := Manifest{Snapshots: []Snapshot{}}
 if err := saveManifest(manifest); err != nil {
  fmt.Printf("Error creating manifest: %v\n", err)
  os.Exit(1)
 }

 if err := os.WriteFile(IgnoreFile, []byte(""), 0644); err != nil {
  fmt.Printf("Error creating %s: %v\n", IgnoreFile, err)
  os.Exit(1)
 }

 fmt.Println("Initialized empty KOS repository in .kos/")
}

func cmdIgnore(pattern string) {
 if _, err := os.Stat(IgnoreFile); os.IsNotExist(err) {
  if err := os.WriteFile(IgnoreFile, []byte(""), 0644); err != nil {
   fmt.Printf("Error creating %s: %v\n", IgnoreFile, err)
   os.Exit(1)
  }
 }

 f, err := os.OpenFile(IgnoreFile, os.O_APPEND|os.O_WRONLY, 0644)
 if err != nil {
  fmt.Printf("Error opening %s: %v\n", IgnoreFile, err)
  os.Exit(1)
 }
 defer f.Close()

 if _, err := f.WriteString(pattern + "\n"); err != nil {
  fmt.Printf("Error writing to %s: %v\n", IgnoreFile, err)
  os.Exit(1)
 }

 fmt.Printf("Added '%s' to %s\n", pattern, IgnoreFile)
}

func saveManifest(m Manifest) error {
 data, err := json.MarshalIndent(m, "", "  ")
 if err != nil {
  return err
 }
 return os.WriteFile(filepath.Join(KosDir, ManifestFile), data, 0644)
}

func loadManifest() (Manifest, error) {
 var m Manifest
 data, err := os.ReadFile(filepath.Join(KosDir, ManifestFile))
 if err != nil {
  return m, err
 }
 err = json.Unmarshal(data, &m)
 return m, err
}
