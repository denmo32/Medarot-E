package main

import "fmt"

// GameDataManager holds all static game data definitions, like parts and medals.
type GameDataManager struct {
	partDefinitions  map[string]*PartDefinition
	medalDefinitions map[string]*Medal // Assuming Medal struct is mostly definition for now
	// Other game data definitions can be added here
}

// NewGameDataManager creates a new instance of GameDataManager.
func NewGameDataManager() *GameDataManager {
	return &GameDataManager{
		partDefinitions:  make(map[string]*PartDefinition),
		medalDefinitions: make(map[string]*Medal),
	}
}

// AddPartDefinition adds a part definition to the manager.
func (gdm *GameDataManager) AddPartDefinition(pd *PartDefinition) error {
	if pd == nil {
		return fmt.Errorf("cannot add nil PartDefinition")
	}
	if _, exists := gdm.partDefinitions[pd.ID]; exists {
		return fmt.Errorf("PartDefinition with ID %s already exists", pd.ID)
	}
	gdm.partDefinitions[pd.ID] = pd
	return nil
}

// GetPartDefinition retrieves a part definition by its ID.
func (gdm *GameDataManager) GetPartDefinition(id string) (*PartDefinition, bool) {
	pd, found := gdm.partDefinitions[id]
	return pd, found
}

// AddMedalDefinition adds a medal definition to the manager.
func (gdm *GameDataManager) AddMedalDefinition(md *Medal) error {
	if md == nil {
		return fmt.Errorf("cannot add nil MedalDefinition")
	}
	if _, exists := gdm.medalDefinitions[md.ID]; exists {
		return fmt.Errorf("MedalDefinition with ID %s already exists", md.ID)
	}
	gdm.medalDefinitions[md.ID] = md
	return nil
}

// GetMedalDefinition retrieves a medal definition by its ID.
func (gdm *GameDataManager) GetMedalDefinition(id string) (*Medal, bool) {
	md, found := gdm.medalDefinitions[id]
	return md, found
}

// Global instance of GameDataManager (or it could be passed around, e.g. via SharedResources)
// For simplicity in this refactoring phase, a global instance can be used.
// Consider dependency injection for a larger application.
var GlobalGameDataManager = NewGameDataManager()
