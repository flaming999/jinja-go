package jinja

import (
	"encoding/json"
	"reflect"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

func registerBuiltinFilters(env *Environment) {
	filters := map[string]FilterFunc{
		// String filters
		"upper":         filterUpper,
		"lower":         filterLower,
		"capitalize":    filterCapitalize,
		"title":         filterTitle,
		"trim":          filterTrim,
		"ltrim":         filterLtrim,
		"rtrim":         filterRtrim,
		"striptags":     filterStriptags,
		"e":             filterEscape,
		"escape":        filterEscape,
		"safe":          filterSafe,
		"forceescape":   filterForceEscape,
		"urlencode":     filterURLEncode,
		"urlize":        filterUrlize,
		"wordcount":     filterWordcount,
		"truncate":      filterTruncate,
		"wordwrap":      filterWordwrap,
		"center":        filterCenter,
		"ljust":         filterLJust,
		"rjust":         filterRJust,
		"zfill":         filterZFill,
		"indent":        filterIndent,
		"nl2br":         filterNl2Br,
		"reverse":       filterReverse,
		"replace":       filterReplace,
		"format":        filterFormat,
		"join":          filterJoin,
		"first":         filterFirst,
		"last":          filterLast,
		"list":          filterList,
		"dict":          filterDict,
		"length":        filterLength,
		"count":         filterLength,
		"abs":           filterAbs,
		"round":         filterRound,
		"int":           filterInt,
		"float":         filterFloat,
		"string":        filterString,
		"batch":         filterBatch,
		"slice":         filterSlice,
		"sort":          filterSort,
		"unique":        filterUnique,
		"min":           filterMin,
		"max":           filterMax,
		"sum":           filterSum,
		"map":           filterMap,
		"select":        filterSelect,
		"reject":        filterReject,
		"selectattr":    filterSelectAttr,
		"rejectattr":    filterRejectAttr,
		"attr":          filterAttr,
		"items":         filterItems,
		"json":          filterJSON,
		"tojson":        filterJSON,
		"pprint":        filterPPrint,
		"default":       filterDefault,
		"d":             filterDefault,
		"bool":          filterBool,
		"filesizeformat": filterFilesizeFormat,
		"date":          filterDate,
		"datetime":      filterDate,
		"yesno":         filterYesNo,
		"random":        filterRandom,
		"groupby":       filterGroupBy,
		"xmlattr":       filterXMLAttr,
	}

	for name, fn := range filters {
		env.filterFuncs[name] = fn
	}
}

// --- String Filters ---

func filterUpper(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return strings.ToUpper(stringify(v)), nil
}

func filterLower(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return strings.ToLower(stringify(v)), nil
}

func filterCapitalize(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	if len(s) == 0 {
		return s, nil
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:]), nil
}

func filterTitle(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return strings.Title(stringify(v)), nil
}

func filterTrim(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	chars := ""
	if len(args) > 0 {
		chars = stringify(args[0])
	}
	if chars != "" {
		return strings.Trim(s, chars), nil
	}
	return strings.TrimSpace(s), nil
}

func filterLtrim(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	chars := ""
	if len(args) > 0 {
		chars = stringify(args[0])
	}
	if chars != "" {
		return strings.TrimLeft(s, chars), nil
	}
	return strings.TrimLeft(s, " \t\n\r"), nil
}

func filterRtrim(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	chars := ""
	if len(args) > 0 {
		chars = stringify(args[0])
	}
	if chars != "" {
		return strings.TrimRight(s, chars), nil
	}
	return strings.TrimRight(s, " \t\n\r"), nil
}

func filterStriptags(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	// Simple HTML tag stripping
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return strings.TrimSpace(s), nil
}

func filterEscape(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	if IsSafe(v) {
		return v, nil
	}
	return SafeString(escapeHTML(s)), nil
}

func filterForceEscape(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return SafeString(escapeHTML(stringify(v))), nil
}

func filterSafe(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return SafeString(stringify(v)), nil
}

func escapeHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func filterURLEncode(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	return url.QueryEscape(s), nil
}

func filterUrlize(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	// Basic URL detection and linking
	re := regexp.MustCompile(`(https?://[^\s<>"']+|www\.[^\s<>"']+)`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		href := match
		if strings.HasPrefix(href, "www.") {
			href = "http://" + href
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, href, match)
	}), nil
}

