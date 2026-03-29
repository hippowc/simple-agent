package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"simple-agent/internal/common"
)

// CompletionItem 表示一条补全：Label 为展示文案，Insert 为选中后替换输入框的整行内容。
type CompletionItem struct {
	Label  string
	Insert string
}

var rootSlashCommands = []string{"/model", "/prompt", "/tools", "/quit"}

// Completions 根据当前输入行返回补全项（仅读配置，可与 RunTurn 并发）。
func (a *Agent) Completions(line string) []CompletionItem {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return completionsForConfig(&a.cfg, line)
}

func completionsForConfig(cfg *common.Config, line string) []CompletionItem {
	if !strings.HasPrefix(line, "/") {
		return nil
	}
	line = strings.TrimRight(line, "\t")

	if !strings.Contains(line, " ") {
		return completeRootCommand(line)
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/model":
		return completeModel(cfg, line, parts)
	case "/prompt":
		return completePrompt(line, parts)
	default:
		return nil
	}
}

func completeRootCommand(line string) []CompletionItem {
	var out []CompletionItem
	for _, cmd := range rootSlashCommands {
		if strings.HasPrefix(cmd, line) && cmd != line {
			out = append(out, CompletionItem{Label: cmd, Insert: cmd + " "})
		}
	}
	if line == "/model" {
		out = append(out,
			CompletionItem{Label: "use", Insert: "/model use "},
			CompletionItem{Label: "add", Insert: "/model add "},
		)
	}
	if line == "/prompt" {
		out = append(out,
			CompletionItem{Label: "system", Insert: "/prompt system "},
			CompletionItem{Label: "user", Insert: "/prompt user "},
		)
	}
	return dedupeInsert(out)
}

func completeModel(cfg *common.Config, line string, parts []string) []CompletionItem {
	if len(parts) < 2 {
		return nil
	}
	switch parts[1] {
	case "use":
		return completeModelUse(cfg, line, parts)
	case "add":
		return nil
	default:
		sub := parts[1]
		var out []CompletionItem
		for _, w := range []string{"use", "add"} {
			if strings.HasPrefix(w, sub) && w != sub {
				out = append(out, CompletionItem{Label: w, Insert: "/model " + w + " "})
			}
		}
		return out
	}
}

func completeModelUse(cfg *common.Config, line string, parts []string) []CompletionItem {
	if len(parts) == 2 && strings.HasSuffix(line, " ") {
		return profileItems(cfg, "")
	}
	namePrefix := ""
	if len(parts) >= 3 {
		namePrefix = parts[len(parts)-1]
	}
	return profileItems(cfg, namePrefix)
}

func profileItems(cfg *common.Config, prefix string) []CompletionItem {
	var out []CompletionItem
	for _, p := range cfg.LLM.Profiles {
		if prefix != "" && !strings.HasPrefix(p.Name, prefix) {
			continue
		}
		out = append(out, CompletionItem{
			Label:  p.Name + " (" + p.Model + ")",
			Insert: "/model use " + p.Name,
		})
	}
	return out
}

func completePrompt(line string, parts []string) []CompletionItem {
	if len(parts) < 2 {
		return nil
	}
	if parts[1] != "system" && parts[1] != "user" {
		for _, w := range []string{"system", "user"} {
			if strings.HasPrefix(w, parts[1]) && w != parts[1] {
				return []CompletionItem{{Label: w, Insert: "/prompt " + w + " "}}
			}
		}
		return nil
	}
	if parts[1] == "system" {
		return completePromptSystem(line, parts)
	}
	return completePromptUser(line, parts)
}

func completePromptSystem(line string, parts []string) []CompletionItem {
	const pfx = "/prompt system "
	if len(parts) == 2 {
		if strings.HasSuffix(line, " ") {
			return []CompletionItem{
				{Label: "clear", Insert: pfx + "clear"},
				{Label: "file …", Insert: pfx + "file "},
			}
		}
		return nil
	}
	switch parts[2] {
	case "clear":
		return nil
	case "file":
		return fileLineCompletions(line, pfx+"file ")
	default:
		if len(parts) == 3 {
			t := parts[2]
			var out []CompletionItem
			for _, w := range []string{"clear", "file"} {
				if strings.HasPrefix(w, t) && w != t {
					if w == "clear" {
						out = append(out, CompletionItem{Label: w, Insert: pfx + "clear"})
					} else {
						out = append(out, CompletionItem{Label: w, Insert: pfx + "file "})
					}
				}
			}
			if len(out) > 0 {
				return out
			}
			if strings.HasPrefix(t, "@") {
				return atPathCompletions(line, pfx)
			}
		}
	}
	return nil
}

func completePromptUser(line string, parts []string) []CompletionItem {
	const pfx = "/prompt user "
	if len(parts) == 2 {
		if strings.HasSuffix(line, " ") {
			return []CompletionItem{
				{Label: "clear", Insert: pfx + "clear"},
				{Label: "file …", Insert: pfx + "file "},
			}
		}
		return nil
	}
	switch parts[2] {
	case "clear":
		return nil
	case "file":
		return fileLineCompletions(line, pfx+"file ")
	default:
		if len(parts) == 3 {
			t := parts[2]
			var out []CompletionItem
			for _, w := range []string{"clear", "file"} {
				if strings.HasPrefix(w, t) && w != t {
					if w == "clear" {
						out = append(out, CompletionItem{Label: w, Insert: pfx + "clear"})
					} else {
						out = append(out, CompletionItem{Label: w, Insert: pfx + "file "})
					}
				}
			}
			if len(out) > 0 {
				return out
			}
			if strings.HasPrefix(t, "@") {
				return atPathCompletions(line, pfx)
			}
		}
	}
	return nil
}

// fileLineCompletions 在 fixedPrefix（如 "/prompt system file "）之后补全路径。
func fileLineCompletions(line, fixedPrefix string) []CompletionItem {
	if !strings.HasPrefix(line, fixedPrefix) {
		return nil
	}
	rest := strings.TrimPrefix(line, fixedPrefix)
	return pathCompletions(rest, fixedPrefix)
}

// atPathCompletions：行形如 /prompt system @dir/file
func atPathCompletions(line, cmdPrefix string) []CompletionItem {
	i := strings.Index(line, "@")
	if i < 0 {
		return nil
	}
	after := line[i+1:]
	return pathCompletions(after, line[:i+1])
}

func pathCompletions(partialRest, insertPrefix string) []CompletionItem {
	dir, name := filepath.Split(partialRest)
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			continue
		}
		if e.IsDir() {
			names = append(names, e.Name()+string(os.PathSeparator))
		} else {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out []CompletionItem
	for _, n := range names {
		if name != "" && !strings.HasPrefix(n, name) {
			continue
		}
		suffix := filepath.Join(dir, n)
		if dir == "." {
			suffix = n
		}
		insert := insertPrefix + suffix
		out = append(out, CompletionItem{Label: suffix, Insert: insert})
		if len(out) >= 40 {
			break
		}
	}
	return out
}

func dedupeInsert(items []CompletionItem) []CompletionItem {
	seen := map[string]struct{}{}
	var out []CompletionItem
	for _, it := range items {
		if _, ok := seen[it.Insert]; ok {
			continue
		}
		seen[it.Insert] = struct{}{}
		out = append(out, it)
	}
	return out
}
