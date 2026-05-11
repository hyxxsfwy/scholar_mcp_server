package sources

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Seelly/scholar_mcp_server/common"
)

const (
	logJSONMarshalFailed = "[ERROR] JSON序列化失败: %v"
	logReceivedParamsFmt = "[DEBUG] 接收到的参数: %+v"
)

func marshalStructuredData(data any) string {
	jsonResult, err := json.Marshal(data)
	if err != nil {
		log.Printf(logJSONMarshalFailed, err)
		return ""
	}

	return string(jsonResult)
}

func formatPaperSearchEntry(index int, paper common.UnifiedPaper, includeSource bool, abstractLimit int) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("📄 %d. %s\n", index, paper.Title))
	if includeSource {
		builder.WriteString(fmt.Sprintf("   来源: %s\n", paper.Source))
	}

	writeOptionalString(&builder, "   ID: %s\n", paper.ID)
	writeOptionalString(&builder, "   DOI: %s\n", paper.DOI)
	writeOptionalJoined(&builder, "   作者: %s\n", paper.Authors, 100)
	writeOptionalString(&builder, "   期刊: %s\n", paper.Journal)
	writeOptionalString(&builder, "   发表日期: %s\n", paper.PublishedDate)
	writeOptionalInt(&builder, "   引用数: %d\n", paper.CitationCount)
	if paper.IsOpenAccess {
		builder.WriteString("   📖 开放获取\n")
	}
	writeOptionalTruncatedString(&builder, "   摘要: %s\n", paper.Abstract, abstractLimit)
	writeOptionalString(&builder, "   链接: %s\n", paper.URL)
	writeOptionalString(&builder, "   PDF: %s\n", paper.PDFURL)
	writePaperEnrichmentSummary(&builder, paper.Enrichment, "   ")
	builder.WriteString("\n")

	return builder.String()
}

func formatSourceStatusSection(statuses []common.ServiceStatus) string {
	if len(statuses) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("📊 数据源状态:\n")
	for _, status := range statuses {
		statusIcon := "❌"
		if status.Available {
			statusIcon = "✅"
		}
		builder.WriteString(fmt.Sprintf("   %s %s: %s (%dms)\n",
			statusIcon, status.Source, status.Message, status.ResponseTime))
	}

	return builder.String()
}

func formatSuggestionsSection(suggestions []common.SearchHint) string {
	if len(suggestions) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n💡 搜索建议:\n")
	for _, hint := range suggestions {
		builder.WriteString(fmt.Sprintf("   - %s: %s\n", hint.Type, hint.Description))
	}

	return builder.String()
}

func formatPaperDetailBody(paper *common.UnifiedPaper, includeSource bool) string {
	var builder strings.Builder

	if includeSource {
		writeOptionalString(&builder, "🗃️ 数据源: %s\n", paper.Source)
	}

	writePaperIdentitySection(&builder, paper)
	writePaperPublicationSection(&builder, paper)
	writePaperMetricsSection(&builder, paper)
	writePaperClassificationSection(&builder, paper)
	writePaperAbstractSection(&builder, paper)
	writePaperEnrichmentDetail(&builder, paper.Enrichment)
	writePaperExternalLinksSection(&builder, paper)

	return builder.String()
}

func writePaperIdentitySection(builder *strings.Builder, paper *common.UnifiedPaper) {
	writeOptionalString(builder, "🆔 ID: %s\n", paper.ID)
	writeOptionalString(builder, "🔗 DOI: %s\n", paper.DOI)
	if len(paper.Authors) > 0 {
		builder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(paper.Authors, ", ")))
	}
}

func writePaperPublicationSection(builder *strings.Builder, paper *common.UnifiedPaper) {
	writeOptionalString(builder, "📍 期刊: %s\n", paper.Journal)
	writeOptionalString(builder, "🏢 出版商: %s\n", paper.Publisher)
	writeOptionalString(builder, "📅 发表日期: %s\n", paper.PublishedDate)
	writeOptionalString(builder, "📖 卷号: %s\n", paper.Volume)
	writeOptionalString(builder, "📄 期号: %s\n", paper.Issue)
	writeOptionalString(builder, "📄 页码: %s\n", paper.Pages)
}

