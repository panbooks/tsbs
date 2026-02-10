package inputs

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/databases/clickhouse"
	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/databases/influx"
	"github.com/timescale/tsbs/cmd/tsbs_generate_queries/uses/devops"
	queryUtils "github.com/timescale/tsbs/cmd/tsbs_generate_queries/utils"
	internalUtils "github.com/timescale/tsbs/internal/utils"
	"github.com/timescale/tsbs/pkg/data/usecases/common"
	"github.com/timescale/tsbs/pkg/query"
	"github.com/timescale/tsbs/pkg/query/config"
	"github.com/timescale/tsbs/pkg/targets/constants"
)

const (
	errTotalGroupsZero  = "incorrect interleaved groups configuration: total groups = 0"
	errInvalidGroupsFmt = "incorrect interleaved groups configuration: id %d >= total groups %d"
)

func TestQueryGeneratorConfigValidate(t *testing.T) {
	c := &config.QueryGeneratorConfig{
		BaseConfig: common.BaseConfig{
			Seed:   123,
			Format: constants.FormatClickhouse,
			Use:    common.UseCaseDevops,
			Scale:  10,
		},
		QueryType:            "foo",
		InterleavedGroupID:   0,
		InterleavedNumGroups: 1,
	}

	// Test base validation
	err := c.Validate()
	if err != nil {
		t.Errorf("unexpected error for correct config: %v", err)
	}

	c.Format = "bad format"
	err = c.Validate()
	if err == nil {
		t.Errorf("unexpected lack of error for bad format")
	}
	c.Format = constants.FormatClickhouse

	// Test QueryType validation
	c.QueryType = ""
	err = c.Validate()
	if err == nil {
		t.Errorf("unexpected lack of error for empty query type")
	}
	c.QueryType = "foo"

	// Test groups validation
	c.InterleavedNumGroups = 0
	err = c.Validate()
	if err == nil {
		t.Errorf("unexpected lack of error for 0 groups")
	} else if got := err.Error(); got != errTotalGroupsZero {
		t.Errorf("incorrect error for 0 groups: got\n%s\nwant\n%s", got, errTotalGroupsZero)
	}
	c.InterleavedNumGroups = 1

	c.InterleavedGroupID = 2
	err = c.Validate()
	if err == nil {
		t.Errorf("unexpected lack of error for group id > num groups")
	} else {
		want := fmt.Sprintf(errInvalidGroupsFmt, 2, 1)
		if got := err.Error(); got != want {
			t.Errorf("incorrect error for group id > num groups: got\n%s\nwant\n%s", got, want)
		}
	}
}

func TestNewQueryGenerator(t *testing.T) {
	m := map[string]map[string]queryUtils.QueryFillerMaker{
		"foo": {
			"bar": nil,
			"baz": nil,
		},
		"bar": {
			"baz": nil,
		},
	}
	g := NewQueryGenerator(m)
	if !reflect.DeepEqual(g.useCaseMatrix, m) {
		t.Errorf("useCaseMatrix not properly set")
	}
}

