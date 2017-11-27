package expr

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/metrictank/api/models"
	"github.com/grafana/metrictank/test"
	schema "gopkg.in/raintank/schema.v1"
)

func getModel(name string, data []schema.Point) models.Series {
	tags := make(map[string]string, 3)
	tagSplits := strings.Split(name, ";")

	tags["name"] = tagSplits[0]

	if len(tagSplits) > 1 {
		tagSplits = tagSplits[1:]
	}

	for _, split := range tagSplits {
		pair := strings.SplitN(split, "=", 2)
		tags[pair[0]] = pair[1]
	}
	return models.Series{
		Target:     name,
		QueryPatt:  name,
		Tags:       tags,
		Datapoints: getCopy(data),
	}
}
func TestGroupByTagsSingleSeries(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1", a),
	}
	out := in

	testGroupByTags("SingleSeries", in, out, "sum", []string{"tag1"}, t)
}

func TestGroupByTagsMultipleSeriesSingleResult(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1;tag2=val2_0", a),
		getModel("name1;tag1=val1;tag2=val2_1", b),
	}
	out := []models.Series{
		getModel("name1;tag1=val1", sumab),
	}

	testGroupByTags("MultipleSeriesSingleResult", in, out, "sum", []string{"tag1"}, t)
}

func TestGroupByTagsMultipleSeriesMultipleResults(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1;tag2=val2_0", a),
		getModel("name1;tag1=val1;tag2=val2_1", b),
		getModel("name1;tag1=val1_1;tag2=val2_0", c),
		getModel("name1;tag1=val1_1;tag2=val2_1", d),
	}
	out := []models.Series{
		getModel("name1;tag1=val1", sumab),
		getModel("name1;tag1=val1_1", sumcd),
	}

	testGroupByTags("MultipleSeriesMultipleResult", in, out, "sum", []string{"tag1"}, t)
}
func TestGroupByTagsMultipleSeriesMultipleResultsMultipleNames(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1;tag2=val2_0", a),
		getModel("name1;tag1=val1;tag2=val2_1", b),
		getModel("name2;tag1=val1_1;tag2=val2_0", c),
		getModel("name2;tag1=val1_1;tag2=val2_1", d),
	}
	out := []models.Series{
		getModel("sum;tag1=val1", sumab),
		getModel("sum;tag1=val1_1", sumcd),
	}

	testGroupByTags("MultipleSeriesMultipleResultsMultipleNames", in, out, "sum", []string{"tag1"}, t)
}

func TestGroupByTagsMultipleSeriesMultipleResultsGroupByName(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1;tag2=val2_0", a),
		getModel("name1;tag1=val1;tag2=val2_1", b),
		getModel("name2;tag1=val1_1;tag2=val2_0", c),
		getModel("name2;tag1=val1_1;tag2=val2_1", d),
	}
	out := []models.Series{
		getModel("name1;tag1=val1", sumab),
		getModel("name2;tag1=val1_1", sumcd),
	}

	testGroupByTags("MultipleSeriesMultipleResultsGroupByName", in, out, "sum", []string{"tag1", "name"}, t)
}

func TestGroupByTagsMultipleSeriesMissingTag(t *testing.T) {
	in := []models.Series{
		getModel("name1;tag1=val1;tag2=val2_0", a),
		getModel("name1;tag1=val1;tag2=val2_1", b),
		getModel("name2;tag1=val1_1;tag2=val2_0", c),
		getModel("name2;tag1=val1_1;tag2=val2_1", d),
	}
	out := []models.Series{
		getModel("name1;missingTag=;tag1=val1", sumab),
		getModel("name2;missingTag=;tag1=val1_1", sumcd),
	}

	testGroupByTags("MultipleSeriesMultipleResultsGroupByName", in, out, "sum", []string{"tag1", "name", "missingTag"}, t)
}

func TestGroupByTagsAllAggregators(t *testing.T) {
	aggregators := []struct {
		name             string
		result1, result2 []schema.Point
	}{
		{name: "sum", result1: sumab, result2: sumabc},
		{name: "avg", result1: avgab, result2: avgabc},
		{name: "average", result1: avgab, result2: avgabc},
		{name: "max", result1: maxab, result2: maxabc},
	}

	for _, agg := range aggregators {
		in := []models.Series{
			getModel("name1;tag1=val1;tag2=val2_0", a),
			getModel("name1;tag1=val1;tag2=val2_1", b),
			getModel("name2;tag1=val1_1;tag2=val2_0", a),
			getModel("name2;tag1=val1_1;tag2=val2_1", b),
			getModel("name2;tag1=val1_1;tag2=val2_2", c),
		}
		out := []models.Series{
			getModel("name1;tag1=val1", agg.result1),
			getModel("name2;tag1=val1_1", agg.result2),
		}

		testGroupByTags("AllAggregators:"+agg.name, in, out, agg.name, []string{"tag1", "name"}, t)
	}
}

