package initializers

import (
	"fmt"
	"github.com/timescale/tsbs/pkg/targets"
	"github.com/timescale/tsbs/pkg/targets/clickhouse"
	"github.com/timescale/tsbs/pkg/targets/constants"
	"github.com/timescale/tsbs/pkg/targets/influx"
	"github.com/timescale/tsbs/pkg/targets/victoriametrics"
	"strings"
)

func GetTarget(format string) targets.ImplementedTarget {
	switch format {
	case constants.FormatClickhouse:
		return clickhouse.NewTarget()
	case constants.FormatInflux:
		return influx.NewTarget()
	case constants.FormatVictoriaMetrics:
		return victoriametrics.NewTarget()
	}

	supportedFormatsStr := strings.Join(constants.SupportedFormats(), ",")
	panic(fmt.Sprintf("Unrecognized format %s, supported: %s", format, supportedFormatsStr))
}
