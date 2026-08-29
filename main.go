package main

import (
 "archive/tar"
 "compress/gzip"
 "crypto/sha256"
 "encoding/json"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "strings"
 "time"
)

const (
 KosDir       = ".kos"
 ManifestFile = "manifest.json"
 IgnoreFile   = ".kosignore"
)

type Snapshot struct {
 ID        string            `json:"id"`
 Timestamp int64             `json:"timestamp"`
 Message   string            `json:"message"`
 Filename  string            `json:"filename"`
 Files     map[string]string `json:"files"`
}

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
 case "view": // <--- ADD THIS
  cmdView() // <--- ADD THIS
 case "save":
  if len(os.Args) < 3 {
   fmt.Println("Error: missing message. Usage: kos save <message>")
   os.Exit(1)
  }
  // Join arguments in case the user types: kos save "my message"
  message := strings.Join(os.Args[2:], " ")
  cmdSave(message)
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

func cmdSave(message string) {
 // 1. Load manifest
 manifest, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository (or manifest corrupted). Run 'kos init'.\n")
  os.Exit(1)
 }

 // 2. Generate Snapshot ID and Filename
 id := fmt.Sprintf("%x", time.Now().UnixNano())[:8]
 filename := fmt.Sprintf("snap_%s.tar.gz", id)
 filepathTar := filepath.Join(KosDir, filename)

 snapshot := Snapshot{
  ID:        id,
  Timestamp: time.Now().Unix(),
  Message:   message,
  Filename:  filename,
  Files:     make(map[string]string),
 }

 // 3. Load ignore patterns
 ignores := loadIgnores()

 // 4. Create the .tar.gz file (Streaming for low RAM)
 file, err := os.Create(filepathTar)
 if err != nil {
  fmt.Printf("Error creating snapshot file: %v\n", err)
  os.Exit(1)
 }
 defer file.Close()

 gzw := gzip.NewWriter(file)
 defer gzw.Close()
 tw := tar.NewWriter(gzw)
 defer tw.Close()

// 5. Walk directory and stream into tar + hash
 err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
  if err != nil {
   return err
  }

  // Skip the .kos directory itself and the ignore file
  if path == KosDir || path == IgnoreFile {
   if d.IsDir() {
    return filepath.SkipDir
   }
   return nil
  }

  // Check ignore patterns
  for _, pattern := range ignores {
   if pattern == "" {
    continue
   }
   match, _ := filepath.Match(pattern, filepath.Base(path))
   if match {
    if d.IsDir() {
     return filepath.SkipDir
    }
    return nil
   }
  }

  if d.IsDir() {
   return nil // We only process files for the snapshot
  }

  // Calculate SHA256 hash (Memory efficient: streams the file)
  hash, err := getFileHash(path)
  if err != nil {
   fmt.Printf("Warning: Could not hash %s: %v\n", path, err)
   return nil // Continue with other files
  }
  snapshot.Files[path] = hash

  // Add file to tar archive (Memory efficient: streams the file)
  f, err := os.Open(path)
  if err != nil {
   return nil
  }
  defer f.Close()

  info, err := f.Stat()
  if err != nil {
   return nil
  }

  header, err := tar.FileInfoHeader(info, info.Name())
  if err != nil {
   return nil
  }
  header.Name = path // Use relative path

  if err := tw.WriteHeader(header); err != nil {
   return nil
  }
  if _, err := io.Copy(tw, f); err != nil {
   return nil
  }

  return nil
 })

 if err != nil {
  fmt.Printf("Error walking directory: %v\n", err)
  os.Exit(1)
 }

 // 6. Update and save manifest
 manifest.Snapshots = append(manifest.Snapshots, snapshot)
 if err := saveManifest(manifest); err != nil {
  fmt.Printf("Error saving manifest: %v\n", err)
  os.Exit(1)
 }

 fmt.Printf("Saved snapshot %s: %s (%d files)\n", id, message, len(snapshot.Files))
}

// --- Helper Functions ---

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

func loadIgnores() []string {
 data, err := os.ReadFile(IgnoreFile)
 if err != nil {
  return []string{}
 }
 lines := strings.Split(string(data), "\n")
 var ignores []string
 for _, line := range lines {
  trimmed := strings.TrimSpace(line)
  if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
   ignores = append(ignores, trimmed)
  }
 }
 return ignores
}

func getFileHash(path string) (string, error) {
 f, err := os.Open(path)
 if err != nil {
  return "", err
 }
 defer f.Close()

 h := sha256.New()
 // io.Copy streams the file in 32KB chunks. Perfect for 2GB RAM.
 if _, err := io.Copy(h, f); err != nil {
  return "", err
 }
 return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func cmdView() {
 manifest, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
  os.Exit(1)
 }

 // Cinematic Header using ANSI colors
 fmt.Println("\033[1;36m") // Bold Cyan
 fmt.Println("╔══════════════════════════════════════════════════════════╗")
 fmt.Println("║                     KOS TIMELINE                         ║")
 fmt.Println("╚══════════════════════════════════════════════════════════╝")
 fmt.Println("\033[0m") // Reset

 if len(manifest.Snapshots) == 0 {
  fmt.Println("  No snapshots yet. Use 'kos save <message>' to create one.\n")
  return
 }

 // Iterate backwards (newest first)
 for i := len(manifest.Snapshots) - 1; i >= 0; i-- {
  snap := manifest.Snapshots[i]
  
  // Format time beautifully
  t := time.Unix(snap.Timestamp, 0)
  timeStr := t.Format("Jan 02, 15:04")
  
  // ANSI Color Palette
  colorID := "\033[1;33m"    // Bold Yellow
  colorTime := "\033[2;37m"  // Dim White
  colorMsg := "\033[1;37m"   // Bold White
  colorLine := "\033[36m"    // Cyan
  reset := "\033[0m"

  // Print the timeline node
  fmt.Printf("  %s●%s %s%-8s%s ┃ %s%s%s ┃ %s%s%s\n", 
   colorLine, reset,
   colorID, snap.ID, reset,
   colorTime, timeStr, reset,
   colorMsg, snap.Message, reset)
   
  // Print the file count
  fmt.Printf("  %s│%s           ┃  %d files\n", colorLine, reset, len(snap.Files))
  
  // Print the connector or the end of the line
  if i == 0 {
   fmt.Printf("  %s└──%s (End of history)\n\n", colorLine, reset)
  } else {
   fmt.Printf("  %s│%s\n", colorLine, reset)
  }
 }
}
