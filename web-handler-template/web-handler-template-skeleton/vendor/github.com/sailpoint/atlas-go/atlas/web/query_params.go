// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
)

// filters, sorters, offset, limit and count are V3 query param keys.
const filters = "filters"
const sorters = "sorters"
const offset = "offset"
const limit = "limit"
const count = "count"

// QueryOptions is a struct containing V3 query params.
type QueryOptions struct {
	Offset  int
	Limit   int
	Filters Filter
	Sorters []ListSorter
}

// ListSorter is a struct sorting property and its order.
type ListSorter struct {
	Property    string
	IsAscending bool
}

// OrExpression, etc, are string literals representing query operations.
const (
	OrExpression  = "or"
	AndExpression = "and"
	NotExpression = "not"
	EqOperator    = "eq"
	GtOperator    = "gt"
	LtOperator    = "lt"
	NeOperator    = "ne"
	GeOperator    = "ge"
	LeOperator    = "le"
	CoOperator    = "co"
	SwOperator    = "sw"
	PrOperator    = "pr"
	InOperator    = "in"
	CaOperator    = "ca"
)

// LogicalOperation is an enumeration for V3 query logical operations.
type LogicalOperation string

// Eq and other constants discribes the type of the logical operation.
const (
	Eq          LogicalOperation = "EQ"
	Ne          LogicalOperation = "NE"
	Gt          LogicalOperation = "GT"
	Lt          LogicalOperation = "LT"
	Ge          LogicalOperation = "GE"
	Le          LogicalOperation = "LE"
	Like        LogicalOperation = "LIKE"
	NotNull     LogicalOperation = "NOTNULL"
	In          LogicalOperation = "IN"
	ContainsAll LogicalOperation = "CONTAINS_ALL"
)

// operatorMap is a mapping between filter operators and logical operations
var operatorMap = map[string]LogicalOperation{
	EqOperator: Eq,
	NeOperator: Ne,
	GeOperator: Ge,
	LeOperator: Le,
	GtOperator: Gt,
	LtOperator: Lt,
	CaOperator: ContainsAll,
	InOperator: In,
	CoOperator: Like,
	SwOperator: Like,
}

// MatchMode is an enumeration for V3 query matching mode.
type MatchMode string

// Anywhere and start are query mathing modes.
const (
	Anywhere MatchMode = "ANYWHERE"
	Start    MatchMode = "START"
)

// Filter is an interface for V3 filter.
type Filter interface{}

// FilterBuilder is an interface for V3 filter builder.
type FilterBuilder interface {
	And(filters []Filter) (Filter, error)
	Or(filters []Filter) (Filter, error)
	Not(filter Filter) (Filter, error)
	NewFilter(op LogicalOperation, property string, valueObject interface{}) (Filter, error)
	NewFilterWithMatchMode(op LogicalOperation, property string, valueObject interface{}, mode MatchMode) (Filter, error)
	IgnoreCase(filter Filter) (Filter, error)
	NewFilterWithValueList(op LogicalOperation, property string, valueList []interface{}) (Filter, error)
}

// GetQueryOptions returns a struct containing common V3 query params.
func GetQueryOptions(r *http.Request, sortableFields mapset.Set, fb FilterBuilder, queryableFields mapset.Set) (*QueryOptions, error) {
	q := &QueryOptions{}

	// Get offset
	if o := r.URL.Query().Get(offset); o != "" {
		offset, err := strconv.Atoi(o)
		if err != nil || offset < 0 {
			return nil, fmt.Errorf("invalid offset value: %s", o)
		}
		q.Offset = offset
	}

	// Get limit
	if l := r.URL.Query().Get(limit); l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil || limit <= 0 || limit > 250 {
			return nil, fmt.Errorf("invalid limit value: %s", l)
		}
		q.Limit = limit
	} else {
		q.Limit = 250
	}

	// Get sorters
	s, err := GetSorters(r.URL.Query().Get(sorters), sortableFields)
	if err != nil {
		return nil, err
	}
	q.Sorters = s

	// Get filters
	f, err := GetFilter(r.URL.Query().Get(filters), fb, queryableFields)
	if err != nil {
		return nil, err
	}
	q.Filters = f

	return q, nil
}

// GetSorters reads sorter string into a list of sorters.
func GetSorters(sorters string, sortableFields mapset.Set) ([]ListSorter, error) {
	l := []ListSorter{}

	if sorters == "" {
		return l, nil
	}

	sorterComponents := strings.Split(sorters, ",")
	for _, sorter := range sorterComponents {

		ascending := true
		if strings.HasPrefix(sorter, "-") {
			ascending = false
			sorter = strings.TrimPrefix(sorter, "-")
		}

		if !sortableFields.Contains(sorter) {
			return nil, fmt.Errorf("invalid sort propertie: %s", sorter)
		}

		l = append(l, ListSorter{sorter, ascending})
	}
	return l, nil
}

// GetFilter parses filters in string expression into a Filter object.
func GetFilter(filters string, fb FilterBuilder, queryableFields mapset.Set) (Filter, error) {
	filters = strings.TrimSpace(filters)
	if filters == "" || fb == nil {
		return nil, nil
	}

	return compileConditionalOrFilter(filters, fb, queryableFields)
}

