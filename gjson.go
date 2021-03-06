// Package gjson provides searching for json strings.
package gjson

import "strconv"

// Type is Result type
type Type int

const (
	// Null is a null json value
	Null Type = iota
	// False is a json false boolean
	False
	// Number is json number
	Number
	// String is a json string
	String
	// True is a json true boolean
	True
	// JSON is a raw block of JSON
	JSON
)

// Result represents a json value that is returned from Get().
type Result struct {
	// Type is the json type
	Type Type
	// Raw is the raw json
	Raw string
	// Str is the json string
	Str string
	// Num is the json number
	Num float64
}

// String returns a string representation of the value.
func (t Result) String() string {
	switch t.Type {
	default:
		return "null"
	case False:
		return "false"
	case Number:
		return strconv.FormatFloat(t.Num, 'f', -1, 64)
	case String:
		return t.Str
	case JSON:
		return t.Raw
	case True:
		return "true"
	}
}

// Bool returns an boolean representation.
func (t Result) Bool() bool {
	switch t.Type {
	default:
		return false
	case True:
		return true
	case String:
		return t.Str != "" && t.Str != "0"
	case Number:
		return t.Num != 0
	}
}

// Int returns an integer representation.
func (t Result) Int() int64 {
	switch t.Type {
	default:
		return 0
	case True:
		return 1
	case String:
		n, _ := strconv.ParseInt(t.Str, 10, 64)
		return n
	case Number:
		return int64(t.Num)
	}
}

// Float returns an float64 representation.
func (t Result) Float() float64 {
	switch t.Type {
	default:
		return 0
	case True:
		return 1
	case String:
		n, _ := strconv.ParseFloat(t.Str, 64)
		return n
	case Number:
		return t.Num
	}
}

// Array returns back an array of children. The result must be a JSON array.
func (t Result) Array() []Result {
	if t.Type != JSON {
		return nil
	}
	a, _, _, _, _ := t.arrayOrMap('[', false)
	return a
}

//  Map returns back an map of children. The result should be a JSON array.
func (t Result) Map() map[string]Result {
	if t.Type != JSON {
		return map[string]Result{}
	}
	_, _, o, _, _ := t.arrayOrMap('{', false)
	return o
}

// Get searches result for the specified path.
// The result should be a JSON array or object.
func (t Result) Get(path string) Result {
	return Get(t.Raw, path)
}

func (t Result) arrayOrMap(vc byte, valueize bool) (
	[]Result,
	[]interface{},
	map[string]Result,
	map[string]interface{},
	byte,
) {
	var a []Result
	var ai []interface{}
	var o map[string]Result
	var oi map[string]interface{}
	var json = t.Raw
	var i int
	var value Result
	var count int
	var key Result
	if vc == 0 {
		for ; i < len(json); i++ {
			if json[i] == '{' || json[i] == '[' {
				vc = json[i]
				i++
				break
			}
			if json[i] > ' ' {
				goto end
			}
		}
	} else {
		for ; i < len(json); i++ {
			if json[i] == vc {
				i++
				break
			}
			if json[i] > ' ' {
				goto end
			}
		}
	}
	if vc == '{' {
		if valueize {
			oi = make(map[string]interface{})
		} else {
			o = make(map[string]Result)
		}
	} else {
		if valueize {
			ai = make([]interface{}, 0)
		} else {
			a = make([]Result, 0)
		}
	}
	for ; i < len(json); i++ {
		if json[i] <= ' ' {
			continue
		}
		// get next value
		if json[i] == ']' || json[i] == '}' {
			break
		}
		switch json[i] {
		default:
			if (json[i] >= '0' && json[i] <= '9') || json[i] == '-' {
				value.Type = Number
				value.Raw, value.Num = tonum(json[i:])
			} else {
				continue
			}
		case '{', '[':
			value.Type = JSON
			value.Raw = squash(json[i:])
		case 'n':
			value.Type = Null
			value.Raw = tolit(json[i:])
		case 't':
			value.Type = True
			value.Raw = tolit(json[i:])
		case 'f':
			value.Type = False
			value.Raw = tolit(json[i:])
		case '"':
			value.Type = String
			value.Raw, value.Str = tostr(json[i:])
		}
		i += len(value.Raw) - 1

		if vc == '{' {
			if count%2 == 0 {
				key = value
			} else {
				if valueize {
					oi[key.Str] = value.Value()
				} else {
					o[key.Str] = value
				}
			}
			count++
		} else {
			if valueize {
				ai = append(ai, value.Value())
			} else {
				a = append(a, value)
			}
		}
	}
end:
	return a, ai, o, oi, vc
}

