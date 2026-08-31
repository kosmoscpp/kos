package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
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
	ManifestFile  = "manifest.json"
	IgnoreFile    = ".kosignore"
	GlobalKosBase = ".kos-global"
)

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func updateProjectsIndex(uuid, projectName string) {
	home, _ := os.UserHomeDir()
	indexPath := filepath.Join(home, GlobalKosBase, "projects.json")
	var index map[string]string
	if data, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(data, &index)
	} else {
		index = make(map[string]string)
	}
	index[uuid] = projectName
	data, _ := json.MarshalIndent(index, "", "  ")
	os.WriteFile(indexPath, data, 0644)
}

func getKosDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if data, err := os.ReadFile(".kos-project"); err == nil {
		projectRef := strings.TrimSpace(string(data))
		if projectRef != "" {
			if isUUID(projectRef) {
				return filepath.Join(home, GlobalKosBase, projectRef)
			}
			return filepath.Join(home, GlobalKosBase, projectRef)
		}
	}
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)
	if projectName == "/" || projectName == "" {
		projectName = "root_project"
	}
	return filepath.Join(home, GlobalKosBase, projectName)
}

type Snapshot struct {
	ID        string            `json:"id"`
	Timestamp int64             `json:"timestamp"`
	Message   string            `json:"message"`
	Filename  string            `json:"filename"`
	Files     map[string]string `json:"files"`
	Tag       string            `json:"tag,omitempty"`
}

type Manifest struct {
	Snapshots []Snapshot `json:"snapshots"`
}

type ProjectInfo struct {
	UUID        string
	Name        string
	CommitCount int
	LastCommit  time.Time
	LastMessage string
	TotalSize   int64
	Files       []string
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
			fmt.Println("Error: missing pattern")
			os.Exit(1)
		}
		cmdIgnore(os.Args[2])
	case "save":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing message")
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
			fmt.Println("Error: missing IDs")
			os.Exit(1)
		}
		if len(os.Args) == 4 {
			cmdDiffSummary(os.Args[2], os.Args[3])
		} else {
			cmdDiffFile(os.Args[2], os.Args[3], os.Args[4])
		}
	case "export":
		if len(os.Args) < 4 {
			fmt.Println("Error: missing args")
			os.Exit(1)
		}
		cmdExport(os.Args[2], os.Args[3])
	case "load":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing argument")
			os.Exit(1)
		}
		cmdLoad(os.Args[2:]...)
	case "stats":
		cmdStats()
	case "ks":
		cmdQuickSave()
	case "projects":
		cmdProjects()
	case "rename":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing new name")
			os.Exit(1)
		}
		cmdRename(os.Args[2])
	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing keyword")
			os.Exit(1)
		}
		cmdSearch(os.Args[2])
	case "share":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing snapshot ID")
			os.Exit(1)
		}
		cmdShare(os.Args[2])
	case "tag":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing tag name. Usage: kos tag <name> [snapshot-id]")
			os.Exit(1)
		}
		if len(os.Args) == 3 {
			cmdTag(os.Args[2], "")
		} else {
			cmdTag(os.Args[2], os.Args[3])
		}
	case "checkout":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing snapshot ID or tag. Usage: kos checkout <id-or-tag>")
			os.Exit(1)
		}
		cmdCheckout(os.Args[2:]...)
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
	fmt.Println("  load <target>      Load project or snapshot file")
	fmt.Println("  stats              Show project statistics")
	fmt.Println("  ks                 Quick save with timestamp")
	fmt.Println("  projects           List all projects in global store")
	fmt.Println("  rename <name>      Rename the current project")
	fmt.Println("  search <keyword>   Search commit messages")
	fmt.Println("  share <id>         Export snapshot as portable file")
	fmt.Println("  tag <name> [id]    Tag a snapshot with a name")
	fmt.Println("  checkout <id|tag>  Restore a snapshot to current directory")
}

