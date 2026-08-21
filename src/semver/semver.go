package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Identifier struct {
	Raw     string
	Numeric bool
	Number  uint64
}

type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
	Pre   []Identifier
	Build []string
	Dev   bool
}

func Dev() Version { return Version{Dev: true} }

func ParseBuild(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "dev" {
		return Dev(), nil
	}
	return Parse(strings.TrimPrefix(s, "v"))
}

func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

func Parse(s string) (Version, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	if s == "" {
		return Version{}, errors.New("semantic version is empty")
	}
	var v Version
	coreAndPre, build, hasBuild := strings.Cut(s, "+")
	if hasBuild {
		if build == "" {
			return v, errors.New("empty build metadata")
		}
		parts := strings.Split(build, ".")
		for _, p := range parts {
			if !validIdentifier(p, false) {
				return v, fmt.Errorf("invalid build identifier %q", p)
			}
		}
		v.Build = parts
	}
	core, pre, hasPre := strings.Cut(coreAndPre, "-")
	nums := strings.Split(core, ".")
	if len(nums) != 3 {
		return v, fmt.Errorf("semantic version %q must have major.minor.patch", s)
	}
	vals := []*uint64{&v.Major, &v.Minor, &v.Patch}
	for i, p := range nums {
		n, err := parseCoreNumber(p)
		if err != nil {
			return v, err
		}
		*vals[i] = n
	}
	if hasPre {
		if pre == "" {
			return v, errors.New("empty prerelease")
		}
		for _, p := range strings.Split(pre, ".") {
			if !validIdentifier(p, true) {
				return v, fmt.Errorf("invalid prerelease identifier %q", p)
			}
			id := Identifier{Raw: p}
			if allDigits(p) {
				if len(p) > 1 && p[0] == '0' {
					return v, fmt.Errorf("numeric prerelease identifier %q has leading zero", p)
				}
				n, _ := strconv.ParseUint(p, 10, 64)
				id.Numeric, id.Number = true, n
			}
			v.Pre = append(v.Pre, id)
		}
	}
	return v, nil
}

func parseCoreNumber(s string) (uint64, error) {
	if s == "" || !allDigits(s) {
		return 0, fmt.Errorf("invalid numeric version component %q", s)
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, fmt.Errorf("numeric version component %q has leading zero", s)
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func validIdentifier(s string, prerelease bool) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') || r > unicode.MaxASCII {
			return false
		}
	}
	return true
}
func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func (v Version) String() string {
	if v.Dev {
		return "dev"
	}
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		xs := make([]string, len(v.Pre))
		for i, p := range v.Pre {
			xs[i] = p.Raw
		}
		s += "-" + strings.Join(xs, ".")
	}
	if len(v.Build) > 0 {
		s += "+" + strings.Join(v.Build, ".")
	}
	return s
}
func (v Version) VString() string {
	if v.Dev {
		return "dev"
	}
	return "v" + v.String()
}

func (v Version) Compare(o Version) int {
	if v.Dev || o.Dev {
		if v.Dev == o.Dev {
			return 0
		}
		if v.Dev {
			return -1
		}
		return 1
	}
	if v.Major != o.Major {
		return cmp(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return cmp(v.Minor, o.Minor)
	}
	if v.Patch != o.Patch {
		return cmp(v.Patch, o.Patch)
	}
	if len(v.Pre) == 0 && len(o.Pre) == 0 {
		return 0
	}
	if len(v.Pre) == 0 {
		return 1
	}
	if len(o.Pre) == 0 {
		return -1
	}
	n := len(v.Pre)
	if len(o.Pre) < n {
		n = len(o.Pre)
	}
	for i := 0; i < n; i++ {
		a, b := v.Pre[i], o.Pre[i]
		if a.Raw == b.Raw {
			continue
		}
		if a.Numeric && b.Numeric {
			return cmp(a.Number, b.Number)
		}
		if a.Numeric != b.Numeric {
			if a.Numeric {
				return -1
			}
			return 1
		}
		if a.Raw < b.Raw {
			return -1
		}
		return 1
	}
	if len(v.Pre) < len(o.Pre) {
		return -1
	}
	if len(v.Pre) > len(o.Pre) {
		return 1
	}
	return 0
}
func cmp(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