func filterWordcount(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	words := strings.Fields(s)
	return len(words), nil
}

func filterTruncate(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	length := 255
	killwords := false
	end := "..."

	if len(args) > 0 {
		length = toInt(args[0])
	}
	if len(args) > 1 {
		killwords = isTruthy(args[1])
	}
	if len(args) > 2 {
		end = stringify(args[2])
	}

	if len(s) <= length {
		return s, nil
	}

	if killwords {
		if len(s) > length {
			return s[:length] + end, nil
		}
	}

	// Don't cut words
	if length > len(end) {
		result := s[:length-len(end)]
		// Find last space
		idx := strings.LastIndex(result, " ")
		if idx > 0 {
			result = result[:idx]
		}
		return result + end, nil
	}

	return s[:length] + end, nil
}

func filterWordwrap(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	width := 79
	breakLongWords := true

	if len(args) > 0 {
		width = toInt(args[0])
	}
	if len(args) > 1 {
		breakLongWords = isTruthy(args[1])
	}

	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if len(line) <= width {
			lines = append(lines, line)
			continue
		}
		for len(line) > width {
			idx := width
			if breakLongWords || idx < len(line) {
				spaceIdx := strings.LastIndex(line[:idx], " ")
				if spaceIdx > 0 {
					idx = spaceIdx
				}
			}
			lines = append(lines, line[:idx])
			line = strings.TrimLeft(line[idx:], " ")
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func filterCenter(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	width := 80
	if len(args) > 0 {
		width = toInt(args[0])
	}
	if len(s) >= width {
		return s, nil
	}
	padding := width - len(s)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right), nil
}

func filterLJust(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	width := 80
	if len(args) > 0 {
		width = toInt(args[0])
	}
	if len(s) >= width {
		return s, nil
	}
	return s + strings.Repeat(" ", width-len(s)), nil
}

func filterRJust(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	width := 80
	if len(args) > 0 {
		width = toInt(args[0])
	}
	if len(s) >= width {
		return s, nil
	}
	return strings.Repeat(" ", width-len(s)) + s, nil
}

func filterZFill(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	width := 0
	if len(args) > 0 {
		width = toInt(args[0])
	}
	if len(s) >= width {
		return s, nil
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	pad := width - len(s)
	if neg {
		pad--
	}
	s = strings.Repeat("0", pad) + s
	if neg {
		s = "-" + s
	}
	return s, nil
}

func filterIndent(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	indent := "    "
	first := false
	if len(args) > 0 {
		indent = stringify(args[0])
	}
	if len(args) > 1 {
		first = isTruthy(args[1])
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i == 0 && !first {
			continue
		}
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n"), nil
}

func filterNl2Br(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	return SafeString(strings.ReplaceAll(s, "\n", "<br>\n")), nil
}

func filterReverse(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		s := stringify(v)
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	}
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[len(items)-1-i] = item
	}
	return result, nil
}

func filterReplace(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	s := stringify(v)
	if len(args) < 2 {
		return s, nil
	}
	old := stringify(args[0])
	new := stringify(args[1])
	count := -1
	if len(args) > 2 {
		count = toInt(args[2])
	}
	if count >= 0 {
		return strings.Replace(s, old, new, count), nil
	}
	return strings.ReplaceAll(s, old, new), nil
}

func filterFormat(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	fmtStr := stringify(v)
	// Go-style formatting (simplified from Python's % formatting)
	// Try to use Go fmt.Sprintf
	goArgs := make([]interface{}, len(args))
	for i, a := range args {
		goArgs[i] = a
	}
	// Convert {{}} style to Go-style (simplified)
	return fmt.Sprintf(fmtStr, goArgs...), nil
}

// --- Collection Filters ---

func filterJoin(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return "", nil
	}
	sep := ""
	if len(args) > 0 {
		sep = stringify(args[0])
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = stringify(item)
	}
	return strings.Join(parts, sep), nil
}

func filterFirst(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil || len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func filterLast(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil || len(items) == 0 {
		return nil, nil
	}
	return items[len(items)-1], nil
}

func filterList(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return []interface{}{}, nil
	}
	return items, nil
}

func filterDict(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	m := make(map[string]interface{})
	for _, item := range items {
		switch kv := item.(type) {
		case [2]interface{}:
			m[fmt.Sprintf("%v", kv[0])] = kv[1]
		case mapEntry:
			m[fmt.Sprintf("%v", kv.key)] = kv.value
		}
	}
	return m, nil
}

