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

// Custom MultiSelect template that:
// 1. Removes "type to filter" hint
// 2. Hides typed characters (removes .FilterMessage)
// 3. Shows clear control instructions
var multiSelectTemplateNoFilter = `
{{- define "option"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }}{{color "reset"}}{{else}} {{end}}
    {{- if index .Checked .CurrentOpt.Index }}{{color .Config.Icons.MarkedOption.Format }} {{ .Config.Icons.MarkedOption.Text }} {{else}}{{color .Config.Icons.UnmarkedOption.Format }} {{ .Config.Icons.UnmarkedOption.Text }} {{end}}
    {{- color "reset"}}
    {{- " "}}{{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "cyan"}}{{ $.GetDescription .CurrentOpt }}{{color "reset"}}{{end}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- "  "}}{{- color "cyan"}}[Arrow keys: move, Space: toggle, Enter: confirm]{{color "reset"}}
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

// useMultiSelectTemplateNoFilter temporarily overrides the global MultiSelect template
// to hide "type to filter" and prevent typed characters from showing.
// Returns a restore function that must be called to restore the original template.
func useMultiSelectTemplateNoFilter() func() {
	original := survey.MultiSelectQuestionTemplate
	survey.MultiSelectQuestionTemplate = multiSelectTemplateNoFilter
	return func() {
		survey.MultiSelectQuestionTemplate = original
	}
}
