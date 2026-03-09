// Package filter implements a query language for filtering apt packages.
//
// Syntax examples:
//
//	section:utils          section contains "utils"
//	arch:amd64             architecture equals "amd64"
//	size>10MB              size greater than 10 MB
//	size<5MB               size less than 5 MB
//	size>=100kB            size >= 100 kB
//	installed              only installed packages
//	!installed             only not-installed packages
//	upgradable             only upgradable packages
//	name:vim               name contains "vim"
//	ver:2.0                version contains "2.0"
//	desc:editor            description contains "editor"
//
// Multiple tokens are ANDed together.
package filter

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// SortColumn represents which column to sort by.
type SortColumn int

// SortInfo holds the current sorting state for display purposes.
type SortInfo struct {
	Column SortColumn
	Desc   bool
}

const (
	SortNone SortColumn = iota
	SortName
	SortVersion
	SortSize
	SortSection
	SortArchitecture
)

// SizeOp represents a comparison operator for size filters.
type SizeOp int

const (
	SizeGt SizeOp = iota
	SizeLt
	SizeGe
	SizeLe
	SizeEq
)

// SizeFilter holds a parsed size comparison.
type SizeFilter struct {
	Op SizeOp
	KB int64 // size in kB
}

// Filter represents a parsed set of filter criteria.
type Filter struct {
	Section      string // contains (case-insensitive)
	Architecture string // exact (case-insensitive)
	Size         *SizeFilter
	Installed    *bool
	Upgradable   *bool
	Name         string     // contains (case-insensitive)
	Version      string     // contains (case-insensitive)
	Description  string     // contains (case-insensitive)
	OrderBy      SortColumn // column to sort by
	OrderDesc    bool       // true for descending order
}

// IsEmpty returns true if no filter criteria are set.
func (f Filter) IsEmpty() bool {
	return f.Section == "" &&
		f.Architecture == "" &&
		f.Size == nil &&
		f.Installed == nil &&
		f.Upgradable == nil &&
		f.Name == "" &&
		f.Version == "" &&
		f.Description == "" &&
		f.OrderBy == SortNone
}

// NeedsMetadata returns true if the filter uses fields (Section, Architecture, Size)
// that require package metadata from apt-cache show.
func (f Filter) NeedsMetadata() bool {
	return f.Section != "" || f.Architecture != "" || f.Size != nil
}

// PackageData is the minimal interface a package must expose for filtering.
type PackageData struct {
	Name         string
	Version      string
	NewVersion   string
	Size         string // formatted, e.g. "1.5 MB"
	Description  string
	Installed    bool
	Upgradable   bool
	Section      string
	Architecture string
}

// Match returns true if the package satisfies all filter criteria.
func (f Filter) Match(p PackageData) bool {
	if !f.matchNonMetadata(p) {
		return false
	}
	return f.matchMetadata(p)
}

// MatchWithoutMetadata returns true if the package satisfies all non-metadata
// filter criteria (name, version, description, installed, upgradable).
// This is used to narrow candidates before loading metadata.
func (f Filter) MatchWithoutMetadata(p PackageData) bool {
	return f.matchNonMetadata(p)
}

func (f Filter) matchNonMetadata(p PackageData) bool {
	if f.Name != "" && !containsFold(p.Name, f.Name) {
		return false
	}
	if f.Version != "" {
		v := p.Version
		if p.NewVersion != "" {
			v = p.NewVersion
		}
		if !containsFold(v, f.Version) {
			return false
		}
	}
	if f.Description != "" && !containsFold(p.Description, f.Description) {
		return false
	}
	if f.Installed != nil {
		if *f.Installed != p.Installed {
			return false
		}
	}
	if f.Upgradable != nil {
		if *f.Upgradable != p.Upgradable {
			return false
		}
	}
	return true
}

func (f Filter) matchMetadata(p PackageData) bool {
	if f.Section != "" && !containsFold(p.Section, f.Section) {
		return false
	}
	if f.Architecture != "" && !strings.EqualFold(p.Architecture, f.Architecture) {
		return false
	}
	if f.Size != nil {
		pkgKB := ParseSizeToKB(p.Size)
		if pkgKB <= 0 {
			return false
		}
		switch f.Size.Op {
		case SizeGt:
			if pkgKB <= f.Size.KB {
				return false
			}
		case SizeLt:
			if pkgKB >= f.Size.KB {
				return false
			}
		case SizeGe:
			if pkgKB < f.Size.KB {
				return false
			}
		case SizeLe:
			if pkgKB > f.Size.KB {
				return false
			}
		case SizeEq:
			if pkgKB != f.Size.KB {
				return false
			}
		}
	}
	return true
}

