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
 ManifestFile   = "manifest.json"
 IgnoreFile     = ".kosignore"
 GlobalKosBase  = ".kos-global"
)

// getgetKosDir() returns the global path for the current project
func getKosDir() string {
 home, err := os.UserHomeDir()
 if err != nil {
  home = "."
 }
 
 // Check if .kos-project exists (for loaded projects)
 if data, err := os.ReadFile(".kos-project"); err == nil {
  projectName := strings.TrimSpace(string(data))
  if projectName != "" {
   result := filepath.Join(home, GlobalKosBase, projectName)  
   return result
  }
 }
 
 // Otherwise, use current directory name
 cwd, _ := os.Getwd()
 projectName := filepath.Base(cwd)
 if projectName == "/" || projectName == "" {
  projectName = "root_project"
 }
 
 result := filepath.Join(home, GlobalKosBase, projectName)
 return result
}

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
   fmt.Println("Error: missing IDs. Usage: kos diff <id1> <id2> [filename]")
   os.Exit(1)
  }
  if len(os.Args) == 4 {
   cmdDiffSummary(os.Args[2], os.Args[3])
  } else {
   cmdDiffFile(os.Args[2], os.Args[3], os.Args[4])
  }
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
 case "stats":
  cmdStats()
 case "ks":
  cmdQuickSave()
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
 fmt.Println("  init               Initialize a new KOS repository")
 fmt.Println("  ignore <pattern>   Add a pattern to .kosignore")
 fmt.Println("  save <message>     Create a new snapshot")
 fmt.Println("  view               View the ASCII timeline")
 fmt.Println("  status             Show modified files")
 fmt.Println("  diff <id1> <id2>   Compare two snapshots")
 fmt.Println("  export <id> <dir>  Extract a snapshot to a folder")
 fmt.Println("  load <file.tar.gz> Restore from a portable snapshot")
 fmt.Println("  stats              Show project statistics")
 fmt.Println("  ks                 Quick save with timestamp")
}