func cmdInit() {
	if _, err := os.Stat(".kos-project"); err == nil {
		fmt.Println("KOS repository already initialized.")
		return
	}
	projectUUID := generateUUID()
	if err := os.WriteFile(".kos-project", []byte(projectUUID), 0644); err != nil {
		fmt.Printf("Error creating .kos-project: %v\n", err)
		os.Exit(1)
	}
	kosDir := getKosDir()
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
	cwd, _ := os.Getwd()
	projectName := filepath.Base(cwd)
	updateProjectsIndex(projectUUID, projectName)
	fmt.Printf("Initialized empty KOS repository in %s\n", kosDir)
	fmt.Printf("Project UUID: %s\n", projectUUID)
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
		Name: ".kos/" + ManifestFile,
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

func formatRelativeTime(timestamp int64) string {
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 60 {
		return "just now"
	} else if diff < 3600 {
		mins := diff / 60
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	} else if diff < 86400 {
		hours := diff / 3600
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 2592000 {
		days := diff / 86400
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	return "older"
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
		timeStr2 := formatRelativeTime(snap.Timestamp)
		tagStr := ""
		if snap.Tag != "" {
			tagStr = fmt.Sprintf(" \033[1;32m[%s]\033[0m", snap.Tag)
		}
		fmt.Printf("  \033[36m●\033[0m \033[1;33m%-8s\033[0m ┃ \033[2;37m%s (%s)\033[0m ┃ \033[1;37m%s\033[0m%s\n", snap.ID, timeStr, timeStr2, snap.Message, tagStr)
		fmt.Printf("  \033[36m│\033[0m           ┃  %d files\n", len(snap.Files))
		if i == 0 {
			fmt.Printf("  \033[36m└──\033[0m (End of history)\n\n")
		} else {
			fmt.Printf("  \033[36m│\033[0m\n")
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
		for _, f := range added {
			fmt.Printf("  %s\n", f)
		}
	}
	if len(modified) > 0 {
		fmt.Printf("\033[33m~ Modified (%d):\033[0m\n", len(modified))
		for _, f := range modified {
			fmt.Printf("  %s\n", f)
		}
	}
	if len(deleted) > 0 {
		fmt.Printf("\033[31m- Deleted (%d):\033[0m\n", len(deleted))
		for _, f := range deleted {
			fmt.Printf("  %s\n", f)
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
		if manifest.Snapshots[i].ID == id {
			targetSnap = &manifest.Snapshots[i]
		}
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

func resolveRef(ref string) *Snapshot {
	manifest, err := loadManifest()
	if err != nil {
		return nil
	}
	snaps := manifest.Snapshots
	if len(snaps) == 0 {
		return nil
	}
	if ref == "HEAD" {
		return &snaps[len(snaps)-1]
	}
	if strings.HasPrefix(ref, "HEAD~") {
		parts := strings.Split(ref, "~")
		if len(parts) == 2 {
			var offset int
			fmt.Sscanf(parts[1], "%d", &offset)
			idx := len(snaps) - 1 - offset
			if idx >= 0 && idx < len(snaps) {
				return &snaps[idx]
			}
		}
		return nil
	}
	for i := range snaps {
		if snaps[i].ID == ref || snaps[i].Tag == ref {
			return &snaps[i]
		}
	}
	return nil
}

func cmdCheckout(args ...string) {
	if len(args) == 0 {
		fmt.Println("Error: missing reference. Usage: kos checkout <id-or-tag> [filename]")
		os.Exit(1)
	}
	ref := args[0]
	var filename string
	if len(args) > 1 {
		filename = args[1]
	}
	snap := resolveRef(ref)
	if snap == nil {
		fmt.Printf("Error: Snapshot '%s' not found (by ID or tag)\n", ref)
		os.Exit(1)
	}
	fmt.Printf("\n\033[1;33m⚠️  CHECKOUT WARNING\033[0m\n")
	fmt.Printf("This will restore snapshot \033[1;36m%s\033[0m", snap.ID)
	if snap.Tag != "" {
		fmt.Printf(" \033[1;32m[%s]\033[0m", snap.Tag)
	}
	fmt.Println()
	if filename != "" {
		fmt.Printf("Target: Single file \033[1;37m%s\033[0m\n", filename)
	} else {
		fmt.Printf("Target: \033[1;37mEntire working directory\033[0m\n")
	}
	fmt.Printf("Message: \033[2;37m%s\033[0m\n", snap.Message)
	fmt.Printf("\n\033[33mAre you sure? This will overwrite current files. (y/N): \033[0m")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("\n\033[2;37mCheckout cancelled.\033[0m\n")
		return
	}
	tarPath := filepath.Join(getKosDir(), snap.Filename)
	if filename != "" {
		content, err := extractFileFromTar(tarPath, filename)
		if err != nil {
			fmt.Printf("\n\033[31mError: File '%s' not found in snapshot %s\033[0m\n", filename, snap.ID)
			os.Exit(1)
		}
		os.MkdirAll(filepath.Dir(filename), 0755)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			fmt.Printf("\n\033[31mError writing file: %v\033[0m\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n\033[1;32m✅ Restored '%s' from snapshot %s\033[0m\n\n", filename, snap.ID)
	} else {
		if err := extractTarGz(tarPath, "."); err != nil {
			fmt.Printf("Error: Failed to extract snapshot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n\033[1;32m✅ Checked out snapshot %s\033[0m\n", snap.ID)
		fmt.Printf("\033[2;37mYour working directory now matches this snapshot.\033[0m\n\n")
	}
}

func findProjectsByName(name string) []string {
	home, _ := os.UserHomeDir()
	indexPath := filepath.Join(home, GlobalKosBase, "projects.json")
	var index map[string]string
	if data, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(data, &index)
	} else {
		return []string{}
	}
	var matches []string
	for uuid, projectName := range index {
		if projectName == name {
			matches = append(matches, uuid)
		}
	}
	return matches
}

func getProjectInfo(uuid string) (ProjectInfo, error) {
	home, _ := os.UserHomeDir()
	projectDir := filepath.Join(home, GlobalKosBase, uuid)
	manifestPath := filepath.Join(projectDir, ManifestFile)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return ProjectInfo{}, err
	}
	var m Manifest
	json.Unmarshal(data, &m)
	indexPath := filepath.Join(home, GlobalKosBase, "projects.json")
	var index map[string]string
	if indexData, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(indexData, &index)
	}
	info := ProjectInfo{
		UUID:        uuid,
		Name:        index[uuid],
		CommitCount: len(m.Snapshots),
	}
	if len(m.Snapshots) > 0 {
		lastSnap := m.Snapshots[len(m.Snapshots)-1]
		info.LastCommit = time.Unix(lastSnap.Timestamp, 0)
		info.LastMessage = lastSnap.Message
		info.Files = make([]string, 0, len(lastSnap.Files))
		for file := range lastSnap.Files {
			info.Files = append(info.Files, file)
		}
	}
	snapEntries, _ := os.ReadDir(projectDir)
	for _, entry := range snapEntries {
		if strings.HasPrefix(entry.Name(), "snap_") && strings.HasSuffix(entry.Name(), ".tar.gz") {
			if entryInfo, err := entry.Info(); err == nil {
				info.TotalSize += entryInfo.Size()
			}
		}
	}
	return info, nil
}

func showProjectPreview(info ProjectInfo) {
	fmt.Printf("\nProject \033[1;36m%s\033[0m\n", info.Name)
	fmt.Println("───────────────────────")
	fmt.Printf("Latest snapshot: \033[1;37m\"%s\"\033[0m\n", info.LastMessage)
	fmt.Printf("Files: \033[1;33m%d\033[0m\n", len(info.Files))
	fmt.Printf("Size: \033[1;33m%s\033[0m\n\n", formatBytes(info.TotalSize))
	limit := 5
	if len(info.Files) < limit {
		limit = len(info.Files)
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("  \033[2;37m%s\033[0m\n", info.Files[i])
	}
	if len(info.Files) > 5 {
		fmt.Printf("  \033[2;37m... and %d more\033[0m\n", len(info.Files)-5)
	}
	fmt.Println()
}

func loadProjectByUUID(uuid string, info ProjectInfo) {
	home, _ := os.UserHomeDir()
	globalProjectDir := filepath.Join(home, GlobalKosBase, uuid)
	manifestPath := filepath.Join(globalProjectDir, ManifestFile)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Printf("Error reading manifest: %v\n", err)
		os.Exit(1)
	}
	var m Manifest
	json.Unmarshal(data, &m)
	if len(m.Snapshots) == 0 {
		fmt.Println("Error: Project exists but has no snapshots.")
		os.Exit(1)
	}
	var targetSnap *Snapshot
	for i := len(m.Snapshots) - 1; i >= 0; i-- {
		if len(m.Snapshots[i].Files) > 0 {
			targetSnap = &m.Snapshots[i]
			break
		}
	}
	if targetSnap == nil {
		fmt.Println("Error: No snapshots with files found.")
		os.Exit(1)
	}
	tarPath := filepath.Join(globalProjectDir, targetSnap.Filename)
	fmt.Printf("Loading project '%s' into current directory...\n", info.Name)
	fmt.Printf("   Restoring from: %s (ID: %s)\n", targetSnap.Filename, targetSnap.ID)
	if err := extractTarGz(tarPath, "."); err != nil {
		fmt.Printf("Error extracting: %v\n", err)
		os.Exit(1)
	}
	ignorePath := filepath.Join(globalProjectDir, IgnoreFile)
	if _, err := os.Stat(ignorePath); err == nil {
		ignoreData, _ := os.ReadFile(ignorePath)
		os.WriteFile(IgnoreFile, ignoreData, 0644)
	}
	os.WriteFile(".kos-project", []byte(uuid), 0644)
	fmt.Println("Load complete! History and files restored.")
	fmt.Println("Tip: Run 'kos view' to see your restored timeline.")
}

func cmdLoad(args ...string) {
 if len(args) == 0 {
  fmt.Println("Error: missing argument")
  os.Exit(1)
 }
 target := args[0]
 
 // Handle .tar.gz file import
 if strings.HasSuffix(target, ".tar.gz") {
  if _, err := os.Stat(target); os.IsNotExist(err) {
   fmt.Printf("Error: File '%s' not found\n", target)
   os.Exit(1)
  }
  fmt.Printf("\n\033[1;36m📦 Loading from file: %s\033[0m\n\n", target)
  tempDir, err := os.MkdirTemp("", "kos-load-*")
  if err != nil {
   fmt.Printf("Error: Cannot create temp directory: %v\n", err)
   os.Exit(1)
  }
  defer os.RemoveAll(tempDir)
  if err := extractTarGz(target, tempDir); err != nil {
   fmt.Printf("Error: Failed to extract: %v\n", err)
   os.Exit(1)
  }
  embeddedManifestPath := filepath.Join(tempDir, ".kos", ManifestFile)
  var embeddedManifest Manifest
  if data, err := os.ReadFile(embeddedManifestPath); err == nil {
   json.Unmarshal(data, &embeddedManifest)
  } else {
   embeddedManifest = Manifest{Snapshots: []Snapshot{}}
  }
  projectUUID := generateUUID()
  home, _ := os.UserHomeDir()
  globalProjectDir := filepath.Join(home, GlobalKosBase, projectUUID)
  os.MkdirAll(globalProjectDir, 0755)
  tarData, err := os.ReadFile(target)
  if err != nil {
   fmt.Printf("Error: Cannot read snapshot file: %v\n", err)
   os.Exit(1)
  }
  baseName := filepath.Base(target)
  globalTarPath := filepath.Join(globalProjectDir, baseName)
  if err := os.WriteFile(globalTarPath, tarData, 0644); err != nil {
   fmt.Printf("Error: Cannot copy to global store: %v\n", err)
   os.Exit(1)
  }
  manifestData, _ := json.MarshalIndent(embeddedManifest, "", "  ")
  globalManifestPath := filepath.Join(globalProjectDir, ManifestFile)
  if err := os.WriteFile(globalManifestPath, manifestData, 0644); err != nil {
   fmt.Printf("Error: Cannot save manifest: %v\n", err)
   os.Exit(1)
  }
  if err := extractTarGz(target, "."); err != nil {
   fmt.Printf("Error: Failed to extract files: %v\n", err)
   os.Exit(1)
  }
  os.WriteFile(".kos-project", []byte(projectUUID), 0644)
  cwd, _ := os.Getwd()
  projectName := filepath.Base(cwd)
  updateProjectsIndex(projectUUID, projectName)
  fmt.Printf("\n\033[1;32m✅ Loaded successfully!\033[0m\n")
  fmt.Printf("   Project: \033[1;36m%s\033[0m\n", projectName)
  fmt.Printf("   UUID: \033[2;37m%s\033[0m\n", projectUUID)
  fmt.Printf("   Commits: \033[1;36m%d\033[0m\n", len(embeddedManifest.Snapshots))
  fmt.Printf("\n\033[2;37mRun 'kos view' to see the timeline.\033[0m\n\n")
  return
 }
 
 // Check if current directory is already a KOS project
 if _, err := os.Stat(".kos-project"); err == nil {
  fmt.Println("Error: Current directory is already a KOS project.")
  fmt.Println("Run 'kos load' from an empty directory.")
  os.Exit(1)
 }
 
 // Check if target is a UUID
 if isUUID(target) {
  info, err := getProjectInfo(target)
  if err != nil {
   fmt.Printf("Error: Project with UUID '%s' not found.\n", target)
   os.Exit(1)
  }
  loadProjectByUUID(target, info)
  return
 }
 
 // Find all projects with this name
 matches := findProjectsByName(target)
 
 if len(matches) == 0 {
  fmt.Printf("Error: No project named '%s' found in global store.\n", target)
  fmt.Println("Did you mean to run 'kos init' first?")
  os.Exit(1)
 }
 
 if len(matches) == 1 {
  uuid := matches[0]
  info, err := getProjectInfo(uuid)
  if err != nil {
   fmt.Printf("Error loading project: %v\n", err)
   os.Exit(1)
  }
  showProjectPreview(info)
  fmt.Printf("Load this project? [y/N]: ")
  var response string
  fmt.Scanln(&response)
  if response != "y" && response != "Y" {
   fmt.Println("Load cancelled.")
   return
  }
  loadProjectByUUID(uuid, info)
  return
 }
 
 // Multiple matches - show picker
 fmt.Printf("\n\033[1;33m🔀 Multiple projects named '%s' found:\033[0m\n\n", target)
 for i, uuid := range matches {
	info, _ := getProjectInfo(uuid)
  fmt.Printf("  [\033[1;33m%d\033[0m] \033[2;37m%s\033[0m │ %d commits │ %s\n",
   i+1, uuid[:8], info.CommitCount, info.LastCommit.Format("Jan 02, 15:04"))
 }
 
 fmt.Printf("\nSelect project [1-%d/Q]: ", len(matches))
 var choice string
 fmt.Scanln(&choice)
 if choice == "Q" || choice == "q" {
  fmt.Println("Load cancelled.")
  return
 }
 var idx int
 fmt.Sscanf(choice, "%d", &idx)
 if idx < 1 || idx > len(matches) {
  fmt.Println("Invalid selection.")
  os.Exit(1)
 }
 uuid := matches[idx-1]
 info, err := getProjectInfo(uuid)
 if err != nil {
  fmt.Printf("Error loading project: %v\n", err)
  os.Exit(1)
 }
 showProjectPreview(info)
 fmt.Printf("Load this project? [y/N]: ")
 var response string
 fmt.Scanln(&response)
 if response != "y" && response != "Y" {
  fmt.Println("Load cancelled.")
  return
 }
 loadProjectByUUID(uuid, info)
}

func cmdStats() {
	manifest, err := loadManifest()
	if err != nil {
		fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
		os.Exit(1)
	}
	projectName := "unknown"
	if data, err := os.ReadFile(".kos-project"); err == nil {
		projectName = strings.TrimSpace(string(data))
	} else {
		cwd, _ := os.Getwd()
		projectName = filepath.Base(cwd)
	}
	totalCommits := len(manifest.Snapshots)
	totalFiles := 0
	uniqueFiles := make(map[string]bool)
	var totalSize int64
	for _, snap := range manifest.Snapshots {
		totalFiles += len(snap.Files)
		for file := range snap.Files {
			uniqueFiles[file] = true
		}
		snapPath := filepath.Join(getKosDir(), snap.Filename)
		if info, err := os.Stat(snapPath); err == nil {
			totalSize += info.Size()
		}
	}
	sizeStr := formatBytes(totalSize)
	fmt.Println("\n\033[1;36m╔══════════════════════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;36m║\033[0m                    \033[1;37mKOS PROJECT STATS\033[0m                     \033[1;36m║\033[0m")
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
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("Quick save: %s", timestamp)
	cmdSave(message)
}

func cmdProjects() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error: Cannot determine home directory\n")
		os.Exit(1)
	}
	globalDir := filepath.Join(home, GlobalKosBase)
	indexPath := filepath.Join(globalDir, "projects.json")
	var index map[string]string
	if data, err := os.ReadFile(indexPath); err == nil {
		json.Unmarshal(data, &index)
	} else {
		index = make(map[string]string)
	}
	fmt.Println("\n\033[1;36m📦 KOS PROJECTS\033[0m\n")
	for uuid, projectName := range index {
		projectDir := filepath.Join(globalDir, uuid)
		manifestPath := filepath.Join(projectDir, ManifestFile)
		var m Manifest
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		json.Unmarshal(data, &m)
		var totalSize int64
		snapEntries, _ := os.ReadDir(projectDir)
		for _, snapEntry := range snapEntries {
			if strings.HasPrefix(snapEntry.Name(), "snap") && strings.HasSuffix(snapEntry.Name(), ".tar.gz") {
				info, _ := snapEntry.Info()
				if info != nil {
					totalSize += info.Size()
				}
			}
		}
		commitCount := len(m.Snapshots)
		sizeStr := formatBytes(totalSize)
		fmt.Printf("  \033[1;33m%-20s\033[0m \033[2;37m%d commits, %s\033[0m\n", projectName, commitCount, sizeStr)
	}
	fmt.Println()
}

