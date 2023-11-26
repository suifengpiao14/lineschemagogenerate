package lineschemagogenerate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/suifengpiao14/funcs"
	"github.com/suifengpiao14/lineschema"
)

func NewSturct(l lineschema.Lineschema) (structs Structs) {
	arraySuffix := "[]"
	structs = make(Structs, 0)
	id := string(l.Meta.ID)
	rootStructName := funcs.ToCamel(id)
	rootStruct := &Struct{
		IsRoot:     true,
		Name:       rootStructName,
		Attrs:      make([]*StructAttr, 0),
		Lineschema: l.String(),
	}
	structs.AddIngore(rootStruct)
	for _, item := range l.Items {
		if item.Fullname == "" {
			continue
		}
		withRootFullname := strings.Trim(fmt.Sprintf("%s.%s", id, item.Fullname), ".")
		nameArr := strings.Split(withRootFullname, ".")
		nameCount := len(nameArr)
		for i := 1; i < nameCount; i++ { //i从1开始,0 为root,已处理
			parentStructName := funcs.ToCamel(strings.Join(nameArr[:i], "_"))
			parentStruct, _ := structs.Get(parentStructName) // 一定存在
			if strings.HasPrefix(parentStruct.Type, "[]") {
				parentStruct, _ = structs.Get(complex2singularName(parentStructName)) //取单数, 一定存在
			}
			baseName := nameArr[i]
			realBaseName := strings.TrimSuffix(baseName, arraySuffix)
			isArray := baseName != realBaseName
			attrName := funcs.ToCamel(realBaseName)
			if i < nameCount-1 { // 非最后一个,即为上级的attr,又为下级的struct
				subStructName := funcs.ToCamel(strings.Join(nameArr[:i+1], "_"))
				attrType := subStructName
				if isArray {
					singularName := complex2singularName(attrType)
					complexStruct := &Struct{
						IsRoot: false,
						Name:   attrType,
						Type:   fmt.Sprintf("[]%s", singularName),
					}
					structs.AddIngore(complexStruct)
					singularStruct := &Struct{
						IsRoot: false,
						Name:   singularName,
					}
					structs.AddIngore(singularStruct)
				}
				attr := StructAttr{
					Name: attrName,
					Type: attrType,
					Tag:  fmt.Sprintf(`json:"%s"`, funcs.ToLowerCamel(attrName)),
					//Comment: comment,// 符合类型comment 无意义，不增加
				}
				parentStruct.AddAttrReplace(attr)
				subStruct := &Struct{
					IsRoot: false,
					Name:   subStructName,
				}
				structs.AddIngore(subStruct)
				continue
			}
			format := item.Format

			switch format { // 格式化format
			case "number":
				format = "int"
			case "float":
				format = "float64"
			}

			// 最后一个
			comment := item.Comments
			if comment == "" {
				comment = item.Description
			}
			typ := item.Type
			// 根据格式，修改类型
			switch format {
			case "int":
				typ = "int"
			case "bool", "boolean":
				typ = "bool"
			}
			tag := fmt.Sprintf(`json:"%s"`, funcs.ToLowerCamel(attrName))
			if !item.Required { //当作入参时,非必填字断,使用引用
				typ = fmt.Sprintf("*%s", typ)
			}
			isArray = isArray || strings.ToLower(item.Type) == "array" // 最后一个接受当前的type字段值
			if isArray {
				if typ == "array" {
					typ = "interface{}"
					if format != "" {
						typ = format
					}
				}
				typ = fmt.Sprintf("[]%s", typ)
			}

			newAttr := &StructAttr{
				Name:    funcs.ToCamel(attrName),
				Type:    typ,
				Tag:     tag,
				Comment: comment,
			}
			attr, ok := parentStruct.GetAttr(attrName)
			if ok { //已经存在,修正类型和备注
				typ := newAttr.Type
				if strings.HasPrefix(attr.Type, "[]") && !strings.HasPrefix(typ, "[]") {
					typ = fmt.Sprintf("[]%s", typ)
				}
				attr.Type = typ
				if newAttr.Comment != "" {
					attr.Comment = newAttr.Comment
				}
				continue
			}
			// 不存在,新增
			parentStruct.AddAttrIgnore(*newAttr)
		}
	}

	return structs
}

