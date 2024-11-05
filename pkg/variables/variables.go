// Package variables is a thin wrapper over the sync.Map package
//
// TODO: This implementation may not really be necessary anymore since we denormalize the graph
// any/all vars and env vars will have their own space
package variables

import (
	"sync"
)

// Container is an interface of variables container.
// Is is simple key-value structure.
type Container interface {
	Set(string, interface{})
	Get(string) interface{}
	Has(string) bool
	Map() map[string]interface{}
	Merge(Container) Container
	With(string, interface{}) Container
}

// Variables is struct containing simple key-value string values
type Variables struct {
	m sync.Map
}

// NewVariables creates new Variables instance
func NewVariables() *Variables {
	return &Variables{}
}

// FromMap creates new Variables instance from given map
func FromMap(values map[string]string) Container {
	vars := &Variables{}
	for k, v := range values {
		vars.m.Store(k, v)
	}

	return vars
}

// Set stores value with given key
func (vars *Variables) Set(key string, value any) {
	vars.m.Store(key, value)
}

// Get returns value by given key
func (vars *Variables) Get(key string) any {
	v, ok := vars.m.Load(key)
	if !ok {
		return ""
	}

	return v
}

// Has checks if value exists
func (vars *Variables) Has(name string) bool {
	_, ok := vars.m.Load(name)
	return ok
}

// Map returns container in map[string]string form
func (vars *Variables) Map() map[string]interface{} {
	m := make(map[string]interface{})
	vars.m.Range(func(key, value interface{}) bool {
		m[key.(string)] = value
		return true
	})
	return m
}

// Merge merges into current container with the src Container
// src will overwrite the existing keys if exists
// returns a new Container
func (vars *Variables) Merge(src Container) Container {
	dst := &Variables{}

	if vars != nil {
		for k, v := range vars.Map() {
			dst.Set(k, v)
		}
	}

	for k, v := range src.Map() {
		dst.Set(k, v)
	}

	return dst
}

// With creates new container and sets key to given value
func (vars *Variables) With(key string, value interface{}) Container {
	dst := &Variables{}
	dst = dst.Merge(vars).(*Variables)
	dst.Set(key, value)

	return dst
}
