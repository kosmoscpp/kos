package main

import (
 "archive/tar"
 "bytes"
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
 case "export":
  if len(os.Args) < 4 {
   fmt.Println("Error: missing args. Usage: kos export <id> <folder>")
   os.Exit(1)
  }
  cmdExport(os.Args[2], os.Args[3])
 case "load":
  if len(os.Args) < 3 {
   fmt.Println("Error: missing file. Usage: kos load <file.tar.gz>")
   os.Exit(1)
  }
  cmdLoad(os.Args[2])
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
 fmt.Println("  export <id> <dir> Extract a snapshot to a folder")
 fmt.Println("  load <file.tar.gz> Restore from a portable snapshot")
}

func cmdInit() {
 if _, err := os.Stat(KosDir); err == nil {
  fmt.Println("KOS repository already initialized.")
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
  os.WriteFile(IgnoreFile, []byte(""), 0644)
 }
 f, err := os.OpenFile(IgnoreFile, os.O_APPEND|os.O_WRONLY, 0644)
 if err != nil {
  fmt.Printf("Error opening %s: %v\n", IgnoreFile, err)
  os.Exit(1)
 }
 defer f.Close()
 f.WriteString(pattern + "\n")
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

 currentFiles, err := getWorkingDirectoryFiles()
 if err != nil {
  fmt.Printf("Error scanning directory: %v\n", err)
  os.Exit(1)
 }
 snapshot.Files = currentFiles

 manifest.Snapshots = append(manifest.Snapshots, snapshot)
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
 
  for path := range currentFiles {
   addFileToTar(tw, path)
  }
 
  manifestData, _ := json.MarshalIndent(manifest, "", "  ")
  mHeader := &tar.Header{
   Name: filepath.Join(KosDir, ManifestFile),
   Mode: 0644,
   Size: int64(len(manifestData)),
  }
  tw.WriteHeader(mHeader)
  tw.Write(manifestData)
 
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
  fmt.Println("║                     KOS TIMELINE                         ║")
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
 
  currentFiles, _ := getWorkingDirectoryFiles()
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


// FUNCTION cmdDiff ===================================
func cmdDiff(id1, id2 string) {
fmt.Println("DEBUG: Running new cmdDiff!")
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
  
  // Show actual code diffs for modified files
  for _, path := range modified {
   fmt.Printf("\n\033[1;34m📄 %s\033[0m\n", path)
   
   oldContent, err := extractFileFromTar(filepath.Join(KosDir, snap1.Filename), path)
   if err != nil {
    fmt.Printf("  \033[31mError reading old version: %v\033[0m\n", err)
    continue
   }
   
   newContent, err := extractFileFromTar(filepath.Join(KosDir, snap2.Filename), path)
   if err != nil {
    fmt.Printf("  \033[31mError reading new version: %v\033[0m\n", err)
    continue
   }
   
   showDiff(oldContent, newContent)
  }
 }
 fmt.Println()
}
  
  func cmdExport(id, destDir string) {
   manifest, err := loadManifest()
   if err != nil {
    fmt.Printf("Error: Not a KOS repository.\n")
    os.Exit(1)
   }
  
   var targetSnap *Snapshot
   for i := range manifest.Snapshots {
    if manifest.Snapshots[i].ID == id { targetSnap = &manifest.Snapshots[i] }
   }
   if targetSnap == nil {
    fmt.Printf("Error: Snapshot ID %s not found.\n", id)
    os.Exit(1)
   }
  
   tarPath := filepath.Join(KosDir, targetSnap.Filename)
   fmt.Printf("Exporting snapshot %s to %s...\n", id, destDir)
   
   if err := os.MkdirAll(destDir, 0755); err != nil {
    fmt.Printf("Error creating directory: %v\n", err)
    os.Exit(1)
   }
  
   if err := extractTarGz(tarPath, destDir); err != nil {
    fmt.Printf("Error extracting: %v\n", err)
    os.Exit(1)
   }
   fmt.Println("Export complete!")
  }
  
  func cmdLoad(tarPath string) {
   if _, err := os.Stat(tarPath); os.IsNotExist(err) {
    fmt.Printf("Error: File %s not found.\n", tarPath)
    os.Exit(1)
   }
  
   if _, err := os.Stat(KosDir); os.IsNotExist(err) {
    cmdInit()
   }
  
   fmt.Printf("Loading portable snapshot from %s...\n", tarPath)
   if err := extractTarGz(tarPath, "."); err != nil {
    fmt.Printf("Error extracting: %v\n", err)
    os.Exit(1)
   }
   fmt.Println("Load complete! History and files restored.")
  }
  
  // --- Helper Functions ---
  
  func saveManifest(m Manifest) error {
   data, err := json.MarshalIndent(m, "", "  ")
   if err != nil { return err }
   return os.WriteFile(filepath.Join(KosDir, ManifestFile), data, 0644)
  }
  
  func loadManifest() (Manifest, error) {
   var m Manifest
   data, err := os.ReadFile(filepath.Join(KosDir, ManifestFile))
   if err != nil { return m, err }
   err = json.Unmarshal(data, &m)
   return m, err
  }
  
  func loadIgnores() []string {
   data, err := os.ReadFile(IgnoreFile)
   if err != nil { return []string{} }
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
   if err != nil { return "", err }
   defer f.Close()
   h := sha256.New()
   if _, err := io.Copy(h, f); err != nil { return "", err }
   return fmt.Sprintf("%x", h.Sum(nil)), nil
  }
  
  func getWorkingDirectoryFiles() (map[string]string, error) {
   files := make(map[string]string)
   ignores := loadIgnores()
  
   err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
    if err != nil { return err }
    if path == KosDir || path == IgnoreFile {
     if d.IsDir() { return filepath.SkipDir }
     return nil
    }
    for _, pattern := range ignores {
     if pattern == "" { continue }
     match, _ := filepath.Match(pattern, filepath.Base(path))
     if match {
      if d.IsDir() { return filepath.SkipDir }
      return nil
     }
    }
    if d.IsDir() { return nil }
    hash, err := getFileHash(path)
    if err != nil { return nil }
    files[path] = hash
    return nil
   })
   return files, err
  }

  func addFileToTar(tw *tar.Writer, path string) {
   f, err := os.Open(path)
   if err != nil { return }
   defer f.Close()
  
   info, err := f.Stat()
   if err != nil { return }
  
   header, err := tar.FileInfoHeader(info, info.Name())
   if err != nil { return }
   header.Name = path
  
   if err := tw.WriteHeader(header); err != nil { return }
   io.Copy(tw, f)
  }
  
  // FIXED: Uses absolute paths to prevent false-positive zip-slip errors when destDir is "."
  func extractTarGz(tarPath, destDir string) error {
   file, err := os.Open(tarPath)
   if err != nil { return err }
   defer file.Close()
  
   gzr, err := gzip.NewReader(file)
   if err != nil { return err }
   defer gzr.Close()
  
   tr := tar.NewReader(gzr)
   
   // Get absolute path of destination for secure zip-slip check
   absDest, err := filepath.Abs(destDir)
   if err != nil { return err }
  
   for {
    header, err := tr.Next()
    if err == io.EOF { break }
    if err != nil { return err }
  
    target := filepath.Join(destDir, header.Name)
    absTarget, err := filepath.Abs(target)
    if err != nil { return err }
    
    // Secure zip-slip protection using absolute paths
    if !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) && absTarget != absDest {
     return fmt.Errorf("illegal file path: %s", header.Name)
    }
  
    switch header.Typeflag {
    case tar.TypeDir:
     os.MkdirAll(target, 0755)
    case tar.TypeReg:
     os.MkdirAll(filepath.Dir(target), 0755)
     f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
     if err != nil { continue }
     io.Copy(f, tr)
     f.Close()
    }
   }
   return nil
  }