func filterLength(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return 0, nil
	}
	return len(items), nil
}

func filterBatch(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}
	size := 1
	if len(args) > 0 {
		size = toInt(args[0])
	}
	if size <= 0 {
		return items, nil
	}

	var result [][]interface{}
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		result = append(result, items[i:end])
	}

	// Convert to interface slice
	final := make([]interface{}, len(result))
	for i, batch := range result {
		final[i] = batch
	}
	return final, nil
}

func filterSlice(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}
	slices := 1
	fillWith := interface{}(nil)

	if len(args) > 0 {
		slices = toInt(args[0])
	}
	if len(args) > 1 {
		fillWith = args[1]
	}

	if slices <= 0 {
		return items, nil
	}

	itemsPerSlice := int(math.Ceil(float64(len(items)) / float64(slices)))
	result := make([]interface{}, slices)

	for i := 0; i < slices; i++ {
		start := i * itemsPerSlice
		end := start + itemsPerSlice
		if end > len(items) {
			end = len(items)
		}
		if start < end {
			result[i] = items[start:end]
		} else if fillWith != nil {
			slice_ := make([]interface{}, itemsPerSlice)
			for j := range slice_ {
				slice_[j] = fillWith
			}
			result[i] = slice_
		}
	}

	return result, nil
}

func filterSort(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}

	reverse := false
	caseSensitive := true
	attribute := ""
	if len(args) > 0 {
		attribute = stringify(args[0])
	}
	if len(args) > 1 {
		reverse = isTruthy(args[1])
	}
	if len(args) > 2 {
		caseSensitive = isTruthy(args[2])
	}

	// Create a copy
	result := make([]interface{}, len(items))
	copy(result, items)

	sort.SliceStable(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		if attribute != "" {
			a = getattr(a, attribute)
			b = getattr(b, attribute)
		}

		as_ := fmt.Sprintf("%v", a)
		bs := fmt.Sprintf("%v", b)

		if !caseSensitive {
			as_ = strings.ToLower(as_)
			bs = strings.ToLower(bs)
		}

		cmp := strings.Compare(as_, bs)
		if reverse {
			return cmp > 0
		}
		return cmp < 0
	})

	return result, nil
}

func filterUnique(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}

	attribute := ""
	caseSensitive := true
	if len(args) > 0 {
		attribute = stringify(args[0])
	}
	if len(args) > 1 {
		caseSensitive = isTruthy(args[1])
	}

	seen := make(map[string]bool)
	var result []interface{}

	for _, item := range items {
		key := fmt.Sprintf("%v", item)
		if attribute != "" {
			key = fmt.Sprintf("%v", getattr(item, attribute))
		}
		if !caseSensitive {
			key = strings.ToLower(key)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result, nil
}

func filterMin(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil || len(items) == 0 {
		return nil, nil
	}

	minItem := items[0]
	minKey := toFloat(minItem)

	for _, item := range items[1:] {
		key := toFloat(item)
		if key < minKey {
			minItem = item
			minKey = key
		}
	}
	return minItem, nil
}

func filterMax(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil || len(items) == 0 {
		return nil, nil
	}

	maxItem := items[0]
	maxKey := toFloat(maxItem)

	for _, item := range items[1:] {
		key := toFloat(item)
		if key > maxKey {
			maxItem = item
			maxKey = key
		}
	}
	return maxItem, nil
}

func filterSum(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return 0, nil
	}

	attribute := ""
	start := 0
	if len(args) > 0 {
		attribute = stringify(args[0])
	}
	if len(args) > 1 {
		start = toInt(args[1])
	}

	sum := float64(start)
	for _, item := range items {
		var val interface{} = item
		if attribute != "" {
			val = getattr(item, attribute)
		}
		sum += toFloat(val)
	}

	return sum, nil
}

func filterMap(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}

	attribute := ""
	if len(args) > 0 {
		attribute = stringify(args[0])
	}

	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = getattr(item, attribute)
	}
	return result, nil
}

