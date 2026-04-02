package template

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

var _ ports.TemplateRenderer = (*renderer)(nil)

type renderer struct {
	templates map[domain.ChannelType]map[domain.EventType]*htmlTemplate.Template
}

func NewRenderer(basePath string) (ports.TemplateRenderer, error) {
	r := &renderer{
		templates: make(map[domain.ChannelType]map[domain.EventType]*htmlTemplate.Template),
	}

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(d.Name()) != ".tmpl" {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) != 2 {
			return nil
		}

		channelStr := parts[0]
		fileName := parts[1]

		eventStr := strings.TrimSuffix(fileName, ".tmpl")

		channelType := domain.ChannelType(channelStr)
		eventType := domain.EventType(eventStr)

		tpl, err := htmlTemplate.ParseFiles(path)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		if _, ok := r.templates[channelType]; !ok {
			r.templates[channelType] = make(map[domain.EventType]*htmlTemplate.Template)
		}

		r.templates[channelType][eventType] = tpl

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load templates from %s: %w", basePath, err)
	}

	return r, nil
}

func (r *renderer) Render(eventType domain.EventType, channelType domain.ChannelType, payload any) (string, error) {
	channelTpls, ok := r.templates[channelType]
	if !ok {
		return "", fmt.Errorf("no templates found for channel type: %s", channelType)
	}

	tpl, ok := channelTpls[eventType]
	if !ok {
		return "", fmt.Errorf("template not found for event %s in channel %s", eventType, channelType)
	}

	var buf bytes.Buffer

	if err := tpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