func TestQueryGeneratorInit(t *testing.T) {
	const okQueryType = "single-groupby-1-1-1"
	g := &QueryGenerator{
		useCaseMatrix: map[string]map[string]queryUtils.QueryFillerMaker{
			common.UseCaseDevops: {
				okQueryType: nil,
			},
		},
		factories: make(map[string]interface{}),
	}
	// Test that empty config fails
	err := g.init(nil)
	if err == nil {
		t.Errorf("unexpected lack of error with empty config")
	} else if got := err.Error(); got != ErrNoConfig {
		t.Errorf("incorrect error: got\n%s\nwant\n%s", got, ErrNoConfig)
	}

	// Test that wrong type of config fails
	err = g.init(&common.BaseConfig{})
	if err == nil {
		t.Errorf("unexpected lack of error with invalid config")
	} else if got := err.Error(); got != ErrInvalidDataConfig {
		t.Errorf("incorrect error: got\n%s\nwant\n%s", got, ErrInvalidDataConfig)
	}

	// Test that empty, invalid config fails
	err = g.init(&config.QueryGeneratorConfig{})
	if err == nil {
		t.Errorf("unexpected lack of error with empty QueryGeneratorConfig")
	}

	c := &config.QueryGeneratorConfig{
		BaseConfig: common.BaseConfig{
			Format: constants.FormatClickhouse,
			Use:    common.UseCaseCPUOnly, // not in the useCaseMatrix
			Scale:  1,
		},
		QueryType:            "unknown query type",
		InterleavedNumGroups: 1,
	}

	// Test use case not in matrix
	err = g.init(c)
	want := fmt.Sprintf(errBadUseFmt, common.UseCaseCPUOnly)
	if err == nil {
		t.Errorf("unexpected lack of error with bad use case")
	} else if got := err.Error(); got != want {
		t.Errorf("incorrect error for bad use case:\ngot\n%s\nwant\n%s", got, want)
	}
	c.Use = common.UseCaseDevops

	// Test unknown query type
	err = g.init(c)
	want = fmt.Sprintf(errBadQueryTypeFmt, common.UseCaseDevops, "unknown query type")
	if err == nil {
		t.Errorf("unexpected lack of error with bad query type")
	} else if got := err.Error(); got != want {
		t.Errorf("incorrect error for bad query type:\ngot\n%s\nwant\n%s", got, want)
	}
	c.QueryType = okQueryType

	const errTimePrefix = "cannot parse time from string"

	// Test incorrect time format for start
	c.TimeStart = "2006 Jan 2"
	err = g.init(c)
	if err == nil {
		t.Errorf("unexpected lack of error with bad start date")
	} else if got := err.Error(); !strings.HasPrefix(got, errTimePrefix) {
		t.Errorf("unexpected error for bad start date: got\n%s", got)
	}
	c.TimeStart = defaultTimeStart

	// Test incorrect time format for end
	c.TimeEnd = "Jan 3rd 2016"
	err = g.init(c)
	if err == nil {
		t.Errorf("unexpected lack of error with bad end date")
	} else if got := err.Error(); !strings.HasPrefix(got, errTimePrefix) {
		t.Errorf("unexpected error for bad end date: got\n%s", got)
	}
	c.TimeEnd = defaultTimeEnd

	// Test that Out is set to os.Stdout if unset
	err = g.init(c)
	if err != nil {
		t.Errorf("unexpected error when checking Out: got %v", err)
	} else if g.Out != os.Stdout {
		t.Errorf("Out not set to Stdout")
	}

	// Test that Out is same if set
	var buf bytes.Buffer
	g.Out = &buf
	err = g.init(c)
	if err != nil {
		t.Errorf("unexpected error when checking Out: got %v", err)
	} else if g.Out != &buf {
		t.Errorf("Out not set to explicit io.Writer")
	}

	// Test that DebugOut is set to os.Stderr if unset
	err = g.init(c)
	if err != nil {
		t.Errorf("unexpected error when checking DebugOut: got %v", err)
	} else if g.DebugOut != os.Stderr {
		t.Errorf("DebugOut not set to Stderr")
	}
}