func writePaperMetricsSection(builder *strings.Builder, paper *common.UnifiedPaper) {
	writeOptionalInt(builder, "📊 引用数: %d\n", paper.CitationCount)
	writeOptionalInt(builder, "👁️ 阅读数: %d\n", paper.ReadCount)
	if paper.IsOpenAccess {
		builder.WriteString("📖 开放获取: 是\n")
	}
}

func writePaperClassificationSection(builder *strings.Builder, paper *common.UnifiedPaper) {
	writeOptionalJoined(builder, "🏷️ 分类: %s\n", paper.Categories, 0)
	writeOptionalString(builder, "🌐 语言: %s\n", paper.Language)
	writeOptionalString(builder, "📋 类型: %s\n", paper.Type)
}

func writePaperAbstractSection(builder *strings.Builder, paper *common.UnifiedPaper) {
	if paper.Abstract != "" {
		builder.WriteString(fmt.Sprintf("\n📋 摘要:\n%s\n", paper.Abstract))
	}
}

func writePaperEnrichmentSummary(builder *strings.Builder, enrichment *common.PaperEnrichment, prefix string) {
	if enrichment == nil {
		return
	}
	if enrichment.Error != "" && enrichment.Content == "" {
		builder.WriteString(fmt.Sprintf("%sLLM梗概: 生成失败 (%s)\n", prefix, enrichment.Error))
		return
	}
	if enrichment.Content == "" {
		return
	}

	builder.WriteString(fmt.Sprintf("%sLLM文献大纲/梗概 (%s):\n", prefix, enrichment.Model))
	builder.WriteString(indentMultiline(common.TruncateText(enrichment.Content, 1200), prefix+"  "))
	builder.WriteString("\n")
	if enrichment.Error != "" {
		builder.WriteString(fmt.Sprintf("%sLLM补充说明: %s\n", prefix, enrichment.Error))
	}
}

func writePaperEnrichmentDetail(builder *strings.Builder, enrichment *common.PaperEnrichment) {
	if enrichment == nil {
		return
	}
	builder.WriteString("\n🧠 LLM文献大纲/梗概:\n")
	writeOptionalString(builder, "模型: %s\n", enrichment.Model)
	writeOptionalJoined(builder, "输入来源: %s\n", enrichment.InputSources, 0)
	if enrichment.PDFPrefetched {
		pdfSource := enrichment.PDFSource
		if pdfSource == "" {
			pdfSource = "unknown"
		}
		builder.WriteString(fmt.Sprintf("PDF预取: 是，来源 %s，提取字符数 %d\n", pdfSource, enrichment.PDFTextChars))
	}
	if enrichment.Content != "" {
		builder.WriteString(enrichment.Content)
		builder.WriteString("\n")
	}
	writeOptionalString(builder, "补充说明: %s\n", enrichment.Error)
}

func writePaperExternalLinksSection(builder *strings.Builder, paper *common.UnifiedPaper) {
	if len(paper.ExternalIDs) > 0 {
		builder.WriteString("\n🔗 外部链接:\n")
		for source, id := range paper.ExternalIDs {
			if id != "" {
				builder.WriteString(fmt.Sprintf("   %s: %s\n", source, id))
			}
		}
	}

	if paper.URL != "" {
		builder.WriteString(fmt.Sprintf("\n🌐 原始链接: %s\n", paper.URL))
	}
	if paper.PDFURL != "" {
		builder.WriteString(fmt.Sprintf("📄 PDF下载: %s\n", paper.PDFURL))
	}
}

func writeOptionalString(builder *strings.Builder, format, value string) {
	if value != "" {
		builder.WriteString(fmt.Sprintf(format, value))
	}
}

func writeOptionalInt(builder *strings.Builder, format string, value int) {
	if value > 0 {
		builder.WriteString(fmt.Sprintf(format, value))
	}
}

func writeOptionalJoined(builder *strings.Builder, format string, values []string, maxLen int) {
	if len(values) == 0 {
		return
	}

	joined := strings.Join(values, ", ")
	if maxLen > 0 {
		joined = common.TruncateText(joined, maxLen)
	}

	builder.WriteString(fmt.Sprintf(format, joined))
}

func writeOptionalTruncatedString(builder *strings.Builder, format, value string, maxLen int) {
	if value != "" {
		builder.WriteString(fmt.Sprintf(format, common.TruncateText(value, maxLen)))
	}
}

func indentMultiline(text string, prefix string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for index, line := range lines {
		lines[index] = prefix + line
	}
	return strings.Join(lines, "\n")
}
