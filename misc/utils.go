package misc

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"math/rand"
	"mime/multipart"
	"os"
	"runtime"
	"strings"
	"time"
)

func Shuffle(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)

	for i := range out {
		j := rand.Intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}

	return out
}

func Contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}

type MonthRange struct {
	First time.Time
	Last  time.Time
}

func (mr *MonthRange) Format(layout, sep string, thisMonthAsCurrent bool) string {
	var (
		first = mr.First.Format(layout)
		last  = mr.Last.Format(layout)
	)
	if thisMonthAsCurrent {
		y, m, _ := time.Now().UTC().Date()
		if ly, lm, _ := mr.Last.Date(); ly == y && lm == m {
			last = "Current"
		}
	}
	return first + sep + last
}

func MonthRangeSince(ts int64) (out []*MonthRange) {
	var (
		t   = time.Unix(ts, 0).UTC()
		now = time.Now().UTC()
	)
	for !t.After(now) {
		out = append(out, FirstAndLast(t))
		t = t.AddDate(0, 1, 0)
	}
	return
}

func FirstAndLast(t time.Time) *MonthRange {
	var (
		y, m, _ = t.Date()
		first   = time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
	)
	return &MonthRange{first, first.AddDate(0, 1, -1)}
}

func AddPrefix(pre string, ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		if !strings.HasPrefix(s, pre) {
			out[i] = pre + s
		}
	}
	return out
}

func StripPrefix(pre string, ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		if strings.HasPrefix(s, pre) {
			out[i] = s[len(pre):]
		}
	}
	return out
}

func TrimSlice(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}
	m := make(map[string]bool, len(ss))
	for _, v := range ss {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		m[v] = true
	}
	ss = ss[:0]
	for k := range m {
		ss = append(ss, k)
	}
	return ss
}

func Dump(v interface{}) {
	const sep = "-------------------------"
	_, fn, ln, _ := runtime.Caller(1)
	j, _ := json.MarshalIndent(v, "", "\t")
	log.Printf("\n- %s:%d:\n%s\n%s", fn, ln, j, sep)
}

func GetFormField(mf *multipart.Form, k, def string) string {
	if v := mf.Value[k]; len(v) > 0 {
		return v[0]
	}
	return def
}

func SaveUploadedFile(fh *multipart.FileHeader, fpath string) error {
	mf, err := fh.Open()
	if err != nil {
		return err
	}
	defer mf.Close()

	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, mf)
	f.Close()
	if err != nil {
		os.Remove(fpath)
	}
	return err
}
