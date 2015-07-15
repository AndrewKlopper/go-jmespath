package jmespath

import (
	"errors"

	"github.com/jmespath/jmespath.go/jmespath/jputil"
)

/* This is a tree based interpreter.  It walks the AST and directly
   interprets the AST to search through a JSON document.
*/

// Execute takes an ASTNode and input data and interprets the AST directly.
// It will produce the result of applying the JMESPath expression associated
// with the ASTNode to the input data "value".
type TreeInterpreter struct {
}

func NewInterpreter() *TreeInterpreter {
	interpreter := TreeInterpreter{}
	return &interpreter
}

func (intr *TreeInterpreter) Execute(node ASTNode, value interface{}) (interface{}, error) {
	switch node.nodeType {
	case ASTComparator:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, err
		}
		right, err := intr.Execute(node.children[1], value)
		if err != nil {
			return nil, err
		}
		switch node.value {
		case tEQ:
			return jputil.ObjsEqual(left, right), nil
		case tNE:
			return !jputil.ObjsEqual(left, right), nil
		}
		leftNum, ok := left.(float64)
		if !ok {
			return nil, nil
		}
		rightNum, ok := right.(float64)
		if !ok {
			return nil, nil
		}
		switch node.value {
		case tGT:
			return leftNum > rightNum, nil
		case tGTE:
			return leftNum >= rightNum, nil
		case tLT:
			return leftNum < rightNum, nil
		case tLTE:
			return leftNum <= rightNum, nil
		}
	case ASTExpRef:
		return nil, errors.New("NOT IMPLEMENTED ASTExpRef")
	case ASTFunctionExpression:
		return nil, errors.New("NOT IMPLEMENTED ASTFunctionExpression")
	case ASTField:
		if m, ok := value.(map[string]interface{}); ok {
			key := node.value.(string)
			return m[key], nil
		}
		return nil, nil
	case ASTFilterProjection:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, nil
		}
		sliceType, ok := left.([]interface{})
		if !ok {
			return nil, nil
		}
		compareNode := node.children[2]
		collected := make([]interface{}, 0)
		for _, element := range sliceType {
			result, err := intr.Execute(compareNode, element)
			if err != nil {
				return nil, err
			}
			if !jputil.IsFalse(result) {
				current, err := intr.Execute(node.children[1], element)
				if err != nil {
					return nil, err
				}
				if current != nil {
					collected = append(collected, current)
				}
			}
		}
		return collected, nil
	case ASTFlatten:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, nil
		}
		sliceType, ok := left.([]interface{})
		if !ok {
			// Can't flatten a non slice object.
			return nil, nil
		}
		flattened := make([]interface{}, 0)
		for _, element := range sliceType {
			if elementSlice, ok := element.([]interface{}); ok {
				flattened = append(flattened, elementSlice...)
			} else {
				flattened = append(flattened, element)
			}
		}
		return flattened, nil
	case ASTIdentity, ASTCurrentNode:
		return value, nil
	case ASTIndex:
		if sliceType, ok := value.([]interface{}); ok {
			index := node.value.(int)
			if index < 0 {
				index += len(sliceType)
			}
			if index < len(sliceType) && index >= 0 {
				return sliceType[index], nil
			}
		}
		return nil, nil
	case ASTKeyValPair:
		return intr.Execute(node.children[0], value)
	case ASTLiteral:
		return node.value, nil
	case ASTMultiSelectHash:
		if value == nil {
			return nil, nil
		}
		collected := make(map[string]interface{})
		for _, child := range node.children {
			current, err := intr.Execute(child, value)
			if err != nil {
				return nil, err
			}
			key := child.value.(string)
			collected[key] = current
		}
		return collected, nil
	case ASTMultiSelectList:
		if value == nil {
			return nil, nil
		}
		collected := make([]interface{}, 0)
		for _, child := range node.children {
			current, err := intr.Execute(child, value)
			if err != nil {
				return nil, err
			}
			collected = append(collected, current)
		}
		return collected, nil
	case ASTOrExpression:
		matched, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, err
		}
		if jputil.IsFalse(matched) {
			matched, err = intr.Execute(node.children[1], value)
			if err != nil {
				return nil, err
			}
		}
		return matched, nil
	case ASTPipe:
		result := value
		var err error
		for _, child := range node.children {
			result, err = intr.Execute(child, result)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	case ASTProjection:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, err
		}
		if sliceType, ok := left.([]interface{}); ok {
			collected := make([]interface{}, 0)
			var current interface{}
			var err error
			for _, element := range sliceType {
				current, err = intr.Execute(node.children[1], element)
				if err != nil {
					return nil, err
				}
				if current != nil {
					collected = append(collected, current)
				}
			}
			return collected, nil
		}
		return nil, nil
	case ASTSubexpression, ASTIndexExpression:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, err
		}
		return intr.Execute(node.children[1], left)
	case ASTSlice:
		sliceType, ok := value.([]interface{})
		if !ok {
			return nil, nil
		}
		parts := node.value.([]*int)
		sliceParams := make([]jputil.SliceParam, 3)
		for i, part := range parts {
			if part != nil {
				sliceParams[i].Specified = true
				sliceParams[i].N = *part
			}
		}
		return jputil.Slice(sliceType, sliceParams)
	case ASTValueProjection:
		left, err := intr.Execute(node.children[0], value)
		if err != nil {
			return nil, nil
		}
		mapType, ok := left.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		values := make([]interface{}, len(mapType))
		for _, value := range mapType {
			values = append(values, value)
		}
		collected := make([]interface{}, 0)
		for _, element := range values {
			current, err := intr.Execute(node.children[1], element)
			if err != nil {
				return nil, err
			}
			if current != nil {
				collected = append(collected, current)
			}
		}
		return collected, nil
	}
	return nil, errors.New("FAILED")
}