func cmdInit() {
 kosDir := getKosDir()
 
 if _, err := os.Stat(kosDir); err == nil {
  fmt.Println("KOS repository already initialized.")
  return
 }
 
 if err := os.MkdirAll(kosDir, 0755); err != nil {
  fmt.Printf("Error creating %s: %v\n", kosDir, err)
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
 
 fmt.Printf("Initialized empty KOS repository in %s\n", kosDir)
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
 filepathTar := filepath.Join(getKosDir(), filename)

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
   Name: filepath.Join(getKosDir(), ManifestFile),
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
  
   tarPath := filepath.Join(getKosDir(), targetSnap.Filename)
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
  
  func cmdLoad(projectName string) {
   home, _ := os.UserHomeDir()
   globalProjectDir := filepath.Join(home, GlobalKosBase, projectName)
   manifestPath := filepath.Join(globalProjectDir, ManifestFile)
  
   // Check if this project exists in the global store
   if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
    fmt.Printf("Error: No global repository found for project '%s'.\n", projectName)
    fmt.Println("Did you mean to run 'kos init' first?")
    os.Exit(1)
   }
  
   // Load the global manifest for this project
   var m Manifest
   data, err := os.ReadFile(manifestPath)
   if err != nil {
    fmt.Printf("Error reading manifest: %v\n", err)
    os.Exit(1)
   }
   json.Unmarshal(data, &m)
  
   if len(m.Snapshots) == 0 {
    fmt.Println("Error: Project exists but has no snapshots.")
    os.Exit(1)
   }
  
   // Get the latest snapshot
   latestSnap := m.Snapshots[len(m.Snapshots)-1]
   tarPath := filepath.Join(globalProjectDir, latestSnap.Filename)
  
   fmt.Printf("🌍 Loading global project '%s' into current directory...\n", projectName)
   fmt.Printf("   Restoring from: %s\n", latestSnap.Filename)
   
   if err := extractTarGz(tarPath, "."); err != nil {
    fmt.Printf("Error extracting: %v\n", err)
    os.Exit(1)
   }
   
   // Also restore the local .kosignore if it was tracked
   ignorePath := filepath.Join(globalProjectDir, IgnoreFile)
   if _, err := os.Stat(ignorePath); err == nil {
    // Copy it to current dir
    data, _ := os.ReadFile(ignorePath)
    os.WriteFile(IgnoreFile, data, 0644)
   }
  
   fmt.Println("✅ Load complete! History and files restored.")
   fmt.Println("💡 Tip: Run 'kos view' to see your restored timeline.")
  }
  
  // --- Helper Functions ---
  
func saveManifest(m Manifest) error {
 data, err := json.MarshalIndent(m, "", "  ")
 if err != nil {
  return err
 }
 return os.WriteFile(filepath.Join(getKosDir(), ManifestFile), data, 0644)
}

func loadManifest() (Manifest, error) {
 var m Manifest
 data, err := os.ReadFile(filepath.Join(getKosDir(), ManifestFile))
 if err != nil {
  return m, err
 }
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
    if path == getKosDir() || path == IgnoreFile {
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
func cmdStats() {
 manifest, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
  os.Exit(1)
 }

 // Get project name
 projectName := "unknown"
 if data, err := os.ReadFile(".kos-project"); err == nil {
  projectName = strings.TrimSpace(string(data))
 } else {
  cwd, _ := os.Getwd()
  projectName = filepath.Base(cwd)
 }

 // Calculate stats
 totalCommits := len(manifest.Snapshots)
 totalFiles := 0
 uniqueFiles := make(map[string]bool)
 var totalSize int64

 for _, snap := range manifest.Snapshots {
  totalFiles += len(snap.Files)
  for file := range snap.Files {
   uniqueFiles[file] = true
  }
  
  // Get snapshot file size
  snapPath := filepath.Join(getKosDir(), snap.Filename)
  if info, err := os.Stat(snapPath); err == nil {
   totalSize += info.Size()
  }
 }

 // Format size
 sizeStr := formatBytes(totalSize)

 // Print dashboard
 fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════╗\033[0m")
 fmt.Println("\033[1;36m║\033[0m                    \033[1;37mKOS PROJECT STATS\033[0m                    \033[1;36m║\033[0m")
 fmt.Println("\033[1;36m╚══════════════════════════════════════════════════════════╝\033[0m")
 fmt.Println()
fmt.Printf("  \033[1;33mProject Name:\033[0m      %s\n", projectName)
fmt.Printf("  \033[1;33mTotal Commits:\033[0m     %d\n", totalCommits)
fmt.Printf("  \033[1;33mUnique Files:\033[0m      %d\n", len(uniqueFiles))
fmt.Printf("  \033[1;33mFile Entries:\033[0m      %d\n", totalFiles) 
fmt.Printf("  \033[1;33mRepository Size:\033[0m   %s\n", sizeStr)
 
 if totalCommits > 0 {
  oldest := manifest.Snapshots[0]
  newest := manifest.Snapshots[len(manifest.Snapshots)-1]
  fmt.Printf("  \033[1;33mFirst Commit:\033[0m    %s (%s)\n", oldest.ID, time.Unix(oldest.Timestamp, 0).Format("Jan 02, 15:04"))
  fmt.Printf("  \033[1;33mLatest Commit:\033[0m   %s (%s)\n", newest.ID, time.Unix(newest.Timestamp, 0).Format("Jan 02, 15:04"))
 }
 
 fmt.Println()
}

func formatBytes(bytes int64) string {
 const unit = 1024
 if bytes < unit {
  return fmt.Sprintf("%d B", bytes)
 }
 div, exp := int64(unit), 0
 for n := bytes / unit; n >= unit; n /= unit {
  div *= unit
  exp++
 }
 return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func cmdQuickSave() {
 // Generate a timestamp-based message
 timestamp := time.Now().Format("2006-01-02 15:04:05")
 message := fmt.Sprintf("Quick save: %s", timestamp)
 cmdSave(message)
}


// ============DIFF================================
// 1. Compact Summary (Default behavior)
func cmdDiffSummary(id1, id2 string) {
 manifest, err := loadManifest()
 if err != nil { fmt.Printf("Error: Not a KOS repository.\n"); os.Exit(1) }

 var snap1, snap2 *Snapshot
 for i := range manifest.Snapshots {
  if manifest.Snapshots[i].ID == id1 { snap1 = &manifest.Snapshots[i] }
  if manifest.Snapshots[i].ID == id2 { snap2 = &manifest.Snapshots[i] }
 }
 if snap1 == nil || snap2 == nil {
  fmt.Printf("Error: One or both snapshot IDs not found.\n"); os.Exit(1)
 }

 var added, deleted, modified []string
 for path, hash2 := range snap2.Files {
  hash1, exists := snap1.Files[path]
  if !exists { added = append(added, path) } else if hash1 != hash2 { modified = append(modified, path) }
 }
 for path := range snap1.Files {
  if _, exists := snap2.Files[path]; !exists { deleted = append(deleted, path) }
 }

 fmt.Printf("\n\033[1;36m--- DIFF SUMMARY: %s vs %s ---\033[0m\n", id1, id2)
 if len(added) == 0 && len(deleted) == 0 && len(modified) == 0 {
  fmt.Println("\033[32mNo differences found.\033[0m\n"); return
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
  for _, path := range modified {
   oldContent, _ := extractFileFromTar(filepath.Join(getKosDir(), snap1.Filename), path)
   newContent, _ := extractFileFromTar(filepath.Join(getKosDir(), snap2.Filename), path)
   
   // Quick line count diff for summary
   oldLines := len(strings.Split(oldContent, "\n"))
   newLines := len(strings.Split(newContent, "\n"))
   diff := newLines - oldLines
   
   if diff > 0 {
    fmt.Printf("  \033[32m+%d\033[0m \033[2;37m%s\033[0m\n", diff, path)
   } else if diff < 0 {
    fmt.Printf("  \033[31m%d\033[0m \033[2;37m%s\033[0m\n", diff, path)
   } else {
    fmt.Printf("  \033[33m~\033[0m  \033[2;37m%s\033[0m\n", path)
  }
  }
 }
 fmt.Println("\n\033[2;37m💡 Tip: To see the actual code changes, run:\033[0m")
 fmt.Printf("  \033[1;36mkos diff %s %s <filename>\033[0m\n\n", id1, id2)
}

// 2. Detailed File Diff (When a filename is provided)
func cmdDiffFile(id1, id2, filename string) {
 manifest, err := loadManifest()
 if err != nil { fmt.Printf("Error: Not a KOS repository.\n"); os.Exit(1) }

 var snap1, snap2 *Snapshot
 for i := range manifest.Snapshots {
  if manifest.Snapshots[i].ID == id1 { snap1 = &manifest.Snapshots[i] }
  if manifest.Snapshots[i].ID == id2 { snap2 = &manifest.Snapshots[i] }
 }
 if snap1 == nil || snap2 == nil {
  fmt.Printf("Error: One or both snapshot IDs not found.\n"); os.Exit(1)
 }

 fmt.Printf("\n\033[1;36m--- CODE DIFF: %s ---\033[0m\n", filename)

 oldContent, err := extractFileFromTar(filepath.Join(getKosDir(), snap1.Filename), filename)
 if err != nil { fmt.Printf("\033[31mError reading old version: %v\033[0m\n", err); os.Exit(1) }
 
 newContent, err := extractFileFromTar(filepath.Join(getKosDir(), snap2.Filename), filename)
 if err != nil { fmt.Printf("\033[31mError reading new version: %v\033[0m\n", err); os.Exit(1) }

 showDiff(oldContent, newContent)
 fmt.Println()
}

// 3. The Line-by-Line Comparison (Kept from before, but now only runs on demand)
// Shows line-by-line diff using Longest Common Subsequence (LCS)
// This prevents the "cascading" effect when multiple lines are added/removed.
func showDiff(oldContent, newContent string) {
 oldLines := strings.Split(oldContent, "\n")
 newLines := strings.Split(newContent, "\n")

 // Clean up trailing empty lines from split
 if len(oldLines) > 0 && oldLines[len(oldLines)-1] == "" {
  oldLines = oldLines[:len(oldLines)-1]
 }
 if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
  newLines = newLines[:len(newLines)-1]
 }

 m, n := len(oldLines), len(newLines)
 
 // DP table for LCS length
 dp := make([][]int, m+1)
 for i := range dp {
  dp[i] = make([]int, n+1)
 }

 // Build the DP table
 for i := 1; i <= m; i++ {
  for j := 1; j <= n; j++ {
   if oldLines[i-1] == newLines[j-1] {
    dp[i][j] = dp[i-1][j-1] + 1
   } else if dp[i-1][j] > dp[i][j-1] {
    dp[i][j] = dp[i-1][j]
   } else {
    dp[i][j] = dp[i][j-1]
   }
  }
 }

 // Backtrack to find the actual diff
 i, j := m, n
 var result []string

 for i > 0 || j > 0 {
  if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
   // Line is unchanged
   if strings.TrimSpace(oldLines[i-1]) != "" {
    result = append(result, fmt.Sprintf("  \033[2;37m%4d | %s\033[0m", i, oldLines[i-1]))
   } else {
    result = append(result, "")
   }
   i--
   j--
  } else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
   // Line was added in new file
   result = append(result, fmt.Sprintf("  \033[32m+ %4d | %s\033[0m", j, newLines[j-1]))
   j--
  } else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
   // Line was deleted from old file
   result = append(result, fmt.Sprintf("  \033[31m- %4d | %s\033[0m", i, oldLines[i-1]))
   i--
  }
 }

 // Print in reverse order (top to bottom)
 for k := len(result) - 1; k >= 0; k-- {
  fmt.Println(result[k])
 }
}