// compileNotFilter tries to compile filter expressions separated by "or".
func compileConditionalOrFilter(filters string, fb FilterBuilder, queryableFields mapset.Set) (Filter, error) {
	if components, found := splitFilters(filters, OrExpression); found {
		filterList := []Filter{}
		for _, c := range components {
			embededFilter, err := compileConditionalAndFilter(c, fb, queryableFields)
			if err != nil {
				return nil, err
			}
			filterList = append(filterList, embededFilter)
		}
		return fb.Or(filterList)
	}

	return compileConditionalAndFilter(filters, fb, queryableFields)
}

// compileNotFilter tries to compile filter expressions separated by "and".
func compileConditionalAndFilter(filters string, fb FilterBuilder, queryableFields mapset.Set) (Filter, error) {
	if components, found := splitFilters(filters, AndExpression); found {
		filterList := []Filter{}
		for _, c := range components {
			embededFilter, err := compileNotFilter(c, fb, queryableFields)
			if err != nil {
				return nil, err
			}
			filterList = append(filterList, embededFilter)
		}
		return fb.And(filterList)
	}

	return compileNotFilter(filters, fb, queryableFields)
}

// compileNotFilter tries to compile filter expression to a "not" filter.
func compileNotFilter(filters string, fb FilterBuilder, queryableFields mapset.Set) (Filter, error) {
	if _, exp, found := parseProperty(filters, NotExpression); found {
		f, err := compilePrimary(exp, fb, queryableFields)
		if err != nil {
			return nil, err
		}
		return fb.Not(f)
	}

	return compilePrimary(filters, fb, queryableFields)
}

// compilePrimary compiles filter expression to a Filter by parsing the string expression.
func compilePrimary(filters string, fb FilterBuilder, queryableFields mapset.Set) (Filter, error) {

	for operator, operation := range operatorMap {
		if prop, value, found := parseProperty(filters, operator); found {
			// Make sure the filter is queryable
			if !queryableFields.Contains(prop) {
				return nil, fmt.Errorf("invalid filter propertie: %s", prop)
			}

			// Parse filter with lists
			if operator == CaOperator || operator == InOperator {
				ll, err := parseLiteralList(value)
				if err != nil {
					return nil, err
				}
				return fb.NewFilterWithValueList(operation, prop, ll)
			}

			// Parse filter value
			l, err := parseLiteral(value)
			if err != nil {
				return nil, err
			}

			// Return filter after building it
			if operator == CoOperator {
				return fb.NewFilterWithMatchMode(operation, prop, l, Anywhere)
			} else if operator == SwOperator {
				return fb.NewFilterWithMatchMode(operation, prop, l, Start)
			} else {
				return fb.NewFilter(operation, prop, l)
			}
		}
	}

	return nil, fmt.Errorf("failed to parse: %s", filters)
}

// splitFilters splits filter expressions by splitter.
func splitFilters(filters string, splitter string) ([]string, bool) {
	components := strings.Split(fmt.Sprintf(" %s ", filters), fmt.Sprintf(" %s ", splitter))
	return components, len(components) > 1
}

// parseProperty parses filter expression and returns property and value if found by splitting the expression.
func parseProperty(expression string, splitter string) (string, string, bool) {
	components := strings.SplitN(fmt.Sprintf(" %s ", expression), fmt.Sprintf(" %s ", splitter), 2)
	if len(components) == 2 {
		return strings.TrimSpace(components[0]), strings.TrimSpace(components[1]), true
	}
	return "", "", false
}

// parseLiteral parses string literal to different data types in the order of string, date, float, integer, boolean, null and "me".
func parseLiteral(literal string) (interface{}, error) {
	literal = strings.TrimSpace(literal)

	if strings.HasPrefix(literal, "\"") && strings.HasSuffix(literal, "\"") {
		s := strings.TrimSuffix(strings.TrimPrefix(literal, "\""), "\"")
		s = strings.ReplaceAll(s, "\\b", "\b")
		s = strings.ReplaceAll(s, "\\t", "\t")
		s = strings.ReplaceAll(s, "\\n", "\n")
		s = strings.ReplaceAll(s, "\\f", "\f")
		s = strings.ReplaceAll(s, "\\r", "\r")
		return s, nil
	} else if t, err := time.Parse(time.RFC3339Nano, literal); err == nil {
		return t, nil
	} else if strings.Count(literal, " ") == 0 && strings.Count(literal, ".") == 1 {
		return strconv.ParseFloat(literal, 64)
	} else if i, err := strconv.Atoi(literal); err == nil {
		return i, nil
	} else if b, err := strconv.ParseBool(literal); err == nil {
		return b, nil
	} else if literal == "me" {
		return literal, nil
	} else if literal == "null" {
		return nil, nil
	}

	return nil, fmt.Errorf("cannot parse literal: %s", literal)
}

// parseLiteralList parses a string formatted literal list to a list of objects of corresponding data types.
func parseLiteralList(literal string) ([]interface{}, error) {
	literal = strings.TrimSpace(literal)
	if strings.HasPrefix(literal, "(") && strings.HasSuffix(literal, ")") {
		literal = strings.TrimSuffix(strings.TrimPrefix(literal, "("), ")")
	}

	components := strings.Split(literal, ",")
	var l []interface{}
	for _, c := range components {
		pl, err := parseLiteral(c)
		if err != nil {
			return nil, err
		}
		l = append(l, pl)
	}
	return l, nil
}