func filterSelect(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}

	testName := ""
	testArg := ""
	if len(args) > 0 {
		testName = stringify(args[0])
	}
	if len(args) > 1 {
		testArg = stringify(args[1])
	}

	var result []interface{}
	for _, item := range items {
		if filterTestItem(ctx, testName, item, testArg) {
			result = append(result, item)
		}
	}
	return result, nil
}

func filterReject(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}

	testName := ""
	testArg := ""
	if len(args) > 0 {
		testName = stringify(args[0])
	}
	if len(args) > 1 {
		testArg = stringify(args[1])
	}

	var result []interface{}
	for _, item := range items {
		if !filterTestItem(ctx, testName, item, testArg) {
			result = append(result, item)
		}
	}
	return result, nil
}

func filterSelectAttr(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}
	if len(args) < 2 {
		return items, nil
	}
	attr := stringify(args[0])
	testName := stringify(args[1])
	testArg := ""
	if len(args) > 2 {
		testArg = stringify(args[2])
	}

	var result []interface{}
	for _, item := range items {
		val := getattr(item, attr)
		if filterTestItem(ctx, testName, val, testArg) {
			result = append(result, item)
		}
	}
	return result, nil
}

func filterRejectAttr(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}
	if len(args) < 2 {
		return items, nil
	}
	attr := stringify(args[0])
	testName := stringify(args[1])
	testArg := ""
	if len(args) > 2 {
		testArg = stringify(args[2])
	}

	var result []interface{}
	for _, item := range items {
		val := getattr(item, attr)
		if !filterTestItem(ctx, testName, val, testArg) {
			result = append(result, item)
		}
	}
	return result, nil
}

func filterTestItem(ctx *ExecutionContext, testName string, item interface{}, testArg string) bool {
	if ctx == nil || ctx.env == nil {
		return isTruthy(item)
	}
	fn, ok := ctx.env.GetTest(testName)
	if !ok {
		return isTruthy(item)
	}
	var args []interface{}
	if testArg != "" {
		args = []interface{}{testArg}
	}
	result, err := fn(ctx, item, args...)
	if err != nil {
		return false
	}
	return result
}

func filterAttr(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return v, nil
	}
	attr := stringify(args[0])
	return getattr(v, attr), nil
}


func filterItems(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	items := make([]interface{}, 0, len(m))
	for k, v := range m {
		items = append(items, [2]interface{}{k, v})
	}
	return items, nil
}

// --- Numeric Filters ---

func filterAbs(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	f := toFloat(v)
	if isInt(v) {
		i := toInt(v)
		if i < 0 {
			return -i, nil
		}
		return i, nil
	}
	if f < 0 {
		return -f, nil
	}
	return f, nil
}

func filterRound(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	f := toFloat(v)
	precision := 0
	method := "common"
	if len(args) > 0 {
		precision = toInt(args[0])
	}
	if len(args) > 1 {
		method = stringify(args[1])
	}

	pow := math.Pow(10, float64(precision))
	result := f * pow

	switch method {
	case "ceil":
		result = math.Ceil(result)
	case "floor":
		result = math.Floor(result)
	default:
		result = math.Round(result)
	}

	return result / pow, nil
}

func filterInt(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	default_ := 0
	if len(args) > 0 {
		default_ = toInt(args[0])
	}
	switch val := v.(type) {
	case int:
		return val, nil
	case float64:
		return int(val), nil
	case string:
		i, err := parseInt(val)
		if err != nil {
			return default_, nil
		}
		return i, nil
	}
	return default_, nil
}

func filterFloat(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	default_ := 0.0
	if len(args) > 0 {
		default_ = toFloat(args[0])
	}
	f := toFloat(v)
	if v == nil {
		return default_, nil
	}
	return f, nil
}

func filterString(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return stringify(v), nil
}

func filterBool(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	return isTruthy(v), nil
}

// --- Default ---

func filterDefault(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	default_ := interface{}("")
	if len(args) > 0 {
		default_ = args[0]
	}

	if IsUndefined(v) {
		return default_, nil
	}
	return v, nil
}

// --- JSON ---

func filterJSON(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}", nil
	}
	return SafeString(string(data)), nil
}

func filterPPrint(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return stringify(v), nil
	}
	return string(data), nil
}

// --- File Size ---

func filterFilesizeFormat(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	bytes := toFloat(v)
	if bytes == 0 {
		return "0 B", nil
	}

	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for bytes >= 1024 && i < len(suffixes)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", bytes, suffixes[i]), nil
}

