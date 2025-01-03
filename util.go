package cloudcostexplorer

import (
	"bytes"
	"cmp"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"math/rand"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/dustin/go-humanize"
)

const DataSeparator = "|"

// FormatMoney formats a money value.
func FormatMoney(value float64) string {
	return fmt.Sprintf("$%s", humanize.CommafWithDigits(value, 2))
}

// Ptr returns a pointer to the passed value.
func Ptr[T any](v T) *T {
	return &v
}

// DefaultHash is the default hash function.
func DefaultHash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// SortMapByValue sorts a map by value.
func SortMapByValue[K comparable, V any](m map[K]V, cmp func(V, V) int, reverse bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		stableSorted := slices.SortedStableFunc(maps.Keys(m), func(a, b K) int {
			return cmp(m[a], m[b])
		})
		if reverse {
			slices.Reverse(stableSorted)
		}
		for _, k := range stableSorted {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}

// SortMapByValueOrdered sorts a map by value where the value is [cmp.Ordered].
func SortMapByValueOrdered[K comparable, V cmp.Ordered](m map[K]V, reverse bool) iter.Seq2[K, V] {
	return SortMapByValue(m, cmp.Compare[V], reverse)
}

// IndentJSON indents a string containing a JSON.
func IndentJSON(data string) string {
	if data == "" {
		return data
	}

	var ppsource bytes.Buffer
	err := json.Indent(&ppsource, []byte(data), "", "  ")
	if err != nil {
		return data
	}
	return ppsource.String()
}

// EllipticalTruncate truncates a string with ellipsis at the end.
func EllipticalTruncate(text string, maxLen int) string {
	lastSpaceIx := maxLen
	ln := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		ln++
		if ln > maxLen {
			return text[:lastSpaceIx] + "..."
		}
	}
	// If here, string is shorter or equal to maxLen
	return text
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandString returns a random string of length n.
func RandString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}
