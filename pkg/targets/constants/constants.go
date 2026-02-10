package constants

// Formats supported for generation
const (
	FormatClickhouse      = "clickhouse"
	FormatInflux          = "influx"
	FormatVictoriaMetrics = "victoriametrics"
)

func SupportedFormats() []string {
	return []string{
		FormatClickhouse,
		FormatInflux,
		FormatVictoriaMetrics,
	}
}
