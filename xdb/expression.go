package xdb

import (
	"regexp"
	//	"github.com/emirpasic/gods/v2/maps/treemap"
)

// 新建一个模板匹配器
var NewTemplateMatcher func(matchers ...ExpressionMatcher) TemplateMatcher

func init() {
	NewTemplateMatcher = NewDefaultTemplateMatcher
}

// 属性表达式匹配器
type ExpressionMatcher interface {
	Name() string
	Pattern() string
	LoadSymbol(symbol string) (Symbol, bool)
	MatchString(string) (ExpressionValuer, bool)
}

type ExpressionMatcherMap interface {
	Regist(...ExpressionMatcher)
	Load(name string) (ExpressionMatcher, bool)
	Each(call func(name string, matcher ExpressionMatcher) bool)
	BuildFullRegexp() *regexp.Regexp
}

// xdb表达式
type ExpressionValuer interface {
	GetPropName() string
	GetFullfield() string
	GetOper() string
	GetSymbol() string
	Build(input DBParam, argName string) (string, MissError)
}

// 表达式回调
type ExpressionBuildCallback func(item *ExpressionItem, param DBParam, argName string) (expression string, err MissError)

type ExpressionItem struct {
	FullField               string
	PropName                string
	Oper                    string
	Symbol                  string
	ExpressionBuildCallback ExpressionBuildCallback
}

func (m *ExpressionItem) GetSymbol() string {
	return m.Symbol
}

func (m *ExpressionItem) GetPropName() string {
	return m.PropName
}

func (m *ExpressionItem) GetFullfield() string {
	return m.FullField
}

func (m *ExpressionItem) GetOper() string {
	return m.Oper
}

func (m *ExpressionItem) Build(param DBParam, argName string) (expression string, err MissError) {
	if m.ExpressionBuildCallback == nil {
		return
	}
	return m.ExpressionBuildCallback(m, param, argName)
}