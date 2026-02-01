package vgofile

import (
	"fmt"
)

// Control represents a visual control on a form
type Control struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Left       int                    `json:"left"`
	Top        int                    `json:"top"`
	Width      int                    `json:"width"`
	Height     int                    `json:"height"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Events     map[string]string      `json:"events,omitempty"`
}

// NewControl creates a new control with the given type
func NewControl(controlType string) *Control {
	return &Control{
		Type:       controlType,
		Width:      100,
		Height:     25,
		Properties: make(map[string]interface{}),
		Events:     make(map[string]string),
	}
}

// Clone creates a deep copy of the control
func (c *Control) Clone() *Control {
	clone := &Control{
		Name:       c.Name,
		Type:       c.Type,
		Left:       c.Left,
		Top:        c.Top,
		Width:      c.Width,
		Height:     c.Height,
		Properties: make(map[string]interface{}),
		Events:     make(map[string]string),
	}
	for k, v := range c.Properties {
		clone.Properties[k] = v
	}
	for k, v := range c.Events {
		clone.Events[k] = v
	}
	return clone
}

// SetProperty sets a property value
func (c *Control) SetProperty(name string, value interface{}) {
	// Handle built-in properties
	switch name {
	case "Name":
		if s, ok := value.(string); ok {
			c.Name = s
		}
	case "Left":
		if i, ok := value.(int); ok {
			c.Left = i
		}
	case "Top":
		if i, ok := value.(int); ok {
			c.Top = i
		}
	case "Width":
		if i, ok := value.(int); ok {
			c.Width = i
		}
	case "Height":
		if i, ok := value.(int); ok {
			c.Height = i
		}
	default:
		c.Properties[name] = value
	}
}

// GetProperty gets a property value
func (c *Control) GetProperty(name string) interface{} {
	switch name {
	case "Name":
		return c.Name
	case "Type":
		return c.Type
	case "Left":
		return c.Left
	case "Top":
		return c.Top
	case "Width":
		return c.Width
	case "Height":
		return c.Height
	default:
		return c.Properties[name]
	}
}

// GetStringProperty gets a string property with default value
func (c *Control) GetStringProperty(name string, defaultVal string) string {
	if v, ok := c.Properties[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetIntProperty gets an int property with default value
func (c *Control) GetIntProperty(name string, defaultVal int) int {
	if v, ok := c.Properties[name]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultVal
}

// GetBoolProperty gets a bool property with default value
func (c *Control) GetBoolProperty(name string, defaultVal bool) bool {
	if v, ok := c.Properties[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// SetEvent sets an event handler
func (c *Control) SetEvent(eventName, handlerName string) {
	c.Events[eventName] = handlerName
}

// GetEvent gets an event handler name
func (c *Control) GetEvent(eventName string) string {
	return c.Events[eventName]
}

// String returns a string representation of the control
func (c *Control) String() string {
	return fmt.Sprintf("%s (%s) at (%d,%d) size %dx%d", c.Name, c.Type, c.Left, c.Top, c.Width, c.Height)
}
