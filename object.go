package otto

import (
	"sync"
)

type _object struct {
	runtime *_runtime

	class       string
	objectClass *_objectClass
	value       interface{}

	prototype  *_object
	extensible bool

	property      map[string]_property
	propertyOrder []string

	mu sync.RWMutex

	cyclicalCount int
}

func newObject(runtime *_runtime, class string) *_object {
	self := &_object{
		runtime:     runtime,
		class:       class,
		objectClass: _classObject,
		property:    make(map[string]_property),
		extensible:  true,
	}
	return self
}

func (self *_object) MemUsage(ctx *MemUsageContext) (uint64, error) {
	if ctx.IsObjVisited(self) {
		return 0, nil
	}
	ctx.VisitObj(self)
	total := EmptySize
	self.mu.RLock()
	defer self.mu.RUnlock()
	for k, v := range self.property {
		if val, ok := v.value.(Value); ok {
			inc, err := val.MemUsage(ctx)
			total += inc
			total += uint64(len(k)) // count size of property name towards total object size.
			if err != nil {
				return total, err
			}
		} else {
			// most likely a propertyGetSet. ignore for now.
		}
	}
	return total, nil

}

// 8.12

// 8.12.1
func (self *_object) getOwnProperty(name string) *_property {
	return self.objectClass.getOwnProperty(self, name)
}

// 8.12.2
func (self *_object) getProperty(name string) *_property {
	return self.objectClass.getProperty(self, name)
}

// 8.12.3
func (self *_object) get(name string) Value {
	return self.objectClass.get(self, name)
}

// 8.12.4
func (self *_object) canPut(name string) bool {
	return self.objectClass.canPut(self, name)
}

// 8.12.5
func (self *_object) put(name string, value Value, throw bool) {
	self.objectClass.put(self, name, value, throw)
}

// 8.12.6
func (self *_object) hasProperty(name string) bool {
	return self.objectClass.hasProperty(self, name)
}

func (self *_object) hasOwnProperty(name string) bool {
	return self.objectClass.hasOwnProperty(self, name)
}

type _defaultValueHint int

const (
	defaultValueNoHint _defaultValueHint = iota
	defaultValueHintString
	defaultValueHintNumber
	defaultValueHintSymbol
)

// 8.12.8
func (self *_object) DefaultValue(hint _defaultValueHint) Value {
	if hint == defaultValueNoHint {
		if self.class == "Date" {
			// Date exception
			hint = defaultValueHintString
		} else {
			hint = defaultValueHintNumber
		}
	}

	var methodSequence []string
	switch hint {
	case defaultValueHintString:
		methodSequence = []string{"toString", "valueOf"}
	case defaultValueHintSymbol:
		methodSequence = []string{"toValueString", "toString", "valueOf"}
	default:
		methodSequence = []string{"valueOf", "toString"}
	}

	for _, methodName := range methodSequence {
		method := self.get(methodName)
		// FIXME This is redundant...
		if method.isCallable() {
			result := method._object().call(toValue_object(self), nil, false, nativeFrame)
			if result.IsPrimitive() {
				return result
			}
		}
	}

	panic(self.runtime.panicTypeError())
}

func (self *_object) String() string {
	return self.DefaultValue(defaultValueHintString).string()
}

func (self *_object) defineProperty(name string, value Value, mode _propertyMode, throw bool) bool {
	return self.defineOwnProperty(name, _property{value, mode}, throw)
}

// 8.12.9
func (self *_object) defineOwnProperty(name string, descriptor _property, throw bool) bool {
	return self.objectClass.defineOwnProperty(self, name, descriptor, throw)
}

func (self *_object) delete(name string, throw bool) bool {
	return self.objectClass.delete(self, name, throw)
}

func (self *_object) enumerate(all bool, each func(string) bool) {
	self.objectClass.enumerate(self, all, each)
}

func (self *_object) _exists(name string) bool {
	self.mu.RLock()
	defer self.mu.RUnlock()
	_, exists := self.property[name]
	return exists
}

func (self *_object) _read(name string) (_property, bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	property, exists := self.property[name]
	return property, exists
}

func (self *_object) _write(name string, value interface{}, mode _propertyMode) {
	self.mu.Lock()
	defer self.mu.Unlock()
	if value == nil {
		value = Value{}
	}
	_, exists := self.property[name]
	self.property[name] = _property{value, mode}
	if !exists {
		self.propertyOrder = append(self.propertyOrder, name)
	}
}

func (self *_object) _incCyclicalCount() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.cyclicalCount++
}

func (self *_object) _decCyclicalCount() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.cyclicalCount--
}

func (self *_object) _delete(name string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	_, exists := self.property[name]
	delete(self.property, name)
	if exists {
		for index, property := range self.propertyOrder {
			if name == property {
				if index == len(self.propertyOrder)-1 {
					self.propertyOrder = self.propertyOrder[:index]
				} else {
					self.propertyOrder = append(self.propertyOrder[:index], self.propertyOrder[index+1:]...)
				}
			}
		}
	}
}
