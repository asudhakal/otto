package otto

import (
	"fmt"
)

// Object

func builtinObject(call FunctionCall) Value {
	value := call.Argument(0)
	switch value.kind {
	case valueUndefined, valueNull:
		return toValue_object(call.runtime.newObject())
	}

	return toValue_object(call.runtime.toObject(value))
}

func builtinNewObject(self *_object, argumentList []Value) Value {
	value := valueOfArrayIndex(argumentList, 0)
	switch value.kind {
	case valueNull, valueUndefined:
	case valueNumber, valueString, valueBoolean:
		return toValue_object(self.runtime.toObject(value))
	case valueObject:
		return value
	default:
	}
	return toValue_object(self.runtime.newObject())
}

func builtinObject_valueOf(call FunctionCall) Value {
	return toValue_object(call.thisObject())
}

func builtinObject_hasOwnProperty(call FunctionCall) Value {
	propertyName := call.Argument(0).string()
	thisObject := call.thisObject()
	return toValue_bool(thisObject.hasOwnProperty(propertyName))
}

func builtinObject_defineSetter(call FunctionCall) Value {
	objectValue := call.This
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}
	name := call.Argument(0).string()
	descriptor := _property{}
	descriptor.mode = 0211
	descriptor.value = _propertyGetSet{nil, call.Argument(1)._object()}
	object.defineOwnProperty(name, descriptor, true)
	return objectValue
}

func builtinObject_isPrototypeOf(call FunctionCall) Value {
	value := call.Argument(0)
	if !value.IsObject() {
		return falseValue
	}
	prototype := call.toObject(value).prototype
	thisObject := call.thisObject()
	for prototype != nil {
		if thisObject == prototype {
			return trueValue
		}
		prototype = prototype.prototype
	}
	return falseValue
}

func builtinObject_propertyIsEnumerable(call FunctionCall) Value {
	propertyName := call.Argument(0).string()
	thisObject := call.thisObject()
	property := thisObject.getOwnProperty(propertyName)
	if property != nil && property.enumerable() {
		return trueValue
	}
	return falseValue
}

func builtinObject_toString(call FunctionCall) Value {
	result := ""
	if call.This.IsUndefined() {
		result = "[object Undefined]"
	} else if call.This.IsNull() {
		result = "[object Null]"
	} else {
		toStringTagProp := call.thisObject().getProperty(symbolToStringTagPropertyName)
		var className string
		if toStringTagProp != nil {
			className = toStringTagProp.get(call.thisObject()).String()
		}

		if len(className) > 0 && className != "undefined" {
			result = fmt.Sprintf("[object %s]", className)
		} else {
			result = fmt.Sprintf("[object %s]", call.thisObject().class)
		}
	}
	return toValue_string(result)
}

func builtinObject_toLocaleString(call FunctionCall) Value {
	toString := call.thisObject().get("toString")
	if !toString.isCallable() {
		panic(call.runtime.panicTypeError())
	}
	return toString.call(call.runtime, call.This)
}

func builtinObject_getPrototypeOf(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}

	if object.prototype == nil {
		return nullValue
	}

	return toValue_object(object.prototype)
}

func builtinObject_setPrototypeOf(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}

	prototypeValue := call.Argument(1)
	if !prototypeValue.IsNull() && !prototypeValue.IsObject() {
		panic(call.runtime.panicTypeError("Object prototype may only be an Object or null"))
	}

	object.prototype = prototypeValue._object()

	return toValue_object(object)
}

func builtinObject_getOwnPropertyDescriptor(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}

	name := call.Argument(1).string()
	descriptor := object.getOwnProperty(name)
	if descriptor == nil {
		if name != symbolToStringTagPropertyName {
			return Value{}
		}

		descriptor := _property{}
		descriptor.mode = 0211
		descriptor.value = _propertyGetSet{
			&_object{
				runtime:     call.runtime,
				class:       "Function",
				objectClass: _classObject,
				prototype:   call.runtime.global.FunctionPrototype,
				extensible:  true,
				property: map[string]_property{
					"length": _property{
						mode: 0,
						value: Value{
							kind:  valueNumber,
							value: 0,
						},
					},
				},
				propertyOrder: []string{
					"length",
				},
				value: _nativeFunctionObject{
					name: "get",
					call: func(call FunctionCall) Value {
						return UndefinedValue()
					},
				},
			},
			nil,
		}
		result := call.runtime.fromPropertyDescriptor(descriptor)
		result.defineOwnProperty(name, descriptor, false)
		return toValue_object(result)
	}
	return toValue_object(call.runtime.fromPropertyDescriptor(*descriptor))
}

func builtinObject_defineProperty(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}
	name := call.Argument(1).string()
	descriptor := toPropertyDescriptor(call.runtime, call.Argument(2))
	object.defineOwnProperty(name, descriptor, true)
	return objectValue
}