// jsonschemaline 生成go 结构体工具
type Struct struct {
	IsRoot     bool
	Name       string
	Lineschema string
	Attrs      []*StructAttr
	Type       string
}

// AddAttrIgnore 已经存在则跳过
func (s *Struct) AddAttrIgnore(attrs ...StructAttr) {
	if len(s.Attrs) == 0 {
		s.Attrs = make([]*StructAttr, 0)
	}
	for _, attr := range attrs {
		if _, exists := s.GetAttr(attr.Name); exists {
			continue
		}
		s.Attrs = append(s.Attrs, &attr)
	}
}

// AddAttrReplace 增加或者替换
func (s *Struct) AddAttrReplace(attrs ...StructAttr) {
	if len(s.Attrs) == 0 {
		s.Attrs = make([]*StructAttr, 0)
	}
	for _, attr := range attrs {
		if old, exists := s.GetAttr(attr.Name); exists {
			*old = attr
			continue
		}
		s.Attrs = append(s.Attrs, &attr)
	}
}
func (s *Struct) GetAttr(attrName string) (structAttr *StructAttr, exists bool) {
	for _, attr := range s.Attrs {
		if attr.Name == attrName {
			return attr, true
		}
	}
	return nil, false
}

type StructAttr struct {
	Name    string
	Type    string
	Tag     string
	Comment string
}

type Structs []*Struct

func (s *Structs) Json() (str string) {
	b, _ := json.Marshal(s)
	str = string(b)
	return str
}

func (s *Structs) GetRoot() (struc *Struct, exists bool) {
	for _, stru := range *s {
		if stru.IsRoot {
			return stru, true
		}
	}
	return struc, false
}

func (s *Structs) Get(name string) (struc *Struct, exists bool) {
	for _, stru := range *s {
		if stru.Name == name {
			return stru, true
		}
	}
	return struc, false
}

func (s *Structs) AddIngore(structs ...*Struct) {
	if len(*s) == 0 {
		*s = make(Structs, 0)
	}
	for _, structRef := range structs {
		if _, exists := s.Get(structRef.Name); exists {
			continue
		}
		*s = append(*s, structRef)

	}
}

// Copy 深度复制
func (s Structs) Copy() (newStructs Structs) {
	newStructs = make(Structs, 0)
	for _, struc := range s {
		newStruct := *struc
		newStruct.Attrs = make([]*StructAttr, 0)
		for _, attr := range struc.Attrs {
			newAttr := *attr
			newStruct.Attrs = append(newStruct.Attrs, &newAttr)
		}
		newStructs = append(newStructs, &newStruct)
	}
	return newStructs
}

func (s *Structs) AddNameprefix(nameprefix string) {
	if len(*s) == 0 {
		return
	}
	allAttrs := make([]*StructAttr, 0)
	for _, struc := range *s {
		allAttrs = append(allAttrs, struc.Attrs...)
	}
	for _, struc := range *s {
		baseName := struc.Name
		struc.Name = funcs.ToCamel(fmt.Sprintf("%s_%s", nameprefix, baseName))
		if struc.Type != "" {
			typ := struc.Type
			arrayPrefix := "[]"
			hasArrPrefix := strings.Contains(typ, arrayPrefix)
			if hasArrPrefix {
				typ = strings.TrimPrefix(typ, arrayPrefix)
			}
			prefix := ""
			if hasArrPrefix {
				prefix = arrayPrefix
			}
			struc.Type = fmt.Sprintf("%s%s%s", prefix, nameprefix, typ)

		}
		for _, attr := range allAttrs {
			if strings.HasSuffix(attr.Type, baseName) {
				attr.Type = fmt.Sprintf("%s%s", attr.Type[:len(attr.Type)-len(baseName)], struc.Name)
			}
		}
	}
}

// complex2singularName 格式化数组名称
func complex2singularName(name string) (friendlyName string) {
	l := len(name)
	if l == 0 {
		return ""
	}

	//优化列表命名
	if strings.HasSuffix(name, "List") {
		friendlyName = name[:l-4]
	} else if strings.HasSuffix(name, "ies") {
		friendlyName = fmt.Sprintf("%sy", name[:l-3])
	} else if l > 0 && name[l-1] == 's' {
		friendlyName = name[:l-1]
	} else {
		friendlyName = fmt.Sprintf("%sItem", name) //默认增加Item后缀,作为数组元素结构体名称
	}
	return friendlyName
}