func cmdRename(newName string) {
	if newName == "" || strings.Contains(newName, "/") || strings.Contains(newName, "\\") {
		fmt.Printf("Error: Invalid project name '%s'\n", newName)
		os.Exit(1)
	}
	home, _ := os.UserHomeDir()
	currentName, err := os.ReadFile(".kos-project")
	if err != nil {
		fmt.Printf("Error: Not a KOS project. Run 'kos init' first.\n")
		os.Exit(1)
	}
	currentNameStr := strings.TrimSpace(string(currentName))
	newGlobalPath := filepath.Join(home, GlobalKosBase, newName)
	if _, err := os.Stat(newGlobalPath); err == nil {
		fmt.Printf("Error: Project '%s' already exists in global store\n", newName)
		os.Exit(1)
	}
	oldGlobalPath := filepath.Join(home, GlobalKosBase, currentNameStr)
	if err := os.Rename(oldGlobalPath, newGlobalPath); err != nil {
		fmt.Printf("Error: Failed to rename global directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(".kos-project", []byte(newName), 0644); err != nil {
		fmt.Printf("Error: Failed to update .kos-project: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Project renamed from '%s' to '%s'\n", currentNameStr, newName)
}

func cmdSearch(keyword string) {
	manifest, err := loadManifest()
	if err != nil {
		fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
		os.Exit(1)
	}
	var matches []Snapshot
	for _, snap := range manifest.Snapshots {
		if strings.Contains(strings.ToLower(snap.Message), strings.ToLower(keyword)) {
			matches = append(matches, snap)
		}
	}
	if len(matches) == 0 {
		fmt.Printf("\n\033[33mNo commits found matching '%s'\033[0m\n\n", keyword)
		return
	}
	fmt.Printf("\n\033[1;36m🔍 SEARCH RESULTS: '%s'\033[0m\n", keyword)
	fmt.Printf("\033[2;37mFound %d matching commits:\033[0m\n\n", len(matches))
	for _, snap := range matches {
		t := time.Unix(snap.Timestamp, 0)
		timeStr := t.Format("Jan 02, 15:04")
		fmt.Printf("  \033[1;33m%-8s\033[0m ┃ \033[2;37m%s\033[0m ┃ %s\n", snap.ID, timeStr, snap.Message)
	}
	fmt.Println()
}

func cmdTag(tagName, snapshotID string) {
	manifest, err := loadManifest()
	if err != nil {
		fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
		os.Exit(1)
	}
	if len(manifest.Snapshots) == 0 {
		fmt.Println("Error: No snapshots to tag.")
		os.Exit(1)
	}
	for _, snap := range manifest.Snapshots {
		if snap.Tag == tagName {
			fmt.Printf("Error: Tag '%s' already exists on snapshot %s\n", tagName, snap.ID)
			os.Exit(1)
		}
	}
	var targetSnap *Snapshot
	if snapshotID == "" {
		targetSnap = &manifest.Snapshots[len(manifest.Snapshots)-1]
	} else {
		for i := range manifest.Snapshots {
			if manifest.Snapshots[i].ID == snapshotID {
				targetSnap = &manifest.Snapshots[i]
				break
			}
		}
		if targetSnap == nil {
			fmt.Printf("Error: Snapshot '%s' not found\n", snapshotID)
			os.Exit(1)
		}
	}
	targetSnap.Tag = tagName
	if err := saveManifest(manifest); err != nil {
		fmt.Printf("Error saving manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Tagged snapshot \033[1;33m%s\033[0m as \033[1;32m[%s]\033[0m\n", targetSnap.ID, tagName)
}

func cmdShare(snapshotID string) {
 _, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository. Run 'kos init'.\n")
  os.Exit(1)
 }
 targetSnap := resolveRef(snapshotID)
 if targetSnap == nil {
  fmt.Printf("Error: Snapshot '%s' not found (by ID or tag)\n", snapshotID)
  os.Exit(1)
 }
	sourcePath := filepath.Join(getKosDir(), targetSnap.Filename)
	destPath := targetSnap.Filename
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		fmt.Printf("Error: Cannot read snapshot: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		fmt.Printf("Error: Cannot write file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n\033[1;32m✅ Snapshot exported!\033[0m\n")
	fmt.Printf("   File: \033[1;36m%s\033[0m\n", destPath)
	fmt.Printf("   Size: \033[1;36m%s\033[0m\n", formatBytes(int64(len(data))))
	fmt.Printf("\n\033[2;37mShare this file via email, USB, or cloud storage.\033[0m\n")
	fmt.Printf("\033[2;37mRecipients can load it with: kos load %s\033[0m\n\n", destPath)
}

func cmdDiffSummary(id1, id2 string) {
 _, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository.\n")
  os.Exit(1)
 }
 snap1 := resolveRef(id1)
 snap2 := resolveRef(id2)
 if snap1 == nil || snap2 == nil {
  fmt.Printf("Error: One or both references not found (by ID, tag, or HEAD).\n")
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
	fmt.Printf("\n\033[1;36m--- DIFF SUMMARY: %s vs %s ---\033[0m\n", id1, id2)
	if len(added) == 0 && len(deleted) == 0 && len(modified) == 0 {
		fmt.Println("\033[32mNo differences found.\033[0m\n")
		return
	}
	if len(added) > 0 {
		fmt.Printf("\033[32m+ Added (%d):\033[0m\n", len(added))
		for _, f := range added {
			fmt.Printf("  %s\n", f)
		}
	}
	if len(deleted) > 0 {
		fmt.Printf("\033[31m- Deleted (%d):\033[0m\n", len(deleted))
		for _, f := range deleted {
			fmt.Printf("  %s\n", f)
		}
	}
	if len(modified) > 0 {
		fmt.Printf("\033[33m~ Modified (%d):\033[0m\n", len(modified))
		for _, path := range modified {
			oldContent, _ := extractFileFromTar(filepath.Join(getKosDir(), snap1.Filename), path)
			newContent, _ := extractFileFromTar(filepath.Join(getKosDir(), snap2.Filename), path)
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
	fmt.Printf("  \033[1;6mkos diff %s %s <filename>\033[0m\n\n", id1, id2)
}

func cmdDiffFile(id1, id2, filename string) {
 _, err := loadManifest()
 if err != nil {
  fmt.Printf("Error: Not a KOS repository.\n")
  os.Exit(1)
 }
 snap1 := resolveRef(id1)
 snap2 := resolveRef(id2)
 if snap1 == nil || snap2 == nil {
  fmt.Printf("Error: One or both references not found (by ID, tag, or HEAD).\n")
  os.Exit(1)
 }
	fmt.Printf("\n\033[1;36m--- CODE DIFF: %s ---\033[0m\n", filename)
	oldContent, err := extractFileFromTar(filepath.Join(getKosDir(), snap1.Filename), filename)
	if err != nil {
		fmt.Printf("\033[31mError reading old version: %v\033[0m\n", err)
		os.Exit(1)
	}
	newContent, err := extractFileFromTar(filepath.Join(getKosDir(), snap2.Filename), filename)
	if err != nil {
		fmt.Printf("\033[31mError reading new version: %v\033[0m\n", err)
		os.Exit(1)
	}
	showDiff(oldContent, newContent)
	fmt.Println()
}

func showDiff(oldContent, newContent string) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	if len(oldLines) > 0 && oldLines[len(oldLines)-1] == "" {
		oldLines = oldLines[:len(oldLines)-1]
	}
	if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
		newLines = newLines[:len(newLines)-1]
	}
	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
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
	i, j := m, n
	var result []string
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			if strings.TrimSpace(oldLines[i-1]) != "" {
				result = append(result, fmt.Sprintf("  \033[2;37m%4d | %s\033[0m", i, oldLines[i-1]))
			} else {
				result = append(result, "")
			}
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, fmt.Sprintf("  \033[32m+ %4d | %s\033[0m", j, newLines[j-1]))
			j--
		} else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			result = append(result, fmt.Sprintf("  \033[31m- %4d | %s\033[0m", i, oldLines[i-1]))
			i--
		}
	}
	for k := len(result) - 1; k >= 0; k-- {
		fmt.Println(result[k])
	}
}

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

func getWorkingDirectoryFiles() (map[string]string, error) {
	files := make(map[string]string)
	ignores := loadIgnores()
	kosDir := getKosDir()
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		absPath, _ := filepath.Abs(path)
		if strings.HasPrefix(absPath, kosDir) {
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

func addFileToTar(tw *tar.Writer, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return
	}
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return
	}
	header.Name = path
	if err := tw.WriteHeader(header); err != nil {
		return
	}
	io.Copy(tw, f)
}

func extractTarGz(tarPath, destDir string) error {
 file, err := os.Open(tarPath)
 if err != nil {
  return err
 }
 defer file.Close()
 gzr, err := gzip.NewReader(file)
 if err != nil {
  return err
 }
 defer gzr.Close()
 tr := tar.NewReader(gzr)
 absDest, err := filepath.Abs(destDir)
 if err != nil {
  return err
 }
 for {
  header, err := tr.Next()
  if err == io.EOF {
   break
  }
  if err != nil {
   return err
  }
  
  // Safe path handling - preserve directory structure
  cleanName := header.Name
  
  // Remove .kos/ prefix if present
  if strings.HasPrefix(cleanName, ".kos/") {
   cleanName = strings.TrimPrefix(cleanName, ".kos/")
  }
  
  // Skip empty or dangerous paths
  if cleanName == "" || cleanName == "." || strings.Contains(cleanName, "..") {
   continue
  }
  
  target := filepath.Join(destDir, cleanName)
  
  // Security check: prevent path traversal
  absTarget, err := filepath.Abs(target)
  if err != nil {
   return err
  }
  if !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) && absTarget != absDest {
   fmt.Printf("Warning: Skipping unsafe path: %s\n", header.Name)
   continue
  }
  
  switch header.Typeflag {
  case tar.TypeDir:
   if err := os.MkdirAll(target, 0755); err != nil {
    fmt.Printf("Warning: Cannot create directory %s: %v\n", target, err)
   }
  case tar.TypeReg:
   if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
    fmt.Printf("Warning: Cannot create parent directory for %s: %v\n", target, err)
    continue
   }
   f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
   if err != nil {
    fmt.Printf("Warning: Cannot create file %s: %v\n", target, err)
    continue
   }
   if _, err := io.Copy(f, tr); err != nil {
    f.Close()
    fmt.Printf("Warning: Error writing file %s: %v\n", target, err)
    continue
   }
   f.Close()
  }
 }
 return nil
}

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

