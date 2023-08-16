// Package build contains functionality to generate a cft.Template
// from specification data in cft.spec
package build

import (
	"fmt"
	"strings"

	"github.com/elishacatherasoo/rain/cft"
	"github.com/elishacatherasoo/rain/cft/spec"
)

const (
	policyDocument           = "PolicyDocument"
	assumeRolePolicyDocument = "AssumeRolePolicyDocument"
	optionalTag              = "Optional"
	changeMeTag              = "CHANGEME"
)

type tracker struct {
	done map[string]bool
	last []string
}

func newTracker() *tracker {
	return &tracker{done: make(map[string]bool), last: make([]string, 0)}
}

func (t *tracker) push(s string) bool {
	if _, ok := t.done[s]; ok {
		return false
	}

	t.last = append(t.last, s)
	t.done[s] = true
	return true
}

func (t *tracker) pop() {
	delete(t.done, t.last[len(t.last)-1])
	t.last = t.last[:len(t.last)-1]
}

// builder generates a template from its Spec
type builder struct {
	Spec                      spec.Spec
	IncludeOptionalProperties bool
	BuildIamPolicies          bool
	tracker                   *tracker
}

var iam iamBuilder

var emptyProp = spec.Property{}

func init() {
	iam = newIamBuilder()
}

func (b builder) newResource(resourceType string) (map[string]interface{}, []*cft.Comment) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building resource type '%s': %v", resourceType, r))
		}
	}()

	rSpec, ok := b.Spec.ResourceTypes[resourceType]
	if !ok {
		panic(fmt.Errorf("no such resource type '%s'", resourceType))
	}

	// fmt.Printf("%#v\n", rSpec)

	b.tracker = newTracker()

	// Generate properties
	properties := make(map[string]interface{})
	comments := make([]*cft.Comment, 0)
	for name, pSpec := range rSpec.Properties {

		// fmt.Printf("Generating prop %v, pSpec: %#v \n", name, pSpec)

		if b.IncludeOptionalProperties || pSpec.Required {
			var p interface{}
			var cs []*cft.Comment

			if b.BuildIamPolicies && (name == policyDocument || name == assumeRolePolicyDocument) {
				p, cs = iam.Policy()
			} else {
				p, cs = b.newProperty(resourceType, name, pSpec)
			}

			properties[name] = p
			for _, c := range cs {
				c.Path = append([]interface{}{"Properties", name}, c.Path...)
			}
			comments = append(comments, cs...)
		}
	}

	resource := map[string]interface{}{
		"Type":       resourceType,
		"Properties": properties,
	}

	if len(properties) == 0 {
		delete(resource, "Properties")
	}

	return resource, comments
}

func (b builder) newProperty(resourceType, propertyName string, pSpec *spec.Property) (interface{}, []*cft.Comment) {
	if !b.tracker.push(fmt.Sprintf("%s/%s", resourceType, propertyName)) {
		return nil, nil
	}

	defer b.tracker.pop()

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building property %s.%s: %v", resourceType, propertyName, r))
		}
	}()

	// Correct badly-formed entries
	if pSpec.PrimitiveType == spec.TypeMap {
		pSpec.PrimitiveType = spec.TypeEmpty
		pSpec.Type = spec.TypeMap
	}

	// Attempt to fix failures do to lack of types in some properties
	// TODO - This is a hack, figure out why they are missing
	/*
		Example from cfn.go

		"DialerConfig": &Property{
						Documentation: "http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-connectcampaigns-campaign.html#cfn-connectcampaigns-campaign-dialerconfig",
						Required:      true,
						UpdateType:    "Mutable",
					},

	*/
	if pSpec.PrimitiveType == spec.TypeEmpty && pSpec.Type == spec.TypeEmpty {
		pSpec.PrimitiveType = "String"
	}

	// Primitive types
	if pSpec.PrimitiveType != spec.TypeEmpty {
		if pSpec.Required {
			return b.newPrimitive(pSpec.PrimitiveType), make([]*cft.Comment, 0)
		}

		return b.newPrimitive(pSpec.PrimitiveType), []*cft.Comment{{
			Path:  []interface{}{},
			Value: optionalTag,
		}}
	}

	if pSpec.Type == spec.TypeList || pSpec.Type == spec.TypeMap {
		var value interface{}
		var subComments []*cft.Comment

		// Calculate a single item example
		if pSpec.PrimitiveItemType != spec.TypeEmpty {
			value = b.newPrimitive(pSpec.PrimitiveItemType)
		} else if pSpec.ItemType != spec.TypeEmpty {
			value, subComments = b.newPropertyType(resourceType, pSpec.ItemType)
		} else {
			value = changeMeTag
		}

		if pSpec.Type == spec.TypeList {
			// Returning a list - append a zero to comment paths
			for _, c := range subComments {
				c.Path = append([]interface{}{0}, c.Path...)
			}

			return []interface{}{value}, subComments
		}

		// Returning a map - append changemetag to comment paths
		for _, c := range subComments {
			c.Path = append([]interface{}{changeMeTag}, c.Path...)
		}

		return map[string]interface{}{changeMeTag: value}, subComments
	}

	// Fall through to property types
	b.tracker.pop()
	defer b.tracker.push(fmt.Sprintf("%s/%s", resourceType, propertyName))
	output, comments := b.newPropertyType(resourceType, pSpec.Type)

	if !pSpec.Required {
		comments = append(comments, &cft.Comment{
			Path:  []interface{}{},
			Value: optionalTag,
		})
	}

	return output, comments
}