// Parse parses a filter query string into a Filter.
func Parse(query string) Filter {
	var f Filter
	tokens := tokenize(query)
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}

		lower := strings.ToLower(tok)

		// Boolean flags
		if lower == "installed" {
			b := true
			f.Installed = &b
			continue
		}
		if lower == "!installed" {
			b := false
			f.Installed = &b
			continue
		}
		if lower == "upgradable" {
			b := true
			f.Upgradable = &b
			continue
		}
		if lower == "!upgradable" {
			b := false
			f.Upgradable = &b
			continue
		}

		// Size filters: size>10MB, size<5MB, size>=100kB, size<=1GB, size=500kB
		if strings.HasPrefix(lower, "size") {
			rest := tok[4:]
			if sf := parseSizeExpr(rest); sf != nil {
				f.Size = sf
				continue
			}
		}

		// Sorting: order:column or order:column:desc
		if strings.HasPrefix(lower, "order:") {
			orderExpr := tok[6:]
			parts := strings.SplitN(orderExpr, ":", 2)
			if len(parts) >= 1 {
				f.OrderBy = parseSortColumn(parts[0])
				if len(parts) == 2 {
					f.OrderDesc = strings.EqualFold(parts[1], "desc")
				}
			}
			continue
		}

		// key:value filters
		if idx := strings.Index(tok, ":"); idx > 0 {
			key := strings.ToLower(tok[:idx])
			val := tok[idx+1:]
			switch key {
			case "section", "sec":
				f.Section = val
			case "arch", "architecture":
				f.Architecture = val
			case "name":
				f.Name = val
			case "ver", "version":
				f.Version = val
			case "desc", "description":
				f.Description = val
			case "size":
				// size:>10MB variant
				if sf := parseSizeExpr(val); sf != nil {
					f.Size = sf
				}
			}
			continue
		}
	}
	return f
}

// Describe returns a human-readable summary of active filters.
func (f Filter) Describe() string {
	var parts []string
	if f.Section != "" {
		parts = append(parts, "sec:"+f.Section)
	}
	if f.Architecture != "" {
		parts = append(parts, "arch:"+f.Architecture)
	}
	if f.Name != "" {
		parts = append(parts, "name:"+f.Name)
	}
	if f.Version != "" {
		parts = append(parts, "ver:"+f.Version)
	}
	if f.Description != "" {
		parts = append(parts, "desc:"+f.Description)
	}
	if f.Installed != nil {
		if *f.Installed {
			parts = append(parts, "installed")
		} else {
			parts = append(parts, "!installed")
		}
	}
	if f.Upgradable != nil {
		if *f.Upgradable {
			parts = append(parts, "upgradable")
		} else {
			parts = append(parts, "!upgradable")
		}
	}
	if f.Size != nil {
		opStr := ">"
		switch f.Size.Op {
		case SizeLt:
			opStr = "<"
		case SizeGe:
			opStr = ">="
		case SizeLe:
			opStr = "<="
		case SizeEq:
			opStr = "="
		}
		parts = append(parts, "size"+opStr+formatKB(f.Size.KB))
	}
	if f.OrderBy != SortNone {
		dir := "asc"
		if f.OrderDesc {
			dir = "desc"
		}
		parts = append(parts, "order:"+sortColumnName(f.OrderBy)+":"+dir)
	}
	return strings.Join(parts, " ")
}

// HelpText returns the syntax help shown in the filter bar.
func HelpText() string {
	return "section: arch: size>|<|= name: ver: desc: installed !installed upgradable order:column:asc|desc"
}

// parseSizeExpr parses a size expression like ">10MB", ">=100kB", "<5GB", "=500kB".
func parseSizeExpr(s string) *SizeFilter {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var op SizeOp
	var rest string

	if strings.HasPrefix(s, ">=") {
		op = SizeGe
		rest = s[2:]
	} else if strings.HasPrefix(s, "<=") {
		op = SizeLe
		rest = s[2:]
	} else if strings.HasPrefix(s, ">") {
		op = SizeGt
		rest = s[1:]
	} else if strings.HasPrefix(s, "<") {
		op = SizeLt
		rest = s[1:]
	} else if strings.HasPrefix(s, "=") {
		op = SizeEq
		rest = s[1:]
	} else {
		// try parsing as ">value" where the whole thing is the number+unit
		return nil
	}

	kb := parseValueToKB(rest)
	if kb < 0 {
		return nil
	}
	return &SizeFilter{Op: op, KB: kb}
}