// Parse parses the json and returns a result
func Parse(json string) Result {
	var value Result
	for i := 0; i < len(json); i++ {
		if json[i] == '{' || json[i] == '[' {
			value.Type = JSON
			value.Raw = json[i:] // just take the entire raw
			break
		}
		if json[i] <= ' ' {
			continue
		}
		switch json[i] {
		default:
			if (json[i] >= '0' && json[i] <= '9') || json[i] == '-' {
				value.Type = Number
				value.Raw, value.Num = tonum(json[i:])
			} else {
				return Result{}
			}
		case 'n':
			value.Type = Null
			value.Raw = tolit(json[i:])
		case 't':
			value.Type = True
			value.Raw = tolit(json[i:])
		case 'f':
			value.Type = False
			value.Raw = tolit(json[i:])
		case '"':
			value.Type = String
			value.Raw, value.Str = tostr(json[i:])
		}
		break
	}
	return value
}

func squash(json string) string {
	// expects that the lead character is a '[' or '{'
	// squash the value, ignoring all nested arrays and objects.
	// the first '[' or '{' has already been read
	depth := 1
	for i := 1; i < len(json); i++ {
		if json[i] >= '"' && json[i] <= '}' {
			switch json[i] {
			case '"':
				i++
				s2 := i
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						// look for an escaped slash
						if json[i-1] == '\\' {
							n := 0
							for j := i - 2; j > s2-1; j-- {
								if json[j] != '\\' {
									break
								}
								n++
							}
							if n%2 == 0 {
								continue
							}
						}
						break
					}
				}
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return json[:i+1]
				}
			}
		}
	}
	return json
}

func tonum(json string) (raw string, num float64) {
	for i := 1; i < len(json); i++ {
		// less than dash might have valid characters
		if json[i] <= '-' {
			if json[i] <= ' ' || json[i] == ',' {
				// break on whitespace and comma
				raw = json[:i]
				num, _ = strconv.ParseFloat(raw, 64)
				return
			}
			// could be a '+' or '-'. let's assume so.
			continue
		}
		if json[i] < ']' {
			// probably a valid number
			continue
		}
		if json[i] == 'e' || json[i] == 'E' {
			// allow for exponential numbers
			continue
		}
		// likely a ']' or '}'
		raw = json[:i]
		num, _ = strconv.ParseFloat(raw, 64)
		return
	}
	raw = json
	num, _ = strconv.ParseFloat(raw, 64)
	return
}

func tolit(json string) (raw string) {
	for i := 1; i < len(json); i++ {
		if json[i] <= 'a' || json[i] >= 'z' {
			return json[:i]
		}
	}
	return json
}

func tostr(json string) (raw string, str string) {
	// expects that the lead character is a '"'
	for i := 1; i < len(json); i++ {
		if json[i] > '\\' {
			continue
		}
		if json[i] == '"' {
			return json[:i+1], json[1:i]
		}
		if json[i] == '\\' {
			i++
			for ; i < len(json); i++ {
				if json[i] > '\\' {
					continue
				}
				if json[i] == '"' {
					// look for an escaped slash
					if json[i-1] == '\\' {
						n := 0
						for j := i - 2; j > 0; j-- {
							if json[j] != '\\' {
								break
							}
							n++
						}
						if n%2 == 0 {
							continue
						}
					}
					break
				}
			}
			return json[:i+1], unescape(json[1:i])
		}
	}
	return json, json[1:]
}

// Exists returns true if value exists.
//
//  if gjson.Get(json, "name.last").Exists(){
//		println("value exists")
//  }
func (t Result) Exists() bool {
	return t.Type != Null || len(t.Raw) != 0
}

// Value returns one of these types:
//
//	bool, for JSON booleans
//	float64, for JSON numbers
//	Number, for JSON numbers
//	string, for JSON string literals
//	nil, for JSON null
//
func (t Result) Value() interface{} {
	if t.Type == String {
		return t.Str
	}
	switch t.Type {
	default:
		return nil
	case False:
		return false
	case Number:
		return t.Num
	case JSON:
		_, ai, _, oi, vc := t.arrayOrMap(0, true)
		if vc == '{' {
			return oi
		} else if vc == '[' {
			return ai
		}
		return nil
	case True:
		return true
	}

}

