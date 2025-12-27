package agent

import "embed"

// Version of the agent templates - increment when updating templates
const TemplateVersion = "1.0.0"

//go:embed templates/skill/*.md
var skillTemplates embed.FS

//go:embed templates/commands/*.md
var commandTemplates embed.FS

//go:embed templates/cursor/*.md
var cursorTemplates embed.FS

//go:embed templates/claude/*.md
var claudeTemplates embed.FS