// Shows line-by-line diff with smart insertion/deletion detection
func showDiff(oldContent, newContent string) {
 oldLines := strings.Split(oldContent, "\n")
 newLines := strings.Split(newContent, "\n")

 i, j := 0, 0
 for i < len(oldLines) || j < len(newLines) {
  if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
   if strings.TrimSpace(oldLines[i]) != "" {
    fmt.Printf("  \033[2;37m%4d | %s\033[0m\n", i+1, oldLines[i])
   } else {
    fmt.Println()
   }
   i++
   j++
  } else if i < len(oldLines) && j+1 < len(newLines) && oldLines[i] == newLines[j+1] {
   fmt.Printf("  \033[32m+ %4d | %s\033[0m\n", j+1, newLines[j])
   j++
  } else if i+1 < len(oldLines) && j < len(newLines) && oldLines[i+1] == newLines[j] {
   fmt.Printf("  \033[31m- %4d | %s\033[0m\n", i+1, oldLines[i])
   i++
  } else {
   if i < len(oldLines) {
    fmt.Printf("  \033[31m- %4d | %s\033[0m\n", i+1, oldLines[i])
   }
   if j < len(newLines) {
    fmt.Printf("  \033[32m+ %4d | %s\033[0m\n", j+1, newLines[j])
   }
   i++
   j++
  }
 }
}
//Recovery fixed
//Day 2 Testing 2

// Extracts a single file from a .tar.gz archive
func extractFileFromTar(tarPath, fileName string) (string, error) {
 file, err := os.Open(tarPath)
 if err != nil {
  return "", err
 }
 defer file.Close()

 gzr, err := gzip.NewReader(file)
 if err != nil {
  return "", err
 }
 defer gzr.Close()

 tr := tar.NewReader(gzr)

 for {
  header, err := tr.Next()
  if err == io.EOF {
   break
  }
  if err != nil {
   return "", err
  }

  if header.Name == fileName && header.Typeflag == tar.TypeReg {
   var buf bytes.Buffer
   if _, err := io.Copy(&buf, tr); err != nil {
    return "", err
   }
   return buf.String(), nil
  }
 }

 return "", fmt.Errorf("file not found in archive")
}
