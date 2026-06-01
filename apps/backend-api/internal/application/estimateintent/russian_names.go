package estimateintent

import (
	"regexp"
	"strings"
	"unicode"
)

// ensureRussianObjectNames — наименования стройки, объекта и позиций только на русском (кириллица).
func ensureRussianObjectNames(e Estimate) Estimate {
	e.ProjectName = ensureRussianName(e.ProjectName)
	e.ObjectDescription = ensureRussianName(e.ObjectDescription)
	for i := range e.LineItems {
		e.LineItems[i].Description = ensureRussianName(e.LineItems[i].Description)
	}
	return e
}

func ensureRussianName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	value = replaceKnownLatinTokens(value)
	value = latinHomoglyphsToCyrillic(value)
	value = transliterateRemainingLatin(value)
	return collapseSpaces(value)
}

var latinTokenReplacements = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)\bTRC\b`), "ТРЦ"},
	{regexp.MustCompile(`(?i)TRC-`), "ТРЦ-"},
	{regexp.MustCompile(`(?i)-TRC\b`), "-ТРЦ"},
	{regexp.MustCompile(`(?i)\bTRK\b`), "ТРК"},
	{regexp.MustCompile(`(?i)TRK-`), "ТРК-"},
	{regexp.MustCompile(`(?i)\bTRTS\b`), "ТРЦ"},
	{regexp.MustCompile(`(?i)\bCrystal\b`), "Кристалл"},
	{regexp.MustCompile(`(?i)\bKristall\b`), "Кристалл"},
	{regexp.MustCompile(`(?i)\bPremier\b`), "Премьер"},
	{regexp.MustCompile(`(?i)\bPremyer\b`), "Премьер"},
	{regexp.MustCompile(`(?i)\bAP(\d)`), "АР$1"},
	{regexp.MustCompile(`(?i)\bAR(\d)`), "АР$1"},
}

func replaceKnownLatinTokens(s string) string {
	for _, item := range latinTokenReplacements {
		s = item.re.ReplaceAllString(s, item.replacement)
	}
	return s
}

// latinHomoglyphsToCyrillic — визуально похожие латинские буквы в кириллицу (TRC → почти ТРЦ после замены токенов).
func latinHomoglyphsToCyrillic(s string) string {
	var b strings.Builder
	for _, r := range s {
		if cyr, ok := latinHomoglyphMap[r]; ok {
			b.WriteRune(cyr)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var latinHomoglyphMap = map[rune]rune{
	'A': 'А', 'a': 'а',
	'B': 'В', 'b': 'в',
	'C': 'С', 'c': 'с',
	'E': 'Е', 'e': 'е',
	'H': 'Н', 'h': 'н',
	'K': 'К', 'k': 'к',
	'M': 'М', 'm': 'м',
	'O': 'О', 'o': 'о',
	'P': 'Р', 'p': 'р',
	'T': 'Т', 't': 'т',
	'X': 'Х', 'x': 'х',
	'Y': 'У', 'y': 'у',
}

// transliterateRemainingLatin — оставшаяся латиница в русские буквы (для редких символов вроде D, G, L).
func transliterateRemainingLatin(s string) string {
	var b strings.Builder
	for _, r := range s {
		if cyr, ok := latinTransliterationMap[r]; ok {
			b.WriteRune(cyr)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var latinTransliterationMap = map[rune]rune{
	'A': 'А', 'a': 'а',
	'B': 'Б', 'b': 'б',
	'C': 'Ц', 'c': 'ц',
	'D': 'Д', 'd': 'д',
	'E': 'Е', 'e': 'е',
	'F': 'Ф', 'f': 'ф',
	'G': 'Г', 'g': 'г',
	'H': 'Х', 'h': 'х',
	'I': 'И', 'i': 'и',
	'J': 'Й', 'j': 'й',
	'K': 'К', 'k': 'к',
	'L': 'Л', 'l': 'л',
	'M': 'М', 'm': 'м',
	'N': 'Н', 'n': 'н',
	'O': 'О', 'o': 'о',
	'P': 'П', 'p': 'п',
	'Q': 'К', 'q': 'к',
	'R': 'Р', 'r': 'р',
	'S': 'С', 's': 'с',
	'T': 'Т', 't': 'т',
	'U': 'У', 'u': 'у',
	'V': 'В', 'v': 'в',
	'W': 'В', 'w': 'в',
	'X': 'К', 'x': 'к',
	'Y': 'Ы', 'y': 'ы',
	'Z': 'З', 'z': 'з',
}

func containsLatinLetters(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Latin) && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