type part struct {
	wild bool
	key  string
}

type frame struct {
	key   string
	count int
	stype byte
}

// Get searches json for the specified path.
// A path is in dot syntax, such as "name.last" or "age".
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return back unexpected results.
// When the value is found it's returned immediately.
//
// A path is a series of keys seperated by a dot.
// A key may contain special wildcard characters '*' and '?'.
// To access an array value use the index as the key.
// To get the number of elements in an array or to access a child path, use the '#' character.
// The dot and wildcard character can be escaped with '\'.
//
//  {
//    "name": {"first": "Tom", "last": "Anderson"},
//    "age":37,
//    "children": ["Sara","Alex","Jack"],
//    "friends": [
//      {"first": "James", "last": "Murphy"},
//      {"first": "Roger", "last": "Craig"}
//    ]
//  }
//  "name.last"          >> "Anderson"
//  "age"                >> 37
//  "children.#"         >> 3
//  "children.1"         >> "Alex"
//  "child*.2"           >> "Jack"
//  "c?ildren.0"         >> "Sara"
//  "friends.#.first"    >> [ "James", "Roger" ]
//
func Get(json string, path string) Result {
	var s int                       // starting index variable
	var wild bool                   // wildcard indicator
	var parts = make([]part, 0, 4)  // parsed path parts
	var i int                       // index of current json character
	var depth int                   // the current stack depth
	var f frame                     // the current frame
	var matched bool                // flag used for key/part matching
	var stack = make([]frame, 1, 4) // the frame stack
	var value Result                // the final value, also used for temp store
	var vc byte                     // the current token value chacter type
	var arrch bool
	var alogok bool
	var alogkey string
	var alog []int
	var uc bool

	// parse the path into multiple parts.
	for i := 0; i < len(path); i++ {
		if path[i]&0x60 == 0x60 {
			// alpha lowercase
			continue
		}
		if path[i] >= 'A' && path[i] <= 'Z' {
			continue
		}
		if path[i] == '.' {
			// append a new part
			parts = append(parts, part{wild: wild, key: path[s:i]})
			if wild {
				wild = false // reset the wild flag
			}
			// set the starting index to one past the dot.
			s = i + 1
			continue
		}
		if (path[i] >= '0' && path[i] <= '9') || path[i] == '_' {
			continue
		}
		if path[i] == '*' || path[i] == '?' {
			wild = true
			continue
		}
		if path[i] == '#' {
			arrch = true
			if s == i && i+1 < len(path) && path[i+1] == '.' {
				alogok = true
				alogkey = path[i+2:]
				path = path[:i+1]
			}
			continue
		}
		if path[i] > 0x7f {
			uc = true
			continue
		}
		if path[i] == '\\' {
			// go into escape mode. this is a slower path that
			// strips off the escape character from the part.
			epart := []byte(path[s:i])
			i++
			if i < len(path) {
				epart = append(epart, path[i])
				i++
				for ; i < len(path); i++ {
					if path[i] > 0x7f {
						uc = true
						continue
					}
					if path[i] == '\\' {
						i++
						if i < len(path) {
							epart = append(epart, path[i])
						}
						continue
					} else if path[i] == '.' {
						parts = append(parts, part{
							wild: wild, key: string(epart),
						})
						if wild {
							wild = false
						}
						s = i + 1
						i++
						goto next_part
					} else if path[i] == '*' || path[i] == '?' {
						wild = true
					} else if path[i] == '#' {
						arrch = true
						if s == i && i+1 < len(path) && path[i+1] == '.' {
							alogok = true
							alogkey = path[i+2:]
							path = path[:i+1]
						}
					}
					epart = append(epart, path[i])
				}
			}
			// append the last part
			parts = append(parts, part{wild: wild, key: string(epart)})
			goto end_parts
		next_part:
			continue
		}
	}
	// append the last part
	parts = append(parts, part{wild: wild, key: path[s:]})
end_parts:

	i = 0

	// look for first delimiter. only allow arrays and objects, other
	// json types will fail. it's ok for control characters to passthrough.
	for ; i < len(json); i++ {
		if json[i] == '{' {
			f.stype = '{'
			i++
			stack[0].stype = f.stype
			break
		} else if json[i] == '[' {
			f.stype = '['
			stack[0].stype = f.stype
			i++
			break
		} else if json[i] <= ' ' {
			continue
		} else {
			return Result{}
		}
	}

	// assume that the depth is at least one
	depth = 1

	// read the next key from the json string
read_key:
	if f.stype == '[' {
		// for arrays we use the index of the value as the key.
		// so "0" is the key for the first value, and "10" is the
		// key for the 10th value.
		f.key = strconv.FormatInt(int64(f.count), 10)
		f.count++
		if alogok && depth == len(parts) {
			alog = append(alog, i)
		}
	} else {
		// for objects we must parse the next string. this string will
		// become the key that is compared against the path parts.
		for ; i < len(json); i++ {
			// begin key string reading routine.
			if json[i] == '"' {
				i++
				// set the starting index. the first double-quote has already
				// been read.
				s = i
				// loop through each character in the string looking for the
				// the double-quote termination character. it's possible that
				// the string contains an escape slash character. if so, we
				// must do a nested loop that will look for an isolated
				// double-quote terminator.
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						// a simple string that contains no escape characters.
						// assign this to the current frame key and we are
						// done parsing the key.
						f.key = json[s:i]
						i++
						break
					}
					if json[i] == '\\' {
						// escape character detected. we now look for the
						// the double-quote terminator.
						i++
						for ; i < len(json); i++ {
							if json[i] == '"' {
								// possibly the end of the string, but let's
								// look to see if the previous character was
								// an escape slash. if so then we must keep
								// reading backwards to see if the slash has a
								// prefixed slashed, and so forth.
								if json[i-1] == '\\' {
									n := 0
									for j := i - 2; j > s-1; j-- {
										if json[j] != '\\' {
											break
										}
										n++
									}
									if n%2 == 0 {
										// the double-quote is not a terminator.
										// keep reading the string.
										continue
									}
								}
								// we found the correct double-quote terminator.
								// stop reading the string.
								break
							}
						}
						// the string contains escape sequences so we must
						// unescape and then assign to the current frame key.
						// done parsing the key
						f.key = unescape(json[s:i])
						i++
						break
					}
				}
				break
			}
			// end of string key reading routine
		}
	}

	// we have a brand new (possibly shiny) key.
	// is it the key that we are looking for?
	if parts[depth-1].wild {
		// the path part contains a wildcard character. we must do a wildcard
		// match to determine if it truly matches.
		matched = wildcardMatch(f.key, parts[depth-1].key, uc)
	} else {
		// just a straight up equality check
		matched = parts[depth-1].key == f.key
	}

	// read the value
	for ; i < len(json); i++ {
		// any thing less than  a double-quote is likely whitespace.
		// just burn past these.
		if json[i] < '"' {
			continue
		}
		// anything less that a dash is likely a double-quote. let's
		// assume that it is.
		if json[i] < '-' {
			i++
			vc = '"'
			// defer reading the string value until we know for sure
			// that we want it. if we don't want it, then we will
			// parse it using a quicker method than if we do want it.
			goto proc_val
		}
		// any character less than an open bracket is likely a number.
		if json[i] < '[' {
			// with one exception, the colon character. we do not care
			// about the colon character. just burn past it.
			if json[i] == ':' {
				continue
			}
			vc = '0'
			s = i
			i++
			// look for any character that might terminate a number
			// break on whitespace, comma, ']', and '}'.
			for ; i < len(json); i++ {
				// less than dash might have valid characters
				if json[i] <= '-' {
					if json[i] <= ' ' || json[i] == ',' {
						// break on whitespace and comma
						break
					}
					// could be a '+' or '-'. let's assume so.
					continue
				}
				if json[i] < ']' {
					// probably a valid number
					continue
				}
				if json[i] == 'e' || json[i] == 'E' {
					// allow for exponential numbers
					continue
				}
				// likely a ']' or '}'
				break
			}
			// we have raw number. jump to the process value routine.
			goto proc_val
		}
		// any character less than ']' is likely '['. let's assume
		// it's an open-array character.
		if json[i] < ']' {
			i++
			vc = '['
			// jump to process delimiter routine.
			goto proc_nested
		}
		// any character less than 'u' likely means tha the value is
		// 'true', 'false', or 'null'.
		if json[i] < 'u' {
			vc = json[i] // assign the vc token character to the actual.
			s = i
			i++
			for ; i < len(json); i++ {
				// let's pick up any non-alpha lowercase character as the
				// terminator. it doesn't matter.
				if json[i] < 'a' || json[i] > 'z' {
					break
				}
			}
			// we have raw literal. jump to the process value routine.
			goto proc_val
		}
		// if we reached this far, then the value must be a nested object.
		i++
		vc = '{'
		// jump to process delimiter routine.
		goto proc_nested
	}
	vc = 0
	// ran out of json buffer
	if i >= len(json) {
		return Result{}
	}

	// process nested array or object