func testGroupByTags(name string, in []models.Series, out []models.Series, agg string, tags []string, t *testing.T) {
	f := NewGroupByTags()
	gby := f.(*FuncGroupByTags)
	gby.in = NewMock(in)
	gby.aggregator = agg
	gby.tags = tags

	got, err := f.Exec(make(map[Req][]models.Series))
	if err != nil {
		t.Fatalf("case %q: err should be nil. got %q", name, err)
	}
	if len(got) != len(out) {
		t.Fatalf("case %q: GroupByTags output expected to be %d but actually %d", name, len(out), len(got))
	}

	// Make sure got and out are in the same order
	sort.Slice(got, func(i, j int) bool {
		return got[i].Target < got[j].Target
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].Target < out[j].Target
	})
	for i, g := range got {
		o := out[i]
		if g.Target != o.Target {
			t.Fatalf("case %q: expected target %q, got %q", name, o.Target, g.Target)
		}
		if len(g.Datapoints) != len(o.Datapoints) {
			t.Fatalf("case %q: len output expected %d, got %d", name, len(o.Datapoints), len(g.Datapoints))
		}
		for j, p := range g.Datapoints {
			bothNaN := math.IsNaN(p.Val) && math.IsNaN(o.Datapoints[j].Val)
			if (bothNaN || p.Val == o.Datapoints[j].Val) && p.Ts == o.Datapoints[j].Ts {
				continue
			}
			t.Fatalf("case %q: output point %d - expected %v got %v", name, j, o.Datapoints[j], p)
		}
	}
}

func BenchmarkGroupByTags10k_1NoNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1, test.RandFloats10k, test.RandFloats10k)
}
func BenchmarkGroupByTags10k_10NoNulls(b *testing.B) {
	benchmarkGroupByTags(b, 10, test.RandFloats10k, test.RandFloats10k)
}
func BenchmarkGroupByTags10k_100NoNulls(b *testing.B) {
	benchmarkGroupByTags(b, 100, test.RandFloats10k, test.RandFloats10k)
}
func BenchmarkGroupByTags10k_1000NoNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1000, test.RandFloats10k, test.RandFloats10k)
}

func BenchmarkGroupByTags10k_1SomeSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1, test.RandFloats10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_10SomeSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 10, test.RandFloats10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_100SomeSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 100, test.RandFloats10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_1000SomeSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1000, test.RandFloats10k, test.RandFloatsWithNulls10k)
}

func BenchmarkGroupByTags10k_1AllSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1, test.RandFloatsWithNulls10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_10AllSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 10, test.RandFloatsWithNulls10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_100AllSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 100, test.RandFloatsWithNulls10k, test.RandFloatsWithNulls10k)
}
func BenchmarkGroupByTags10k_1000AllSeriesHalfNulls(b *testing.B) {
	benchmarkGroupByTags(b, 1000, test.RandFloatsWithNulls10k, test.RandFloatsWithNulls10k)
}

func benchmarkGroupByTags(b *testing.B, numSeries int, fn0, fn1 func() []schema.Point) {
	var input []models.Series
	tagValues := []string{"tag1", "tag2", "tag3", "tag4"}
	for i := 0; i < numSeries; i++ {
		tags := make(map[string]string, len(tagValues))

		for t, tag := range tagValues {
			tags[tag] = strconv.Itoa(t)
		}
		series := models.Series{
			Target: strconv.Itoa(i),
		}
		if i%1 == 0 {
			series.Datapoints = fn0()
		} else {
			series.Datapoints = fn1()
		}
		input = append(input, series)
	}
	b.ResetTimer()
	var err error
	for i := 0; i < b.N; i++ {
		f := NewGroupByTags()
		gby := f.(*FuncGroupByTags)
		gby.in = NewMock(input)
		gby.aggregator = "sum"
		gby.tags = []string{"tag1", "tag2"}
		results, err = f.Exec(make(map[Req][]models.Series))
		if err != nil {
			b.Fatalf("%s", err)
		}
	}
	b.SetBytes(int64(numSeries * len(results[0].Datapoints) * 12))
}