func TestGetUseCaseGenerator(t *testing.T) {
	var useCaseMatrix = map[string]map[string]queryUtils.QueryFillerMaker{
		"devops": {
			devops.LabelLastpoint: devops.NewLastPointPerHost,
		},
	}
	const scale = 10
	tsStart, _ := internalUtils.ParseUTCTime(defaultTimeStart)
	tsEnd, _ := internalUtils.ParseUTCTime(defaultTimeEnd)
	c := &config.QueryGeneratorConfig{
		BaseConfig: common.BaseConfig{
			Scale:     scale,
			TimeStart: defaultTimeStart,
			TimeEnd:   defaultTimeEnd,
		},
	}
	g := &QueryGenerator{
		conf:          c,
		tsStart:       tsStart,
		tsEnd:         tsEnd,
		factories:     make(map[string]interface{}),
		useCaseMatrix: useCaseMatrix,
	}

	checkType := func(format string, want queryUtils.QueryGenerator) queryUtils.QueryGenerator {
		wantType := reflect.TypeOf(want)
		c.Format = format
		c.Use = common.UseCaseDevops
		c.QueryType = "lastpoint"
		c.InterleavedNumGroups = 1
		err := g.init(c)

		if err != nil {
			t.Fatalf("Error initializing query generator: %s", err)
		}

		useGen, err := g.getUseCaseGenerator(c)
		if err != nil {
			t.Errorf("unexpected error with format '%s': %v", format, err)
		}
		if got := reflect.TypeOf(useGen); got != wantType {
			t.Errorf("format '%s' does not give right use case gen: got %v want %v", format, got, wantType)
		}

		return useGen
	}

	bi := influx.BaseGenerator{}
	indb, err := bi.NewDevops(tsStart, tsEnd, scale)
	if err != nil {
		t.Fatalf("Error creating influx query generator")
	}
	checkType(constants.FormatInflux, indb)

	bcc := clickhouse.BaseGenerator{}
	clickh, err := bcc.NewDevops(tsStart, tsEnd, scale)
	if err != nil {
		t.Fatalf("Error creating clickhouse query generator")
	}
	checkType(constants.FormatClickhouse, clickh)

	bcc.UseTags = true
	clickt, err := bcc.NewDevops(tsStart, tsEnd, scale)
	checkType(constants.FormatClickhouse, clickt)
	if got := clickt.(*clickhouse.Devops).UseTags; got != bcc.UseTags {
		t.Errorf("clickhouse UseTags not set correctly: got %v want %v", got, bcc.UseTags)
	}

	// Test error condition
	c.Format = "bad format"
	useGen, err := g.getUseCaseGenerator(c)
	if err == nil {
		t.Errorf("unexpected lack of error for bad format")
	} else if got := err.Error(); got != fmt.Sprintf(errUnknownFormatFmt, c.Format) {
		t.Errorf("incorrect error:\ngot\n%s\nwant\n%s", got, fmt.Sprintf(errUnknownFormatFmt, c.Format))
	} else if useGen != nil {
		t.Errorf("useGen was not nil")
	}
}

func getTestConfigAndGenerator() (*config.QueryGeneratorConfig, *QueryGenerator) {
	const scale = 10
	tsStart, _ := internalUtils.ParseUTCTime(defaultTimeStart)
	tsEnd, _ := internalUtils.ParseUTCTime(defaultTimeEnd)
	tsEnd = tsEnd.Add(time.Second)
	c := &config.QueryGeneratorConfig{
		BaseConfig: common.BaseConfig{
			Format:    constants.FormatClickhouse,
			Use:       common.UseCaseCPUOnly,
			Scale:     scale,
			TimeStart: defaultTimeStart,
			TimeEnd:   strings.Replace(defaultTimeEnd, ":00Z", ":01Z", 1),
			Seed:      123,
		},
		Limit:                3,
		QueryType:            "single-groupby-1-1-1",
		ClickhouseUseTags:    true,
		InterleavedNumGroups: 1,
	}
	g := &QueryGenerator{
		useCaseMatrix: map[string]map[string]queryUtils.QueryFillerMaker{
			common.UseCaseCPUOnly: {
				"single-groupby-1-1-1": devops.NewSingleGroupby(1, 1, 1),
			},
		},
		conf:      c,
		tsStart:   tsStart,
		tsEnd:     tsEnd,
		DebugOut:  os.Stderr,
		factories: make(map[string]interface{}),
	}

	return c, g
}