proc_nested:
	if (matched && depth == len(parts)) || !matched {
		// begin squash
		// squash the value, ignoring all nested arrays and objects.
		s = i - 1
		// the first '[' or '{' has already been read
		depth := 1
	squash:
		for ; i < len(json); i++ {
			if json[i] >= '"' && json[i] <= '}' {
				switch json[i] {
				case '"':
					i++
					s2 := i
					for ; i < len(json); i++ {
						if json[i] > '\\' {
							continue
						}
						if json[i] == '"' {
							// look for an escaped slash
							if json[i-1] == '\\' {
								n := 0
								for j := i - 2; j > s2-1; j-- {
									if json[j] != '\\' {
										break
									}
									n++
								}
								if n%2 == 0 {
									continue
								}
							}
							break
						}
					}
				case '{', '[':
					depth++
				case '}', ']':
					depth--
					if depth == 0 {
						i++
						break squash
					}
				}
			}
		}
		// end squash
		// the 'i' and 's' values should fall-though to the proc_val function
	}

	// process the value
proc_val:
	if matched {
		// hit, that's good!
		if depth == len(parts) {
			switch vc {
			case '{', '[':
				value.Type = JSON
				value.Raw = json[s:i]
			case 'n':
				value.Type = Null
				value.Raw = json[s:i]
			case 't':
				value.Type = True
				value.Raw = json[s:i]
			case 'f':
				value.Type = False
				value.Raw = json[s:i]
			case '"':
				value.Type = String
				// readstr
				// the val has not been read yet
				// the first double-quote has already been read
				s = i
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						value.Raw = json[s-1 : i+1]
						value.Str = json[s:i]
						break
					}
					if json[i] == '\\' {
						i++
						for ; i < len(json); i++ {
							if json[i] > '\\' {
								continue
							}
							if json[i] == '"' {
								// look for an escaped slash
								if json[i-1] == '\\' {
									n := 0
									for j := i - 2; j > s-1; j-- {
										if json[j] != '\\' {
											break
										}
										n++
									}
									if n%2 == 0 {
										continue
									}
								}
								break
							}
						}
						value.Raw = json[s-1 : i+1]
						value.Str = unescape(json[s:i])
						break
					}
				}
				// end readstr
			case '0':
				value.Type = Number
				value.Raw = json[s:i]
				value.Num, _ = strconv.ParseFloat(value.Raw, 64)
			}
			return value
		} else {
			f = frame{stype: vc}
			stack = append(stack, f)
			depth++
			goto read_key
		}
	}
	if vc == '"' {
		// readstr
		// the val has not been read yet. we can read and throw away.
		// the first double-quote has already been read
		s = i
		for ; i < len(json); i++ {
			if json[i] == '"' {
				// look for an escaped slash
				if json[i-1] == '\\' {
					n := 0
					for j := i - 2; j > s-1; j-- {
						if json[j] != '\\' {
							break
						}
						n++
					}
					if n%2 == 0 {
						continue
					}
				}
				break
			}
		}
		i++
		// end readstr
	}

	// read to the comma or end of object
	for ; i < len(json); i++ {
		switch json[i] {
		case '}', ']':
			if arrch && parts[depth-1].key == "#" {
				if alogok {
					var jsons = make([]byte, 0, 64)
					jsons = append(jsons, '[')
					for j := 0; j < len(alog); j++ {
						res := Get(json[alog[j]:], alogkey)
						if res.Exists() {
							if j > 0 {
								jsons = append(jsons, ',')
							}
							jsons = append(jsons, []byte(res.Raw)...)
						}
					}
					jsons = append(jsons, ']')
					return Result{Type: JSON, Raw: string(jsons)}
				} else {
					return Result{Type: Number, Num: float64(f.count)}
				}
			}
			// step the stack back
			depth--
			if depth == 0 {
				return Result{}
			}
			stack = stack[:len(stack)-1]
			f = stack[len(stack)-1]
		case ',':
			i++
			goto read_key
		}
	}
	return Result{}
}