func builtinObject_defineProperties(call FunctionCall) Value {
	objectValue := call.Argument(0)
	object := objectValue._object()
	if object == nil {
		panic(call.runtime.panicTypeError())
	}

	properties := call.runtime.toObject(call.Argument(1))
	properties.enumerate(false, func(name string) bool {
		descriptor := toPropertyDescriptor(call.runtime, properties.get(name))
		object.defineOwnProperty(name, descriptor, true)
		return true
	})

	return objectValue
}

func builtinObject_create(call FunctionCall) Value {
	prototypeValue := call.Argument(0)
	if !prototypeValue.IsNull() && !prototypeValue.IsObject() {
		panic(call.runtime.panicTypeError())
	}

	object := call.runtime.newObject()
	object.prototype = prototypeValue._object()

	propertiesValue := call.Argument(1)
	if propertiesValue.IsDefined() {
		properties := call.runtime.toObject(propertiesValue)
		properties.enumerate(false, func(name string) bool {
			descriptor := toPropertyDescriptor(call.runtime, properties.get(name))
			object.defineOwnProperty(name, descriptor, true)
			return true
		})
	}

	return toValue_object(object)
}

func builtinObject_isExtensible(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		return toValue_bool(object.extensible)
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_preventExtensions(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.extensible = false
	} else {
		panic(call.runtime.panicTypeError())
	}
	return object
}

func builtinObject_isSealed(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		if object.extensible {
			return toValue_bool(false)
		}
		result := true
		object.enumerate(true, func(name string) bool {
			property := object.getProperty(name)
			if property.configurable() {
				result = false
			}
			return true
		})
		return toValue_bool(result)
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_seal(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.enumerate(true, func(name string) bool {
			if property := object.getOwnProperty(name); nil != property && property.configurable() {
				property.configureOff()
				object.defineOwnProperty(name, *property, true)
			}
			return true
		})
		object.extensible = false
	} else {
		panic(call.runtime.panicTypeError())
	}
	return object
}

func builtinObject_isFrozen(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		if object.extensible {
			return toValue_bool(false)
		}
		result := true
		object.enumerate(true, func(name string) bool {
			property := object.getProperty(name)
			if property.configurable() || property.writable() {
				result = false
			}
			return true
		})
		return toValue_bool(result)
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_freeze(call FunctionCall) Value {
	object := call.Argument(0)
	if object := object._object(); object != nil {
		object.enumerate(true, func(name string) bool {
			if property, update := object.getOwnProperty(name), false; nil != property {
				if property.isDataDescriptor() && property.writable() {
					property.writeOff()
					update = true
				}
				if property.configurable() {
					property.configureOff()
					update = true
				}
				if update {
					object.defineOwnProperty(name, *property, true)
				}
			}
			return true
		})
		object.extensible = false
	} else {
		panic(call.runtime.panicTypeError())
	}
	return object
}

func builtinObject_keys(call FunctionCall) Value {
	if object, keys := call.Argument(0)._object(), []Value(nil); nil != object {
		object.enumerate(false, func(name string) bool {
			keys = append(keys, toValue_string(name))
			return true
		})
		return toValue_object(call.runtime.newArrayOf(keys))
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_values(call FunctionCall) Value {
	if object, values := call.Argument(0)._object(), []Value(nil); nil != object {
		object.enumerate(false, func(name string) bool {
			values = append(values, object.get(name))
			return true
		})
		return toValue_object(call.runtime.newArrayOf(values))
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_entries(call FunctionCall) Value {
	if object := call.Argument(0)._object(); nil != object {
		var entries []Value

		object.enumerate(false, func(name string) bool {
			entries = append(entries, toValue_object(call.runtime.newArrayOf([]Value{
				toValue_string(name),
				object.get(name),
			})))
			return true
		})

		return toValue_object(call.runtime.newArrayOf(entries))
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_fromEntries(call FunctionCall) Value {
	res := call.runtime.newObject()
	if object := call.Argument(0)._object(); nil != object {
		if !isArray(object) {
			panic(call.runtime.panicTypeError())
		}
		length := int64(objectLength(object))
		for i := int64(0); i < length; i += 1 {
			entry := object.get(arrayIndexToString(i))._object()
			if !isArray(entry) {
				panic(call.runtime.panicTypeError())
			}
			key := entry.get("0").string()
			value := entry.get("1")
			res.put(key, value, true)
		}
		return toValue_object(res)
	}
	panic(call.runtime.panicTypeError())
}

func builtinObject_getOwnPropertyNames(call FunctionCall) Value {
	if object, propertyNames := call.Argument(0)._object(), []Value(nil); nil != object {
		object.enumerate(true, func(name string) bool {
			if object.hasOwnProperty(name) {
				propertyNames = append(propertyNames, toValue_string(name))
			}
			return true
		})
		return toValue_object(call.runtime.newArrayOf(propertyNames))
	}
	panic(call.runtime.panicTypeError())
}
