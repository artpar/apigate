package convention

import "strings"

// Pluralize returns the plural form of a word.
// Uses simple English pluralization rules.
func Pluralize(word string) string {
	if word == "" {
		return ""
	}

	// Check irregular plurals first
	if plural, ok := irregularPlurals[strings.ToLower(word)]; ok {
		// Preserve case of first letter
		if word[0] >= 'A' && word[0] <= 'Z' {
			return strings.ToUpper(plural[:1]) + plural[1:]
		}
		return plural
	}

	// Apply regular rules
	lower := strings.ToLower(word)

	// Words ending in 's', 'x', 'z', 'ch', 'sh' → add 'es'
	if strings.HasSuffix(lower, "s") ||
		strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") ||
		strings.HasSuffix(lower, "ch") ||
		strings.HasSuffix(lower, "sh") {
		return word + "es"
	}

	// Words ending in consonant + 'y' → change 'y' to 'ies'
	if strings.HasSuffix(lower, "y") && len(word) > 1 {
		secondLast := lower[len(lower)-2]
		if !isVowel(rune(secondLast)) {
			return word[:len(word)-1] + "ies"
		}
	}

	// Words ending in 'f' or 'fe' → change to 'ves'
	if strings.HasSuffix(lower, "f") {
		return word[:len(word)-1] + "ves"
	}
	if strings.HasSuffix(lower, "fe") {
		return word[:len(word)-2] + "ves"
	}

	// Default: add 's'
	return word + "s"
}

// Singularize returns the singular form of a word.
// Inverse of Pluralize.
func Singularize(word string) string {
	if word == "" {
		return ""
	}

	lower := strings.ToLower(word)

	// Check irregular singulars
	for singular, plural := range irregularPlurals {
		if strings.ToLower(plural) == lower {
			if word[0] >= 'A' && word[0] <= 'Z' {
				return strings.ToUpper(singular[:1]) + singular[1:]
			}
			return singular
		}
	}

	// Words ending in 'ies' → change to 'y'
	if strings.HasSuffix(lower, "ies") {
		return word[:len(word)-3] + "y"
	}

	// Words ending in 'ves' → change to 'f'
	if strings.HasSuffix(lower, "ves") {
		return word[:len(word)-3] + "f"
	}

	// Words ending in 'es' (after sibilants) → remove 'es'
	if strings.HasSuffix(lower, "ses") ||
		strings.HasSuffix(lower, "xes") ||
		strings.HasSuffix(lower, "zes") ||
		strings.HasSuffix(lower, "ches") ||
		strings.HasSuffix(lower, "shes") {
		return word[:len(word)-2]
	}

	// Words ending in 's' → remove 's'
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		return word[:len(word)-1]
	}

	return word
}

// isVowel returns true if the rune is a vowel.
func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}

// Common irregular plurals.
var irregularPlurals = map[string]string{
	"person":   "people",
	"man":      "men",
	"woman":    "women",
	"child":    "children",
	"foot":     "feet",
	"tooth":    "teeth",
	"goose":    "geese",
	"mouse":    "mice",
	"ox":       "oxen",
	"index":    "indices",
	"matrix":   "matrices",
	"vertex":   "vertices",
	"analysis": "analyses",
	"crisis":   "crises",
	"thesis":   "theses",
	"datum":    "data",
	"medium":   "media",
	"schema":   "schemas", // Technical usage prefers 'schemas'
	"status":   "statuses",
}