// unescape unescapes a string
func unescape(json string) string { //, error) {
	var str = make([]byte, 0, len(json))
	for i := 0; i < len(json); i++ {
		switch {
		default:
			str = append(str, json[i])
		case json[i] < ' ':
			return "" //, errors.New("invalid character in string")
		case json[i] == '\\':
			i++
			if i >= len(json) {
				return "" //, errors.New("invalid escape sequence")
			}
			switch json[i] {
			default:
				return "" //, errors.New("invalid escape sequence")
			case '\\':
				str = append(str, '\\')
			case '/':
				str = append(str, '/')
			case 'b':
				str = append(str, '\b')
			case 'f':
				str = append(str, '\f')
			case 'n':
				str = append(str, '\n')
			case 'r':
				str = append(str, '\r')
			case 't':
				str = append(str, '\t')
			case '"':
				str = append(str, '"')
			case 'u':
				if i+5 > len(json) {
					return "" //, errors.New("invalid escape sequence")
				}
				i++
				// extract the codepoint
				var code int
				for j := i; j < i+4; j++ {
					switch {
					default:
						return "" //, errors.New("invalid escape sequence")
					case json[j] >= '0' && json[j] <= '9':
						code += (int(json[j]) - '0') << uint(12-(j-i)*4)
					case json[j] >= 'a' && json[j] <= 'f':
						code += (int(json[j]) - 'a' + 10) << uint(12-(j-i)*4)
					case json[j] >= 'a' && json[j] <= 'f':
						code += (int(json[j]) - 'a' + 10) << uint(12-(j-i)*4)
					}
				}
				str = append(str, []byte(string(code))...)
				i += 3 // only 3 because we will increment on the for-loop
			}
		}
	}
	return string(str) //, nil
}

