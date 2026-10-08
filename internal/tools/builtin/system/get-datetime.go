package system

import (
	"time"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type GetDateTimeInput struct{}

type GetDateTimeOutput struct {
	Success  bool   `json:"success"`
	ISO8601  string `json:"iso8601"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Timezone string `json:"timezone"`
}

func DefineGetDateTimeTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[GetDateTimeInput, GetDateTimeOutput] {
	return genkit.DefineTool(
		g,
		"get_datetime",
		"Returns the current system date, time, and timezone in ISO 8601 format.",
		func(ctx *ai.ToolContext, input GetDateTimeInput) (GetDateTimeOutput, error) {
			now := time.Now()
			zone, _ := now.Zone()

			return GetDateTimeOutput{
				Success:  true,
				ISO8601:  now.Format(time.RFC3339),
				Date:     now.Format("2006-01-02"),
				Time:     now.Format("15:04:05"),
				Timezone: zone,
			}, nil
		},
	)
}