// --- Date ---

func filterDate(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	format := "Jan 02, 2006"
	if len(args) > 0 {
		format = stringify(args[0])
	}

	var t time.Time
	switch val := v.(type) {
	case time.Time:
		t = val
	case string:
		// Try to parse as RFC3339
		parsed, err := time.Parse(time.RFC3339, val)
		if err != nil {
			// Try Unix timestamp
			ts := toInt(v)
			if ts > 0 {
				t = time.Unix(int64(ts), 0)
			} else {
				return val, nil
			}
		} else {
			t = parsed
		}
	case int:
		t = time.Unix(int64(val), 0)
	case int64:
		t = time.Unix(val, 0)
	case float64:
		t = time.Unix(int64(val), 0)
	default:
		return stringify(v), nil
	}

	// Convert Python-style format to Go
	format = pythonToGoDateFormat(format)
	return t.Format(format), nil
}

func pythonToGoDateFormat(s string) string {
	// Basic conversion of Python date format to Go
	repl := map[string]string{
		"%Y": "2006",
		"%y": "06",
		"%m": "01",
		"%d": "02",
		"%H": "15",
		"%I": "03",
		"%M": "04",
		"%S": "05",
		"%p": "PM",
		"%A": "Monday",
		"%a": "Mon",
		"%B": "January",
		"%b": "Jan",
		"%f": "000000",
		"%Z": "MST",
		"%z": "-0700",
		"%j": "002",
		"%U": "00",
		"%W": "00",
		"%c": "Mon Jan 2 15:04:05 2006",
		"%x": "01/02/06",
		"%X": "15:04:05",
	}
	for py, go_ := range repl {
		s = strings.ReplaceAll(s, py, go_)
	}
	// Handle %% as %
	s = strings.ReplaceAll(s, "%%", "%")
	return s
}

// --- YesNo ---

func filterYesNo(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	yesStr := "yes"
	noStr := "no"
	maybeStr := "maybe"

	if len(args) > 0 {
		parts := strings.Split(stringify(args[0]), ",")
		yesStr = parts[0]
		if len(parts) > 1 {
			noStr = parts[1]
		}
		if len(parts) > 2 {
			maybeStr = parts[2]
		}
	}

	if IsUndefined(v) || v == nil {
		return maybeStr, nil
	}
	if isTruthy(v) {
		return yesStr, nil
	}
	return noStr, nil
}

// --- Random ---

func filterRandom(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil || len(items) == 0 {
		return nil, nil
	}
	return items[rand.Intn(len(items))], nil
}

// --- GroupBy ---

func filterGroupBy(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	items := toSlice(v)
	if items == nil {
		return nil, nil
	}
	attribute := ""
	if len(args) > 0 {
		attribute = stringify(args[0])
	}

	groups := make(map[string][]interface{})
	var order []string

	for _, item := range items {
		key := fmt.Sprintf("%v", item)
		if attribute != "" {
			key = fmt.Sprintf("%v", getattr(item, attribute))
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], item)
	}

	result := make([]interface{}, len(order))
	for i, key := range order {
		result[i] = map[string]interface{}{
			"grouper": key,
			"list":   groups[key],
		}
	}
	return result, nil
}

// --- XMLAttr ---

func filterXMLAttr(ctx *ExecutionContext, v interface{}, args ...interface{}) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return "", nil
	}
	var parts []string
	for k, v := range m {
		if v == nil {
			continue
		}
		sv := stringify(v)
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, escapeHTML(sv)))
	}
	sort.Strings(parts)
	if len(parts) > 0 {
		return " " + strings.Join(parts, " "), nil
	}
	return "", nil
}

// --- Utility ---

func getattr(v interface{}, attr string) interface{} {
	if v == nil {
		return nil
	}

	if m, ok := v.(map[string]interface{}); ok {
		if val, exists := m[attr]; exists {
			return val
		}
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		fv := rv.FieldByName(attr)
		if fv.IsValid() {
			return fv.Interface()
		}
		t := rv.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("jinja")
			if tag == "" {
				tag = field.Tag.Get("json")
			}
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] == attr {
				return rv.Field(i).Interface()
			}
		}
	}

	return nil
}

func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	result := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not a number")
		}
		result = result*10 + int(ch-'0')
	}
	if negative {
		return -result, nil
	}
	return result, nil
}
