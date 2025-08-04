package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/prompt"
)

// PromptResourceManager manages prompt templates as MCP resources
type PromptResourceManager struct {
	promptManager prompt.Manager
	logger        *zap.Logger
	cache         map[string]*PromptResource
	cacheExpiry   time.Duration
	lastCacheTime time.Time
}

// NewPromptResourceManager creates a new prompt resource manager
func NewPromptResourceManager(promptManager prompt.Manager, logger *zap.Logger) *PromptResourceManager {
	return &PromptResourceManager{
		promptManager: promptManager,
		logger:        logger,
		cache:         make(map[string]*PromptResource),
		cacheExpiry:   5 * time.Minute, // Cache for 5 minutes
	}
}

// ListResources returns all available prompt template resources
func (prm *PromptResourceManager) ListResources(ctx context.Context) ([]PromptResource, error) {
	prm.logger.Debug("Listing prompt template resources")

	// Check cache validity
	if time.Since(prm.lastCacheTime) > prm.cacheExpiry {
		prm.cache = make(map[string]*PromptResource)
	}

	var resources []PromptResource

	// Get all templates from the prompt manager
	categories := []string{
		prompt.CategoryGeneral,
		prompt.CategoryLanguages,
		prompt.CategoryWorkflows,
		prompt.CategoryWorkspace,
		prompt.CategoryCustom,
	}

	for _, category := range categories {
		templates, err := prm.promptManager.ListTemplates(category)
		if err != nil {
			prm.logger.Error("Failed to list templates", zap.String("category", category), zap.Error(err))
			continue
		}

		for _, template := range templates {
			resource := prm.templateToResource(template)
			resources = append(resources, *resource)
			
			// Cache the resource
			prm.cache[resource.URI] = resource
		}
	}

	prm.lastCacheTime = time.Now()
	prm.logger.Debug("Listed prompt resources", zap.Int("count", len(resources)))

	return resources, nil
}

// GetResource retrieves a specific prompt template resource by URI
func (prm *PromptResourceManager) GetResource(ctx context.Context, uri string) (*PromptResource, error) {
	prm.logger.Debug("Getting prompt resource", zap.String("uri", uri))

	// Check cache first
	if resource, exists := prm.cache[uri]; exists && time.Since(prm.lastCacheTime) < prm.cacheExpiry {
		prm.logger.Debug("Resource found in cache", zap.String("uri", uri))
		return resource, nil
	}

	// Extract template name from URI
	templateName := prm.extractTemplateNameFromURI(uri)
	if templateName == "" {
		return nil, fmt.Errorf("invalid template URI: %s", uri)
	}

	// Get template from prompt manager
	template, err := prm.promptManager.GetTemplate(templateName)
	if err != nil {
		prm.logger.Error("Failed to get template", zap.String("name", templateName), zap.Error(err))
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// Convert to resource
	resource := prm.templateToResource(template)
	prm.cache[uri] = resource

	prm.logger.Debug("Retrieved prompt resource", zap.String("uri", uri), zap.String("name", resource.Name))
	return resource, nil
}

// RegisterTemplate registers a new template as an MCP resource
func (prm *PromptResourceManager) RegisterTemplate(ctx context.Context, template prompt.Template) (*PromptResource, error) {
	prm.logger.Debug("Registering template as MCP resource", zap.String("name", template.Name))

	resource := prm.templateToResource(template)
	prm.cache[resource.URI] = resource

	prm.logger.Info("Template registered as MCP resource", 
		zap.String("name", template.Name),
		zap.String("uri", resource.URI),
		zap.String("category", template.Category))

	return resource, nil
}

// UnregisterTemplate removes a template from MCP resources
func (prm *PromptResourceManager) UnregisterTemplate(ctx context.Context, templateName string) error {
	prm.logger.Debug("Unregistering template from MCP resources", zap.String("name", templateName))

	uri := prm.generateTemplateURI(templateName)
	delete(prm.cache, uri)

	prm.logger.Info("Template unregistered from MCP resources", zap.String("name", templateName))
	return nil
}

// RefreshCache clears the resource cache to force reload
func (prm *PromptResourceManager) RefreshCache() {
	prm.logger.Debug("Refreshing prompt resource cache")
	prm.cache = make(map[string]*PromptResource)
	prm.lastCacheTime = time.Time{}
}

// GetResourceMetadata retrieves metadata for a resource without full content
func (prm *PromptResourceManager) GetResourceMetadata(ctx context.Context, uri string) (map[string]interface{}, error) {
	resource, err := prm.GetResource(ctx, uri)
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"name":        resource.Name,
		"description": resource.Description,
		"category":    resource.Category,
		"tags":        resource.Tags,
		"parameters":  len(resource.Parameters),
		"mime_type":   resource.MimeType,
	}

	if resource.Metadata != nil {
		for k, v := range resource.Metadata {
			metadata[k] = v
		}
	}

	return metadata, nil
}

