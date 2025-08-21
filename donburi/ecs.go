package donburi

import (
	"sync/atomic"
)

// --- Entity & World ---

// Entity is a unique identifier for an entity.
type Entity uint32

// Entry is a handle to an entity in a world.
// It provides methods to manipulate the entity's components.
type Entry struct {
	world  *World
	entity Entity
}

// Valid returns true if the entity is still valid in the world.
func (e *Entry) Valid() bool {
	return e.world.Valid(e.entity)
}

// HasComponent returns true if the entity has the given component.
func (e *Entry) HasComponent(ct ComponentType) bool {
	return e.world.HasComponent(e.entity, ct)
}

// Entity returns the underlying Entity ID.
func (e *Entry) Entity() Entity {
	return e.entity
}

// World manages all entities and their components.
type World struct {
	nextEntityID    uint32
	entities        map[Entity]bool
	componentStores map[uint32]map[Entity]any // Stores pointers to components: map[componentId]map[entityId]any
}

// NewWorld creates a new World.
func NewWorld() World {
	return World{
		nextEntityID:    0,
		entities:        make(map[Entity]bool),
		componentStores: make(map[uint32]map[Entity]any),
	}
}

// Create creates a new entity with the given components.
func (w *World) Create(components ...ComponentType) Entity {
	id := atomic.AddUint32(&w.nextEntityID, 1)
	entity := Entity(id)
	w.entities[entity] = true

	// Attach components with a dummy value.
	// This is needed for tag components to be detected by HasComponent.
	// For components with data, `SetValue` will overwrite this later.
	for _, ct := range components {
		w.SetComponent(entity, ct, &struct{}{})
	}

	return entity
}

// Entry returns an entry for the given entity.
func (w *World) Entry(id Entity) *Entry {
	if !w.Valid(id) {
		return nil
	}
	return &Entry{
		world:  w,
		entity: id,
	}
}

// Valid returns true if the entity is still valid in the world.
func (w *World) Valid(id Entity) bool {
	_, ok := w.entities[id]
	return ok
}

// Remove deletes an entity and all its components from the world.
func (w *World) Remove(id Entity) {
	if !w.Valid(id) {
		return
	}
	delete(w.entities, id)
	for _, store := range w.componentStores {
		delete(store, id)
	}
}

// HasComponent returns true if the entity has the given component.
func (w *World) HasComponent(id Entity, ct ComponentType) bool {
	store, ok := w.componentStores[ct.id()]
	if !ok {
		return false
	}
	_, ok = store[id]
	return ok
}

// SetComponent adds or updates a component for the given entity.
// It stores a pointer to the data.
func (w *World) SetComponent(id Entity, ct ComponentType, data any) {
	if !w.Valid(id) {
		return
	}
	store, ok := w.componentStores[ct.id()]
	if !ok {
		store = make(map[Entity]any)
		w.componentStores[ct.id()] = store
	}
	store[id] = data
}

// GetComponent returns the component data for the given entity.
func (w *World) GetComponent(id Entity, ct ComponentType) any {
	store, ok := w.componentStores[ct.id()]
	if !ok {
		return nil
	}
	return store[id]
}

// Entities returns the map of all entities in the world.
func (w *World) Entities() map[Entity]bool {
	return w.entities
}

// --- Component ---

var nextComponentTypeID uint32

// ComponentType represents a type of component.
// It holds a unique ID for the component type.
type ComponentType interface {
	id() uint32
	isComponentType() // A dummy method to make the interface unique
}

// ComponentTypeData is the concrete implementation for ComponentType.
// It uses generics to associate a Go type with the component type.
type ComponentTypeData[T any] struct {
	internalID uint32
}

func (c *ComponentTypeData[T]) id() uint32 {
	return c.internalID
}

func (c *ComponentTypeData[T]) isComponentType() {}

// Get returns the component data for the given entry.
func (c *ComponentTypeData[T]) Get(entry *Entry) *T {
	return Get[T](entry, c)
}

// SetValue sets the component data for the given entry.
func (c *ComponentTypeData[T]) SetValue(entry *Entry, data T) {
	// We store a pointer internally to allow modification.
	entry.world.SetComponent(entry.entity, c, &data)
}

// NewComponentType creates a new component type.
func NewComponentType[T any]() *ComponentTypeData[T] {
	return &ComponentTypeData[T]{
		internalID: atomic.AddUint32(&nextComponentTypeID, 1),
	}
}

// Add adds a component to an entry.
func Add[T any](entry *Entry, ct *ComponentTypeData[T], data *T) {
	entry.world.SetComponent(entry.entity, ct, data)
}

// Get retrieves a component from an entry. It returns a pointer.
func Get[T any](entry *Entry, ct *ComponentTypeData[T]) *T {
	data := entry.world.GetComponent(entry.entity, ct)
	if data == nil {
		var zero *T
		return zero
	}
	// `SetValue` stores *T, so `Get` should return *T.
	val, ok := data.(*T)
	if !ok {
		// This can happen if a component was added via Create (as a tag)
		// but SetValue was never called.
		return nil
	}
	return val
}

// --- Filter ---

// Filter is an interface for filtering entities.
type Filter interface {
	Matches(w *World, id Entity) bool
}

type containsFilter struct {
	components []ComponentType
}

func (f *containsFilter) Matches(w *World, id Entity) bool {
	for _, ct := range f.components {
		if !w.HasComponent(id, ct) {
			return false
		}
	}
	return true
}

// Contains returns a filter that matches entities that have all the given components.
func Contains(components ...ComponentType) Filter {
	return &containsFilter{components: components}
}

type andFilter struct {
	filters []Filter
}

func (f *andFilter) Matches(w *World, id Entity) bool {
	for _, filter := range f.filters {
		if !filter.Matches(w, id) {
			return false
		}
	}
	return true
}

// And returns a filter that matches entities that match all the given filters.
func And(filters ...Filter) Filter {
	return &andFilter{filters: filters}
}

type notFilter struct {
	filter Filter
}

func (f *notFilter) Matches(w *World, id Entity) bool {
	return !f.filter.Matches(w, id)
}

// Not returns a filter that matches entities that do not match the given filter.
func Not(filter Filter) Filter {
	return &notFilter{filter: filter}
}

// --- Query ---

// Query is used to iterate over entities that match a filter.
type Query struct {
	filter Filter
}

// NewQuery creates a new query with the given filter.
func NewQuery(f Filter) *Query {
	return &Query{
		filter: f,
	}
}

// Each iterates over all entities that match the query's filter.
func (q *Query) Each(w World, callback func(entry *Entry)) {
	// The original world is modified via the pointer in the Entry.
	worldPtr := &w

	// Create a copy of entity IDs to iterate over, preventing issues
	// if entities are added/removed during iteration.
	ids := make([]Entity, 0, len(worldPtr.Entities()))
	for id := range worldPtr.Entities() {
		ids = append(ids, id)
	}

	for _, id := range ids {
		// Re-check validity in case it was removed in a previous iteration step
		if !worldPtr.Valid(id) {
			continue
		}
		if q.filter.Matches(worldPtr, id) {
			callback(worldPtr.Entry(id))
		}
	}
}

// First returns the first entity that matches the query's filter.
func (q *Query) First(w World) (*Entry, bool) {
	worldPtr := &w
	// Note: Iteration order is not guaranteed with maps.
	for id := range worldPtr.Entities() {
		if q.filter.Matches(worldPtr, id) {
			return worldPtr.Entry(id), true
		}
	}
	return nil, false
}