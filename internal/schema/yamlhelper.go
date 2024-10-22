package schema

import (
	"fmt"
	"io"
	"reflect"

	yamlv3 "gopkg.in/yaml.v3"
	// yamlv2 "gopkg.in/yaml.v2"
)

// ToYAMLNode converts any Go struct/map/slice/array/primitive into a yamlv3.Node.
//
// This is required to work with the yamlV3 library - it currently does not provide a higher level abstraction
func ToYAMLNode(v interface{}) (*yamlv3.Node, error) {
	node := &yamlv3.Node{}
	err := encodeToYAMLNode(reflect.ValueOf(v), node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// encodeToYAMLNode is a helper function that recursively converts Go values to yamlv3.Node.
func encodeToYAMLNode(value reflect.Value, node *yamlv3.Node) error {
	switch value.Kind() {
	case reflect.Struct:
		node.Kind = yamlv3.MappingNode
		node.Content = []*yamlv3.Node{}
		t := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			fieldType := t.Field(i)
			// YAML tags, if any
			yamlKey := fieldType.Tag.Get("yaml")
			if yamlKey == "" {
				yamlKey = fieldType.Name
			}

			// Add the field name as a scalar node
			keyNode := &yamlv3.Node{Kind: yamlv3.ScalarNode, Value: yamlKey}
			valueNode := &yamlv3.Node{}

			// Recursively encode the field value
			if err := encodeToYAMLNode(field, valueNode); err != nil {
				return err
			}

			// Append the key-value pair to the content of the mapping node
			node.Content = append(node.Content, keyNode, valueNode)
		}

	case reflect.Map:
		node.Kind = yamlv3.MappingNode
		node.Content = []*yamlv3.Node{}
		for _, key := range value.MapKeys() {
			keyNode := &yamlv3.Node{Kind: yamlv3.ScalarNode, Value: fmt.Sprintf("%v", key.Interface())}
			valueNode := &yamlv3.Node{}
			if err := encodeToYAMLNode(value.MapIndex(key), valueNode); err != nil {
				return err
			}
			node.Content = append(node.Content, keyNode, valueNode)
		}

	case reflect.Slice, reflect.Array:
		node.Kind = yamlv3.SequenceNode
		node.Content = []*yamlv3.Node{}
		for i := 0; i < value.Len(); i++ {
			elemNode := &yamlv3.Node{}
			if err := encodeToYAMLNode(value.Index(i), elemNode); err != nil {
				return err
			}
			node.Content = append(node.Content, elemNode)
		}

	case reflect.String:
		node.Kind = yamlv3.ScalarNode
		node.Value = value.String()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		node.Kind = yamlv3.ScalarNode
		node.Value = fmt.Sprintf("%d", value.Int())

	case reflect.Float32, reflect.Float64:
		node.Kind = yamlv3.ScalarNode
		node.Value = fmt.Sprintf("%f", value.Float())

	case reflect.Bool:
		node.Kind = yamlv3.ScalarNode
		node.Value = fmt.Sprintf("%t", value.Bool())
	case reflect.Ptr:
		if err := encodeToYAMLNode(reflect.ValueOf(value), &yamlv3.Node{}); err != nil {
			return err
		}
		// node.Kind = yamlv3.ScalarNode
		// node.Value = fmt.Sprintf("%+v", value)
	default:
		return fmt.Errorf("unsupported type: %s", value.Kind().String())
	}

	return nil
}

func WriteOut(w io.Writer, v any) error {
	enc := yamlv3.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(v)
}
