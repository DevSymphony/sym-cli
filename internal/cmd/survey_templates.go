package cmd

import "github.com/AlecAivazis/survey/v2"

// Custom Select template that:
// 1. Removes "type to filter" hint
// 2. Hides typed characters (removes .FilterMessage)
// 3. Shows clear control instructions
var selectTemplateNoFilter = `
{{- define "option"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "cyan"}}{{ $.GetDescription .CurrentOpt }}{{end}}
    {{- color "reset"}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else}}
  {{- "  "}}{{- color "cyan"}}[Arrow keys: move, Enter: select]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" $.IterateOption $ix $option}}
  {{- end}}
{{- end}}`

// useSelectTemplateNoFilter temporarily overrides the global Select template
// to hide "type to filter" and prevent typed characters from showing.
// Returns a restore function that must be called to restore the original template.
func useSelectTemplateNoFilter() func() {
	original := survey.SelectQuestionTemplate
	survey.SelectQuestionTemplate = selectTemplateNoFilter
	return func() {
		survey.SelectQuestionTemplate = original
	}
}

// Custom Select template with no message output - only shows options
var selectTemplateNoMessage = `
{{- define "option"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}
    {{- color "reset"}}
{{end}}
{{- if .ShowAnswer}}{{/* hide answer line */}}
{{- else}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" $.IterateOption $ix $option}}
  {{- end}}
{{- end}}`

// useSelectTemplateNoMessage temporarily overrides the global Select template
// to hide message and answer output. Only shows options.
func useSelectTemplateNoMessage() func() {
	original := survey.SelectQuestionTemplate
	survey.SelectQuestionTemplate = selectTemplateNoMessage
	return func() {
		survey.SelectQuestionTemplate = original
	}
}