// Less return true if a token is less than another token.
// The caseSensitive paramater is used when the tokens are Strings.
// The order when comparing two different type is:
//
//  Null < False < Number < String < True < JSON
//
func (t Result) Less(token Result, caseSensitive bool) bool {
	if t.Type < token.Type {
		return true
	}
	if t.Type > token.Type {
		return false
	}
	if t.Type == String {
		if caseSensitive {
			return t.Str < token.Str
		}
		return stringLessInsensitive(t.Str, token.Str)
	}
	if t.Type == Number {
		return t.Num < token.Num
	}
	return t.Raw < token.Raw
}

func stringLessInsensitive(a, b string) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] >= 'A' && a[i] <= 'Z' {
			if b[i] >= 'A' && b[i] <= 'Z' {
				// both are uppercase, do nothing
				if a[i] < b[i] {
					return true
				} else if a[i] > b[i] {
					return false
				}
			} else {
				// a is uppercase, convert a to lowercase
				if a[i]+32 < b[i] {
					return true
				} else if a[i]+32 > b[i] {
					return false
				}
			}
		} else if b[i] >= 'A' && b[i] <= 'Z' {
			// b is uppercase, convert b to lowercase
			if a[i] < b[i]+32 {
				return true
			} else if a[i] > b[i]+32 {
				return false
			}
		} else {
			// neither are uppercase
			if a[i] < b[i] {
				return true
			} else if a[i] > b[i] {
				return false
			}
		}
	}
	return len(a) < len(b)
}

// wilcardMatch returns true if str matches pattern. This is a very
// simple wildcard match where '*' matches on any number characters
// and '?' matches on any one character.
func wildcardMatch(str, pattern string, uc bool) bool {
	if pattern == "*" {
		return true
	}
	if !uc {
		return deepMatch(str, pattern)
	}
	rstr := make([]rune, 0, len(str))
	rpattern := make([]rune, 0, len(pattern))
	for _, r := range str {
		rstr = append(rstr, r)
	}
	for _, r := range pattern {
		rpattern = append(rpattern, r)
	}
	return deepMatchRune(rstr, rpattern)
}
func deepMatch(str, pattern string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 {
				return false
			}
		case '*':
			return deepMatch(str, pattern[1:]) ||
				(len(str) > 0 && deepMatch(str[1:], pattern))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
func deepMatchRune(str, pattern []rune) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 {
				return false
			}
		case '*':
			return deepMatchRune(str, pattern[1:]) ||
				(len(str) > 0 && deepMatchRune(str[1:], pattern))
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}