// parseValueToKB parses "10MB", "5GB", "100kB", "500" (assumed kB) into kB.
func parseValueToKB(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1
	}

	// Split into number and unit
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return -1
	}

	numStr := s[:i]
	unit := strings.TrimSpace(strings.ToLower(s[i:]))

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return -1
	}

	switch unit {
	case "gb", "g":
		return int64(num * 1024 * 1024)
	case "mb", "m":
		return int64(num * 1024)
	case "kb", "k", "":
		return int64(num)
	case "b":
		return int64(num / 1024)
	default:
		return -1
	}
}

// ParseSizeToKB parses a formatted size string ("1.5 MB", "324 kB", "2.1 GB") back to kB.
func ParseSizeToKB(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	return parseValueToKB(strings.ReplaceAll(s, " ", ""))
}

func formatKB(kb int64) string {
	switch {
	case kb >= 1024*1024:
		return strconv.FormatFloat(float64(kb)/(1024*1024), 'f', 1, 64) + "GB"
	case kb >= 1024:
		return strconv.FormatFloat(float64(kb)/1024, 'f', 1, 64) + "MB"
	default:
		return strconv.FormatInt(kb, 10) + "kB"
	}
}

// tokenize splits a query into tokens, respecting quoted strings.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		if inQuote {
			if r == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(r)
			}
		} else if r == '"' || r == '\'' {
			inQuote = true
			quoteChar = r
		} else if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func parseSortColumn(s string) SortColumn {
	switch strings.ToLower(s) {
	case "name":
		return SortName
	case "version", "ver":
		return SortVersion
	case "size":
		return SortSize
	case "section", "sec":
		return SortSection
	case "arch", "architecture":
		return SortArchitecture
	default:
		return SortNone
	}
}

func sortColumnName(c SortColumn) string {
	switch c {
	case SortName:
		return "name"
	case SortVersion:
		return "version"
	case SortSize:
		return "size"
	case SortSection:
		return "section"
	case SortArchitecture:
		return "architecture"
	default:
		return ""
	}
}

// SortColumnLabel returns a human-friendly label for a SortColumn.
func SortColumnLabel(c SortColumn) string {
	return sortColumnName(c)
}

// Sort sorts a slice of PackageData in place according to the filter's OrderBy and OrderDesc.
// Packages with unknown/empty values for the sort field are pushed to the end.
func Sort(pkgs []PackageData, f Filter) {
	if f.OrderBy == SortNone {
		return
	}
	sort.SliceStable(pkgs, func(i, j int) bool {
		iEmpty := pdFieldEmpty(pkgs[i], f.OrderBy)
		jEmpty := pdFieldEmpty(pkgs[j], f.OrderBy)
		if iEmpty != jEmpty {
			return !iEmpty
		}
		if iEmpty && jEmpty {
			return false
		}

		var less bool
		switch f.OrderBy {
		case SortName:
			less = strings.ToLower(pkgs[i].Name) < strings.ToLower(pkgs[j].Name)
		case SortVersion:
			less = pkgs[i].Version < pkgs[j].Version
		case SortSize:
			less = ParseSizeToKB(pkgs[i].Size) < ParseSizeToKB(pkgs[j].Size)
		case SortSection:
			less = strings.ToLower(pkgs[i].Section) < strings.ToLower(pkgs[j].Section)
		case SortArchitecture:
			less = strings.ToLower(pkgs[i].Architecture) < strings.ToLower(pkgs[j].Architecture)
		default:
			return false
		}
		if f.OrderDesc {
			return !less
		}
		return less
	})
}

func pdFieldEmpty(p PackageData, col SortColumn) bool {
	switch col {
	case SortName:
		return p.Name == ""
	case SortVersion:
		return p.Version == "" && p.NewVersion == ""
	case SortSize:
		return p.Size == "" || p.Size == "-"
	case SortSection:
		return p.Section == ""
	case SortArchitecture:
		return p.Architecture == ""
	default:
		return false
	}
}

// SortPackages sorts a slice of model-level packages using the filter's ordering.
func SortPackages(pkgs []PackageData, f Filter) {
	Sort(pkgs, f)
}
