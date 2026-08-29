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
 case "save":
  if len(os.Args) < 3 {
   fmt.Println("Error: missing message. Usage: kos save <message>")
   os.Exit(1)
  }
  message := strings.Join(os.Args[2:], " ")
  cmdSave(message)
 case "view":
  cmdView()
 case "status":
  cmdStatus()
 case "diff":
  if len(os.Args) < 4 {
   fmt.Println("Error: missing IDs. Usage: kos diff <id1> <id2>")
   os.Exit(1)
  }
  cmdDiff(os.Args[2], os.Args[3])
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
 manifest, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
  os.Exit(1)
 }

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

 // REFACTORED: Use the helper to get files and hashes
 currentFiles, err := getWorkingDirectoryFiles()
 if err != nil {
  fmt.Printf("Error scanning directory: %v\n", err)
  os.Exit(1)
 }
 snapshot.Files = currentFiles

 // Create the .tar.gz file
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
 
  // Stream files into tar
  for path := range currentFiles {
   f, err := os.Open(path)
   if err != nil {
    continue
   }
   defer f.Close()
 
   info, err := f.Stat()
   if err != nil {
    continue
   }
 
   header, err := tar.FileInfoHeader(info, info.Name())
   if err != nil {
    continue
   }
   header.Name = path
 
   if err := tw.WriteHeader(header); err != nil {
    continue
   }
   if _, err := io.Copy(tw, f); err != nil {
    continue
   }
  }
 
  manifest.Snapshots = append(manifest.Snapshots, snapshot)
  if err := saveManifest(manifest); err != nil {
   fmt.Printf("Error saving manifest: %v\n", err)
   os.Exit(1)
  }
 
  fmt.Printf("Saved snapshot %s: %s (%d files)\n", id, message, len(snapshot.Files))
 }
 
 func cmdView() {
  manifest, err := loadManifest()
  if err != nil {
   fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
   os.Exit(1)
  }
 
  fmt.Println("\033[1;36m")
  fmt.Println("╔══════════════════════════════════════════════════════════╗")
  fmt.Println("║                     KOS TIMELINE                       ║")
  fmt.Println("╚══════════════════════════════════════════════════════════╝")
  fmt.Println("\033[0m")
 
  if len(manifest.Snapshots) == 0 {
   fmt.Println("  No snapshots yet. Use 'kos save <message>' to create one.\n")
   return
  }
 
  for i := len(manifest.Snapshots) - 1; i >= 0; i-- {
   snap := manifest.Snapshots[i]
   t := time.Unix(snap.Timestamp, 0)
   timeStr := t.Format("Jan 02, 15:04")
   
   colorID := "\033[1;33m"
   colorTime := "\033[2;37m"
   colorMsg := "\033[1;37m"
   colorLine := "\033[36m"
   reset := "\033[0m"
 
   fmt.Printf("  %s●%s %s%-8s%s ┃ %s%s%s ┃ %s%s%s\n", 
    colorLine, reset, colorID, snap.ID, reset, colorTime, timeStr, reset, colorMsg, snap.Message, reset)
   fmt.Printf("  %s│%s           ┃  %d files\n", colorLine, reset, len(snap.Files))
   
   if i == 0 {
    fmt.Printf("  %s└──%s (End of history)\n\n", colorLine, reset)
   } else {
    fmt.Printf("  %s│%s\n", colorLine, reset)
   }
  }
 }
 
 func cmdStatus() {
  manifest, err := loadManifest()
  if err != nil {
   fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
   os.Exit(1)
  }
 
  currentFiles, err := getWorkingDirectoryFiles()
  if err != nil {
   fmt.Printf("Error scanning directory: %v\n", err)
   os.Exit(1)
  }
 
  var latestFiles map[string]string
  if len(manifest.Snapshots) > 0 {
   latestFiles = manifest.Snapshots[len(manifest.Snapshots)-1].Files
  } else {
   latestFiles = make(map[string]string)
  }
 
  var added, modified, deleted []string
 
  for path, currentHash := range currentFiles {
   prevHash, exists := latestFiles[path]
   if !exists {
    added = append(added, path)
   } else if prevHash != currentHash {
    modified = append(modified, path)
   }
  }
 
  for path := range latestFiles {
   if _, exists := currentFiles[path]; !exists {
    deleted = append(deleted, path)
   }
  }
 
  fmt.Println("\n\033[1;37m--- KOS STATUS ---\033[0m")
  if len(added) == 0 && len(modified) == 0 && len(deleted) == 0 {
   fmt.Println("\033[32mNothing to save. Working tree clean.\033[0m\n")
   return
  }
 
  if len(added) > 0 {
   fmt.Printf("\033[32m+ Added (%d):\033[0m\n", len(added))
   for _, f := range added { fmt.Printf("  %s\n", f) }
  }
  if len(modified) > 0 {
   fmt.Printf("\033[33m~ Modified (%d):\033[0m\n", len(modified))
   for _, f := range modified { fmt.Printf("  %s\n", f) }
  }
  if len(deleted) > 0 {
   fmt.Printf("\033[31m- Deleted (%d):\033[0m\n", len(deleted))
   for _, f := range deleted { fmt.Printf("  %s\n", f) }
  }
  fmt.Println()
 }
 
 func cmdDiff(id1, id2 string) {
  manifest, err := loadManifest()
  if err != nil {
   fmt.Printf("Error: Not a KOS repository.\n")
   os.Exit(1)
  }
 
  var snap1, snap2 *Snapshot
  for i := range manifest.Snapshots {
   if manifest.Snapshots[i].ID == id1 { snap1 = &manifest.Snapshots[i] }
   if manifest.Snapshots[i].ID == id2 { snap2 = &manifest.Snapshots[i] }
  }
 
  if snap1 == nil || snap2 == nil {
   fmt.Printf("Error: One or both snapshot IDs not found.\n")
   os.Exit(1)
  }
 
  var added, deleted, modified []string

  for path, hash2 := range snap2.Files {
    hash1, exists := snap1.Files[path]
    if !exists {
     added = append(added, path)
    } else if hash1 != hash2 {
     modified = append(modified, path)
    }
   }
  
   for path := range snap1.Files {
    if _, exists := snap2.Files[path]; !exists {
     deleted = append(deleted, path)
    }
   }
  
   fmt.Printf("\n\033[1;36m--- DIFF: %s vs %s ---\033[0m\n", id1, id2)
   if len(added) == 0 && len(deleted) == 0 && len(modified) == 0 {
    fmt.Println("\033[32mNo differences found.\033[0m\n")
    return
   }
  
   if len(added) > 0 {
    fmt.Printf("\033[32m+ Added (%d):\033[0m\n", len(added))
    for _, f := range added { fmt.Printf("  %s\n", f) }
   }
   if len(deleted) > 0 {
    fmt.Printf("\033[31m- Deleted (%d):\033[0m\n", len(deleted))
    for _, f := range deleted { fmt.Printf("  %s\n", f) }
   }
   if len(modified) > 0 {
    fmt.Printf("\033[33m~ Modified (%d):\033[0m\n", len(modified))
    for _, f := range modified { fmt.Printf("  %s\n", f) }
   }
   fmt.Println()
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
   if _, err := io.Copy(h, f); err != nil {
    return "", err
   }
   return fmt.Sprintf("%x", h.Sum(nil)), nil
  }
  
  // REFACTORED HELPER: Walks directory, respects ignores, and hashes files.
  // Used by both 'save' and 'status' to keep code DRY (Don't Repeat Yourself).
  func getWorkingDirectoryFiles() (map[string]string, error) {
   files := make(map[string]string)
   ignores := loadIgnores()
  
   err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
    if err != nil {
     return err
    }
  
    if path == KosDir || path == IgnoreFile {
     if d.IsDir() {
      return filepath.SkipDir
     }
     return nil
    }
  
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
     return nil
    }
  
    hash, err := getFileHash(path)
    if err != nil {
     return nil
    }
    files[path] = hash
    return nil
   })
  
   return files, err
  }