// SearchResources searches for resources by name, category, or tags
func (prm *PromptResourceManager) SearchResources(ctx context.Context, query string) ([]PromptResource, error) {
	prm.logger.Debug("Searching prompt resources", zap.String("query", query))

	allResources, err := prm.ListResources(ctx)
	if err != nil {
		return nil, err
	}

	var matchingResources []PromptResource
	queryLower := strings.ToLower(query)

	for _, resource := range allResources {
		// Search in name, description, category, and tags
		if strings.Contains(strings.ToLower(resource.Name), queryLower) ||
			strings.Contains(strings.ToLower(resource.Description), queryLower) ||
			strings.Contains(strings.ToLower(resource.Category), queryLower) {
			matchingResources = append(matchingResources, resource)
			continue
		}

		// Search in tags
		for _, tag := range resource.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				matchingResources = append(matchingResources, resource)
				break
			}
		}
	}

	prm.logger.Debug("Found matching resources", 
		zap.String("query", query), 
		zap.Int("matches", len(matchingResources)))

	return matchingResources, nil
}

// templateToResource converts a prompt.Template to a PromptResource
func (prm *PromptResourceManager) templateToResource(template prompt.Template) *PromptResource {
	// Convert prompt parameters to MCP template parameters
	var mcpParams []TemplateParameter
	for _, param := range template.Parameters {
		mcpParams = append(mcpParams, TemplateParameter{
			Name:        param.Name,
			Description: param.Description,
			Type:        param.Type,
			Required:    param.Required,
			Default:     param.Default,
			Options:     param.Options,
			Placeholder: param.Placeholder,
			Validation:  param.Validation,
		})
	}

	// Generate metadata
	metadata := map[string]interface{}{
		"author":     template.Author,
		"version":    template.Version,
		"created_at": template.CreatedAt,
		"updated_at": template.UpdatedAt,
		"file_path":  template.FilePath,
		"language":   template.Language,
		"framework":  template.Framework,
		"examples":   template.Examples,
	}

	return &PromptResource{
		URI:         prm.generateTemplateURI(template.Name),
		Name:        template.Name,
		Description: template.Description,
		MimeType:    "text/plain",
		Text:        template.Content,
		Category:    template.Category,
		Tags:        template.Tags,
		Parameters:  mcpParams,
		Metadata:    metadata,
	}
}

// generateTemplateURI generates a unique URI for a template
func (prm *PromptResourceManager) generateTemplateURI(templateName string) string {
	// Use prompt:// scheme for prompt resources
	return fmt.Sprintf("prompt://templates/%s", templateName)
}

// extractTemplateNameFromURI extracts the template name from a URI
func (prm *PromptResourceManager) extractTemplateNameFromURI(uri string) string {
	if !strings.HasPrefix(uri, "prompt://templates/") {
		return ""
	}
	
	name := strings.TrimPrefix(uri, "prompt://templates/")
	// Remove any query parameters or fragments
	if idx := strings.IndexAny(name, "?#"); idx != -1 {
		name = name[:idx]
	}
	
	return name
}

// ValidateResourceURI validates if a URI is a valid prompt resource URI
func (prm *PromptResourceManager) ValidateResourceURI(uri string) bool {
	return strings.HasPrefix(uri, "prompt://templates/") && prm.extractTemplateNameFromURI(uri) != ""
}

// GetResourceStats returns statistics about prompt resources
func (prm *PromptResourceManager) GetResourceStats(ctx context.Context) (map[string]interface{}, error) {
	resources, err := prm.ListResources(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_resources": len(resources),
		"categories":      make(map[string]int),
		"cache_size":      len(prm.cache),
		"cache_expiry":    prm.cacheExpiry.String(),
		"last_cache_time": prm.lastCacheTime,
	}

	// Count by category
	categories := stats["categories"].(map[string]int)
	for _, resource := range resources {
		categories[resource.Category]++
	}

	return stats, nil
}

// WatchTemplateChanges sets up watching for template file changes (placeholder for future implementation)
func (prm *PromptResourceManager) WatchTemplateChanges(ctx context.Context, callback func(string, string)) error {
	// This would integrate with file system watching to detect template changes
	// and automatically refresh the cache and notify MCP clients
	prm.logger.Info("Template change watching not yet implemented")
	return nil
}

// ExportResource exports a resource to a specific format
func (prm *PromptResourceManager) ExportResource(ctx context.Context, uri, format string) ([]byte, error) {
	resource, err := prm.GetResource(ctx, uri)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "json":
		return json.Marshal(resource)
	case "yaml":
		// Would need yaml package
		return []byte(resource.Text), nil
	case "text", "plain":
		return []byte(resource.Text), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}