func (b builder) newPrimitive(primitiveType string) interface{} {
	switch primitiveType {
	case "String":
		return changeMeTag
	case "Integer":
		return 0
	case "Double":
		return 0.0
	case "Long":
		return 0.0
	case "Boolean":
		return false
	case "Timestamp":
		return "1970-01-01 00:00:00"
	case "Json":
		return "{\"JSON\": \"CHANGEME\"}"
	default:
		panic(fmt.Errorf("unimplemented primitive type '%s'", primitiveType))
	}
}

func (b builder) newPropertyType(resourceType, propertyType string) (interface{}, []*cft.Comment) {
	if !b.tracker.push(fmt.Sprintf("%s/%s", resourceType, propertyType)) {
		return nil, nil
	}

	defer b.tracker.pop()

	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("error building property type '%s.%s': %v", resourceType, propertyType, r))
		}
	}()

	var ptSpec *spec.PropertyType
	var ok bool

	// If we've used a property from another resource type
	// switch to that resource type for now
	parts := strings.Split(propertyType, ".")
	if len(parts) == 2 {
		resourceType = parts[0]
	}

	ptSpec, ok = b.Spec.PropertyTypes[propertyType]
	if !ok {
		ptSpec, ok = b.Spec.PropertyTypes[resourceType+"."+propertyType]
	}
	if !ok {
		// TODO - Why is this failing during tests?
		// fmt.Println("About to fail? on", propertyType) // propertyType is ""

		panic(fmt.Errorf("unimplemented property type '%s.%s'", resourceType, propertyType))
	}

	// Deal with the case that a property type is directly a plain property
	// for example AWS::Glue::SecurityConfiguration.S3Encryptions
	if ptSpec.Property != emptyProp {
		return b.newProperty(resourceType, propertyType, &ptSpec.Property)
	}

	comments := make([]*cft.Comment, 0)

	// Generate properties
	properties := make(map[string]interface{})
	for name, pSpec := range ptSpec.Properties {
		if b.IncludeOptionalProperties || pSpec.Required {
			if !pSpec.Required {
				comments = append(comments, &cft.Comment{
					Path:  []interface{}{name},
					Value: optionalTag,
				})
			}

			var p interface{}
			var cs []*cft.Comment

			if b.BuildIamPolicies && (name == policyDocument || name == assumeRolePolicyDocument) {
				p, cs = iam.Policy()
			} else if pSpec.Type == propertyType || pSpec.ItemType == propertyType {
				p = make(map[string]interface{})
				cs = make([]*cft.Comment, 0)
			} else {
				p, cs = b.newProperty(resourceType, name, pSpec)
			}

			properties[name] = p
			for _, c := range cs {
				c.Path = append([]interface{}{name}, c.Path...)
			}
			comments = append(comments, cs...)
		}
	}

	return properties, comments
}