func checkGeneratedOutput(t *testing.T, buf *bytes.Buffer) {
	r := bufio.NewReader(buf)
	decoder := gob.NewDecoder(r)
	i := 0
	for {
		var q query.ClickHouse
		err := decoder.Decode(&q)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("unexpected error while decoding: got %v", err)
		}
		if len(q.SqlQuery) == 0 {
			t.Errorf("query %d has empty SqlQuery", i)
		}
		i++
	}
	if i != 3 {
		t.Errorf("incorrect number of queries: got %d want %d", i, 3)
	}
}

func TestQueryGeneratorRunQueryGeneration(t *testing.T) {
	config, g := getTestConfigAndGenerator()
	err := g.init(config)
	if err != nil {
		t.Fatalf("Error initializing query generator: %s", err)
	}

	var buf bytes.Buffer
	g.bufOut = bufio.NewWriter(&buf)
	var debug bytes.Buffer
	g.DebugOut = &debug

	useGen, err := g.getUseCaseGenerator(config)
	if err != nil {
		t.Fatalf("could not get use case gen: %v", err)
	}
	filler := g.useCaseMatrix[config.Use][config.QueryType](useGen)

	err = g.runQueryGeneration(useGen, filler, config)
	if err != nil {
		t.Errorf("unexpected error: got %v", err)
	}

	checkGeneratedOutput(t, &buf)
}

type badWriter struct {
	when  int
	count int
}

func (w *badWriter) Write(p []byte) (int, error) {
	if w.count >= w.when {
		return 0, fmt.Errorf("error writing")
	}
	w.count++
	return len(p), nil
}

func TestQueryGeneratorRunQueryGenerationErrors(t *testing.T) {
	c, g := getTestConfigAndGenerator()
	var buf bytes.Buffer
	g.bufOut = bufio.NewWriter(&buf)
	g.init(c)

	useGen, err := g.getUseCaseGenerator(c)
	if err != nil {
		t.Fatalf("could not get use case gen: %v", err)
	}
	filler := g.useCaseMatrix[c.Use][c.QueryType](useGen)

	checkErr := func(want string) {
		err = g.runQueryGeneration(useGen, filler, c)
		if err == nil {
			t.Errorf("unexpected lack of error")
		} else if got := err.Error(); !strings.HasPrefix(got, want) {
			t.Errorf("incorrect error for output query stats:\ngot\n%s\nwant\n%s", got, want)
		}
	}

	// Test error condition when printing stats
	g.DebugOut = &badWriter{}
	want := fmt.Sprintf(errCouldNotQueryStatsFmt, "error writing")
	checkErr(want)

	// Test error condition when printing seed
	c.Debug = 1
	want = fmt.Sprintf(errCouldNotDebugFmt, "error writing")
	checkErr(want)

	// Test error condition inside loop; first debug is success
	g.DebugOut = &badWriter{when: 1}
	checkErr(want)

	g.DebugOut = &badWriter{when: 2}
	checkErr(want)

	// Test error on encoding
	g.DebugOut = &badWriter{when: 10000}
	g.bufOut = bufio.NewWriterSize(&badWriter{}, 8) // small buffer forces it to write to underlying
	want = fmt.Sprintf(errCouldNotEncodeQueryFmt, "error writing")
	checkErr(want)
}

func TestQueryGeneratorGenerate(t *testing.T) {
	g := &QueryGenerator{
		factories: make(map[string]interface{}),
	}

	// Test that an invalid config fails
	c := &config.QueryGeneratorConfig{}
	err := g.Generate(c)
	if err == nil {
		t.Errorf("unexpected lack of error with empty QueryGeneratorConfig")
	}

	c, g = getTestConfigAndGenerator()
	var buf bytes.Buffer
	g.Out = &buf
	g.DebugOut = ioutil.Discard
	err = g.Generate(c)
	if err != nil {
		t.Errorf("unexpected error when generating: got %v", err)
	}
	checkGeneratedOutput(t, &buf)
}
