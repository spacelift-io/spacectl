package api

import "strings"

func addTopLevelSpacing(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	previousNonEmpty := false
	previousWasTopLevelComment := false
	inTopLevelBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inTopLevelBlockComment {
				out = append(out, line)
			}
			continue
		}

		isTopLevelComment := isTopLevelCommentLine(line, trimmed, &inTopLevelBlockComment)
		isTopLevelDefinition := isTopLevelDefinitionLine(line, trimmed)
		isTopLevelLine := isTopLevelComment || isTopLevelDefinition

		if isTopLevelLine && previousNonEmpty && !previousWasTopLevelComment {
			out = append(out, "")
		}

		out = append(out, line)
		previousNonEmpty = true
		previousWasTopLevelComment = isTopLevelComment || inTopLevelBlockComment
	}

	return strings.Join(out, "\n") + "\n"
}

func isTopLevelCommentLine(line string, trimmed string, inBlock *bool) bool {
	if strings.TrimLeft(line, " \t") != line {
		return false
	}

	if *inBlock {
		if trimmed == "\"\"\"" {
			*inBlock = false
		}
		return true
	}

	if trimmed == "\"\"\"" {
		*inBlock = true
		return true
	}

	if strings.HasPrefix(trimmed, "\"\"\"") {
		if strings.HasSuffix(trimmed, "\"\"\"") && len(trimmed) > len("\"\"\"\"\"\"") {
			return true
		}
		*inBlock = true
		return true
	}

	return strings.HasPrefix(trimmed, "\"")
}

func isTopLevelDefinitionLine(line string, trimmed string) bool {
	if strings.TrimLeft(line, " \t") != line {
		return false
	}

	switch {
	case strings.HasPrefix(trimmed, "schema "):
		return true
	case strings.HasPrefix(trimmed, "type "):
		return true
	case strings.HasPrefix(trimmed, "interface "):
		return true
	case strings.HasPrefix(trimmed, "input "):
		return true
	case strings.HasPrefix(trimmed, "union "):
		return true
	case strings.HasPrefix(trimmed, "enum "):
		return true
	case strings.HasPrefix(trimmed, "scalar "):
		return true
	case strings.HasPrefix(trimmed, "directive "):
		return true
	case strings.HasPrefix(trimmed, "extend "):
		return true
	default:
		return false
	}
}
