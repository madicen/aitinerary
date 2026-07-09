package aiconfig

import (
	"fmt"
	"strings"
)

// ProfileList returns every configured profile for UI pickers (genmenu, settings).
func (c *Config) ProfileList() []Profile {
	if c == nil || len(c.Profiles) == 0 {
		return []Profile{c.snapshotProfile(DefaultProfileName)}
	}
	out := make([]Profile, len(c.Profiles))
	copy(out, c.Profiles)
	return out
}

// Active returns a copy of the active profile.
func (c *Config) Active() Profile {
	if c == nil {
		return Profile{Name: DefaultProfileName, Provider: ProviderOllama, Model: DefaultOllamaModel, TimeoutSec: 300}
	}
	if idx, ok := c.findProfileIndex(c.ActiveProfile); ok {
		return c.Profiles[idx]
	}
	if len(c.Profiles) > 0 {
		return c.Profiles[0]
	}
	return c.snapshotProfile(DefaultProfileName)
}

// SetActive switches to the named profile.
func (c *Config) SetActive(name string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	idx, ok := c.findProfileIndex(name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	c.ActiveProfile = c.Profiles[idx].Name
	c.applyActiveProfile()
	return nil
}

// AddProfile appends p to the profile list.
func (c *Config) AddProfile(p Profile) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("profile name is empty")
	}
	if _, ok := c.findProfileIndex(name); ok {
		return fmt.Errorf("profile %q already exists", name)
	}
	p.Name = name
	c.Profiles = append(c.Profiles, p)
	return nil
}

// UpdateProfile replaces the profile at the given name.
func (c *Config) UpdateProfile(name string, p Profile) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	idx, ok := c.findProfileIndex(name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	if strings.TrimSpace(p.Name) == "" {
		p.Name = c.Profiles[idx].Name
	}
	if !strings.EqualFold(p.Name, c.Profiles[idx].Name) {
		if other, exists := c.findProfileIndex(p.Name); exists && other != idx {
			return fmt.Errorf("profile %q already exists", p.Name)
		}
	}
	wasActive := strings.EqualFold(c.ActiveProfile, c.Profiles[idx].Name)
	c.Profiles[idx] = p
	if wasActive {
		c.ActiveProfile = p.Name
		c.applyActiveProfile()
	}
	return nil
}

// DeleteProfile removes the named profile.
func (c *Config) DeleteProfile(name string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if len(c.Profiles) <= 1 {
		return fmt.Errorf("cannot delete the last profile")
	}
	idx, ok := c.findProfileIndex(name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	wasActive := strings.EqualFold(c.ActiveProfile, c.Profiles[idx].Name)
	c.Profiles = append(c.Profiles[:idx], c.Profiles[idx+1:]...)
	if wasActive {
		c.ActiveProfile = c.Profiles[0].Name
		c.applyActiveProfile()
	}
	return nil
}

// RenameProfile renames an existing profile.
func (c *Config) RenameProfile(oldName, newName string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("new profile name is empty")
	}
	idx, ok := c.findProfileIndex(oldName)
	if !ok {
		return fmt.Errorf("profile %q not found", oldName)
	}
	if other, exists := c.findProfileIndex(newName); exists && other != idx {
		return fmt.Errorf("profile %q already exists", newName)
	}
	wasActive := strings.EqualFold(c.ActiveProfile, c.Profiles[idx].Name)
	c.Profiles[idx].Name = newName
	if wasActive {
		c.ActiveProfile = newName
	}
	return nil
}

// CycleActive moves the active pointer by delta through the profile list.
func (c *Config) CycleActive(delta int) {
	if c == nil || len(c.Profiles) < 2 {
		return
	}
	idx, ok := c.findProfileIndex(c.ActiveProfile)
	if !ok {
		idx = 0
	}
	n := len(c.Profiles)
	idx = ((idx+delta)%n + n) % n
	c.ActiveProfile = c.Profiles[idx].Name
	c.applyActiveProfile()
}
