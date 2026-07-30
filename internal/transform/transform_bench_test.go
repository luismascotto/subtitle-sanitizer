package transform

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/luismascotto/subtitle-sanitizer/internal/model"
	"github.com/luismascotto/subtitle-sanitizer/internal/rules"
	"github.com/luismascotto/subtitle-sanitizer/internal/subtitle"
)

func loadBenchDocument(tb testing.TB) model.Document {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "sub-example.ass"))
	if err != nil {
		tb.Fatalf("read sub-example.ass: %v", err)
	}
	doc, err := subtitle.ParseASS(data)
	if err != nil {
		tb.Fatalf("ParseASS: %v", err)
	}
	if len(doc.Cues) == 0 {
		tb.Fatal("no cues in sub-example.ass")
	}
	return *doc
}

func scaleDocument(doc model.Document, factor int) model.Document {
	if factor <= 1 {
		return doc
	}
	cues := make([]*model.Cue, 0, len(doc.Cues)*factor)
	idx := 1
	for range factor {
		for _, c := range doc.Cues {
			cp := *c
			cp.Index = idx
			idx++
			cues = append(cues, &cp)
		}
	}
	return model.Document{
		Format: doc.Format,
		Header: doc.Header,
		Cues:   cues,
	}
}

func TestApplyAll_sequentialMatchesParallel(t *testing.T) {
	base := loadBenchDocument(t)
	conf := rules.DefaultConfig()
	for _, factor := range []int{1, 4} {
		doc := scaleDocument(base, factor)
		seqOut, seqCh := ApplyAllSequential(doc, conf)
		parOut, parCh := ApplyAllParallel(doc, conf)
		if len(seqOut.Cues) != len(parOut.Cues) {
			t.Fatalf("factor=%d cue count seq=%d par=%d", factor, len(seqOut.Cues), len(parOut.Cues))
		}
		for i := range seqOut.Cues {
			if seqOut.Cues[i].Lines != parOut.Cues[i].Lines ||
				seqOut.Cues[i].Index != parOut.Cues[i].Index {
				t.Fatalf("factor=%d cue %d mismatch\nseq=%+v\npar=%+v", factor, i, seqOut.Cues[i], parOut.Cues[i])
			}
		}
		if len(seqCh) != len(parCh) {
			t.Fatalf("factor=%d change count seq=%d par=%d", factor, len(seqCh), len(parCh))
		}
		for i := range seqCh {
			if seqCh[i].CueIndex != parCh[i].CueIndex ||
				seqCh[i].Original != parCh[i].Original ||
				seqCh[i].Transformed != parCh[i].Transformed ||
				len(seqCh[i].Rules) != len(parCh[i].Rules) {
				t.Fatalf("factor=%d change %d mismatch\nseq=%+v\npar=%+v", factor, i, seqCh[i], parCh[i])
			}
			for j := range seqCh[i].Rules {
				if seqCh[i].Rules[j] != parCh[i].Rules[j] {
					t.Fatalf("factor=%d change %d rule %d: seq=%q par=%q", factor, i, j, seqCh[i].Rules[j], parCh[i].Rules[j])
				}
			}
		}
	}
}

func BenchmarkApplyAll(b *testing.B) {
	base := loadBenchDocument(b)
	conf := rules.DefaultConfig()
	impls := []struct {
		name string
		fn   ApplyFn
	}{
		{"Sequential", ApplyAllSequential},
		{"Parallel", ApplyAllParallel},
	}
	for _, factor := range []int{1, 10, 50} {
		doc := scaleDocument(base, factor)
		for _, impl := range impls {
			b.Run(impl.name+"/cues_"+strconv.Itoa(len(doc.Cues)), func(b *testing.B) {
				b.ReportAllocs()
				var sink int
				for b.Loop() {
					out, ch := impl.fn(doc, conf)
					sink += len(out.Cues) + len(ch)
				}
				_ = sink
			})
		}
	}
}
