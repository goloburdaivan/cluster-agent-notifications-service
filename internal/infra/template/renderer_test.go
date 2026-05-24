package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

func TestRenderer_Render_Success(t *testing.T) {
	dir := t.TempDir()

	channelDir := filepath.Join(dir, "telegram")
	require.NoError(t, os.MkdirAll(channelDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(channelDir, "trivy.security.tmpl"), []byte(`Hello {{.Name}}!`), 0644))

	r, err := NewRenderer(dir)
	require.NoError(t, err)

	result, err := r.Render(domain.EventTypeSecurity, domain.ChannelTypeTelegram, struct{ Name string }{"World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestRenderer_Render_MultipleChannelTypes(t *testing.T) {
	dir := t.TempDir()

	for _, ch := range []string{"telegram", "slack", "email"} {
		channelDir := filepath.Join(dir, ch)
		require.NoError(t, os.MkdirAll(channelDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(channelDir, "trivy.security.tmpl"),
			[]byte(ch+": {{.Msg}}"),
			0644,
		))
	}

	r, err := NewRenderer(dir)
	require.NoError(t, err)

	data := struct{ Msg string }{"alert"}

	result, err := r.Render(domain.EventTypeSecurity, domain.ChannelTypeTelegram, data)
	require.NoError(t, err)
	assert.Equal(t, "telegram: alert", result)

	result, err = r.Render(domain.EventTypeSecurity, domain.ChannelTypeSlack, data)
	require.NoError(t, err)
	assert.Equal(t, "slack: alert", result)

	result, err = r.Render(domain.EventTypeSecurity, domain.ChannelTypeEmail, data)
	require.NoError(t, err)
	assert.Equal(t, "email: alert", result)
}

func TestRenderer_Render_NoChannelType(t *testing.T) {
	dir := t.TempDir()
	r, err := NewRenderer(dir)
	require.NoError(t, err)

	_, err = r.Render(domain.EventTypeSecurity, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no templates found")
}

func TestRenderer_Render_NoEventType(t *testing.T) {
	dir := t.TempDir()

	channelDir := filepath.Join(dir, "telegram")
	require.NoError(t, os.MkdirAll(channelDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(channelDir, "other.event.tmpl"), []byte("test"), 0644))

	r, err := NewRenderer(dir)
	require.NoError(t, err)

	_, err = r.Render(domain.EventTypeSecurity, domain.ChannelTypeTelegram, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestRenderer_Render_TemplateExecutionError(t *testing.T) {
	dir := t.TempDir()

	channelDir := filepath.Join(dir, "email")
	require.NoError(t, os.MkdirAll(channelDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(channelDir, "trivy.security.tmpl"), []byte(`{{.Missing.Field}}`), 0644))

	r, err := NewRenderer(dir)
	require.NoError(t, err)

	_, err = r.Render(domain.EventTypeSecurity, domain.ChannelTypeEmail, struct{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execute template")
}

func TestNewRenderer_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	r, err := NewRenderer(dir)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestNewRenderer_SkipsNonTmplFiles(t *testing.T) {
	dir := t.TempDir()
	channelDir := filepath.Join(dir, "slack")
	require.NoError(t, os.MkdirAll(channelDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(channelDir, "readme.txt"), []byte("not a template"), 0644))

	r, err := NewRenderer(dir)
	require.NoError(t, err)

	_, err = r.Render(domain.EventTypeSecurity, domain.ChannelTypeSlack, nil)
	assert.Error(t, err)
}

func TestNewRenderer_SkipsDeepNesting(t *testing.T) {
	dir := t.TempDir()
	deepDir := filepath.Join(dir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deepDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deepDir, "test.tmpl"), []byte("test"), 0644))

	r, err := NewRenderer(dir)
	require.NoError(t, err)
	assert.NotNil(t, r)
}

func TestNewRenderer_InvalidPath(t *testing.T) {
	_, err := NewRenderer("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}
