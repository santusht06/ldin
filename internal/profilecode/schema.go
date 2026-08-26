// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package profilecode

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ProfileAsCode represents declarative LinkedIn profile state in YAML
type ProfileAsCode struct {
	Version        string          `yaml:"version,omitempty" json:"version,omitempty"`
	Name           string          `yaml:"name" json:"name"`
	Headline       string          `yaml:"headline" json:"headline"`
	Location       string          `yaml:"location,omitempty" json:"location,omitempty"`
	Industry       string          `yaml:"industry,omitempty" json:"industry,omitempty"`
	About          string          `yaml:"about" json:"about"`
	CustomURL      string          `yaml:"custom_url,omitempty" json:"custom_url,omitempty"`
	Skills         []string        `yaml:"skills" json:"skills"`
	Experience     []Experience    `yaml:"experience" json:"experience"`
	Education      []Education     `yaml:"education,omitempty" json:"education,omitempty"`
	Projects       []Project       `yaml:"projects,omitempty" json:"projects,omitempty"`
	Certifications []Certification `yaml:"certifications,omitempty" json:"certifications,omitempty"`
	Languages      []string        `yaml:"languages,omitempty" json:"languages,omitempty"`
	ContactInfo    *ContactInfo    `yaml:"contact_info,omitempty" json:"contact_info,omitempty"`
}

// Experience item
type Experience struct {
	Company      string   `yaml:"company" json:"company"`
	Role         string   `yaml:"role" json:"role"`
	Employment   string   `yaml:"employment_type,omitempty" json:"employment_type,omitempty"` // Full-time, Contract, etc.
	Location     string   `yaml:"location,omitempty" json:"location,omitempty"`
	StartDate    string   `yaml:"start_date" json:"start_date"` // YYYY-MM
	EndDate      string   `yaml:"end_date,omitempty" json:"end_date,omitempty"` // YYYY-MM or "Present"
	Current      bool     `yaml:"current,omitempty" json:"current,omitempty"`
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	SkillsUsed   []string `yaml:"skills_used,omitempty" json:"skills_used,omitempty"`
}

// Education item
type Education struct {
	School       string `yaml:"school" json:"school"`
	Degree       string `yaml:"degree,omitempty" json:"degree,omitempty"`
	FieldOfStudy string `yaml:"field_of_study,omitempty" json:"field_of_study,omitempty"`
	StartDate    string `yaml:"start_date,omitempty" json:"start_date,omitempty"`
	EndDate      string `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	Grade        string `yaml:"grade,omitempty" json:"grade,omitempty"`
	Activities   string `yaml:"activities,omitempty" json:"activities,omitempty"`
}

// Project item
type Project struct {
	Name         string   `yaml:"name" json:"name"`
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	URL          string   `yaml:"url,omitempty" json:"url,omitempty"`
	StartDate    string   `yaml:"start_date,omitempty" json:"start_date,omitempty"`
	EndDate      string   `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	Technologies []string `yaml:"technologies,omitempty" json:"technologies,omitempty"`
}

// Certification item
type Certification struct {
	Name           string `yaml:"name" json:"name"`
	IssuingOrg     string `yaml:"issuing_organization" json:"issuing_organization"`
	IssueDate      string `yaml:"issue_date,omitempty" json:"issue_date,omitempty"`
	ExpirationDate string `yaml:"expiration_date,omitempty" json:"expiration_date,omitempty"`
	CredentialID   string `yaml:"credential_id,omitempty" json:"credential_id,omitempty"`
	CredentialURL  string `yaml:"credential_url,omitempty" json:"credential_url,omitempty"`
}

// ContactInfo item
type ContactInfo struct {
	Email    string   `yaml:"email,omitempty" json:"email,omitempty"`
	Websites []string `yaml:"websites,omitempty" json:"websites,omitempty"`
	Twitter  string   `yaml:"twitter,omitempty" json:"twitter,omitempty"`
	GitHub   string   `yaml:"github,omitempty" json:"github,omitempty"`
}

// LoadProfileFile parses a profile YAML file from disk
func LoadProfileFile(path string) (*ProfileAsCode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p ProfileAsCode
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveProfileFile writes a profile struct to YAML file
func SaveProfileFile(path string, p *ProfileAsCode) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ToYAML converts struct to clean YAML string
func (p *ProfileAsCode) ToYAML() (string, error) {
	data, err := yaml.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
