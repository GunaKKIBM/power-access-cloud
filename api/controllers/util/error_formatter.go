package util

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FormatErrorForEmail formats error messages for clean display in emails
// It handles:
// - HTML entity decoding (e.g., &#34; to ")
// - JSON formatting
// - Removing excessive whitespace
// - Making the message more readable
func FormatErrorForEmail(errorMsg string) string {
	if errorMsg == "" {
		return ""
	}

	// Decode HTML entities (e.g., &#34; becomes ")
	formatted := html.UnescapeString(errorMsg)

	// Try to extract and format JSON-like error messages
	formatted = formatJSONError(formatted)

	// Clean up excessive whitespace
	formatted = strings.TrimSpace(formatted)

	return formatted
}

// formatJSONError attempts to format JSON-like error messages for better readability
func formatJSONError(msg string) string {
	// Pattern to match common API error format: [METHOD /path][code] {json}
	apiErrorPattern := regexp.MustCompile(`\[([A-Z]+)\s+([^\]]+)\]\[(\d+)\]\s*(.+)`)
	matches := apiErrorPattern.FindStringSubmatch(msg)

	if len(matches) == 5 {
		method := matches[1]
		path := matches[2]
		statusCode := matches[3]
		jsonPart := matches[4]

		// Try to parse and format the JSON part
		jsonFormatted := formatJSONString(jsonPart)

		// Build a more readable format
		var builder strings.Builder
		builder.WriteString("API Error Details:\n")
		builder.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		builder.WriteString(fmt.Sprintf("Method:      %s\n", method))
		builder.WriteString(fmt.Sprintf("Endpoint:    %s\n", path))
		builder.WriteString(fmt.Sprintf("Status Code: %s\n", statusCode))
		builder.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		builder.WriteString(jsonFormatted)

		return builder.String()
	}

	return msg
}

// formatJSONString formats a JSON string for better readability
func formatJSONString(jsonStr string) string {
	// Remove outer braces if present
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = strings.TrimPrefix(jsonStr, "{")
	jsonStr = strings.TrimSuffix(jsonStr, "}")

	// Split by comma and format each field
	var formatted strings.Builder

	// Pattern to match JSON key-value pairs
	fieldPattern := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]*)"`)
	matches := fieldPattern.FindAllStringSubmatch(jsonStr, -1)

	if len(matches) > 0 {
		for _, match := range matches {
			if len(match) == 3 {
				key := match[1]
				value := match[2]

				// Capitalize first letter of key for display
				displayKey := cases.Title(language.English).String(strings.ReplaceAll(key, "_", " "))
				formatted.WriteString(fmt.Sprintf("%s: %s\n", displayKey, value))
			}
		}
		return formatted.String()
	}

	// If regex parsing fails, do simple formatting
	jsonStr = strings.ReplaceAll(jsonStr, `","`, "\n")
	jsonStr = strings.ReplaceAll(jsonStr, `":"`, ": ")
	jsonStr = strings.ReplaceAll(jsonStr, `"`, "")

	return jsonStr
}

// FormatErrorForLog formats error messages for logging (keeps more technical details)
func FormatErrorForLog(errorMsg string) string {
	// For logs, just decode HTML entities but keep the structure
	return html.UnescapeString(errorMsg)
}
