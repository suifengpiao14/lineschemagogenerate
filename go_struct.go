package lineschemagogenerate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/suifengpiao14/funcs"
	"github.com/suifengpiao14/lineschema"
)

func isStructType(name string, lineschemaItems lineschema.LineschemaItems) (ok bool) {
	for _, item := range lineschemaItems {
		typ := strings.TrimPrefix(item.Type, "[]")
		if typ == name {
			return true
		}
	}
	return false
}

func NewSturct(l lineschema.Lineschema) (structs Structs) {
	arraySuffix := "[]"
	structs = make(Structs, 0)
	id := string(l.Meta.ID)
	rootStructName := funcs.ToCamel(id)
	rootStruct := &Struct{
		IsRoot:     true,
		Name:       rootStructName,
		Attrs:      make(StructAttrs, 0),
		IsTypeName: false,
		Lineschema: l.String(),
	}
	structs.AddIngore(rootStruct)
	for _, item := range l.Items {
		if item.Fullname == "" && item.Type == "" {
			continue
		}
		fullname := item.Fullname
		fullname = strings.Trim(fmt.Sprintf("%s.%s", id, fullname), ".")
		if item.Fullname == "" {
			fullname = "." // fullname为空,追加.
		}
		nameArr := strings.Split(fullname, ".")
		nameCount := len(nameArr)
		for i := 1; i < nameCount; i++ { //i从1开始,0 为root,已处理
			parentStructName := funcs.ToCamel(strings.Join(nameArr[:i], "_"))
			parentStruct, ok := structs.Get(parentStructName) // 一定存在（此处就是占时记录_originBaseName 的原因）
			if !ok {
				parentStruct = rootStruct
			}
			if strings.HasPrefix(parentStruct.Type, arraySuffix) {
				parentStruct, _ = structs.Get(complex2singularName(parentStructName)) //取单数, 一定存在
			}
			baseName := nameArr[i]
			realBaseName := strings.TrimSuffix(baseName, arraySuffix)
			isArray := baseName != realBaseName
			attrName := funcs.ToCamel(realBaseName)
			if i < nameCount-1 { // 非最后一个,即为上级的attr,又为下级的struct
				makeParentStuct(&structs, parentStruct, l.Items, nameArr, i, isArray, attrName, "")
				continue // 非最后一个，在此处结束
			}
			baseTypeArr := "[]int,[]string,[]object,[]number,[]float"                                   // 排除基本类型数组
			if strings.HasPrefix(item.Type, arraySuffix) && !strings.Contains(baseTypeArr, item.Type) { // 类型有[] 直接当做结构体如  fullname=Parameters,type=[]Parameter
				makeParentStuct(&structs, parentStruct, l.Items, nameArr, i, isArray, attrName, item.Type)
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
			if strings.EqualFold(typ, "object") {
				typ = "any"
			}
			// 根据格式，修改类型
			switch format {
			case "int":
				typ = "int"
			case "bool", "boolean":
				typ = "bool"
			}
			tagName := funcs.ToLowerCamel(attrName)
			if tagName == "" {
				tagName = "-"
			}
			tag := fmt.Sprintf(`json:"%s"`, tagName)
			if !item.Required { //当作入参时,非必填字断,使用引用
				typ = fmt.Sprintf("*%s", typ)
			}
			isArray = isArray || strings.ToLower(item.Type) == "array" // 最后一个接受当前的type字段值
			if isArray {
				if typ == "array" {
					typ = "any"
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

	// 将自定义的类型替换成原始名称，并且将相关属性剔除
	allAttrs := make(StructAttrs, 0)
	for _, struc := range structs {
		allAttrs.Add(struc.Attrs...)
	}

	for _, struc := range structs {
		if struc.IsTypeName {
			oldName := struc.Name
			struc.Name = struc._originBaseName        // 已经当做其它结构体属性类型的 子结构体，其名字修改会使用原始名称
			struc._parent.Attrs.RemoveByType(oldName) // 父类属性中移除自动生成的该结构体名称类型对应的属性
		}
	}

	return structs
}

func makeParentStuct(structs *Structs, parentStruct *Struct, lineschemaItems lineschema.LineschemaItems, nameArr []string, i int, isArray bool, attrName string, typ string) {
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
		IsRoot:          false,
		_originBaseName: nameArr[i],
		IsTypeName:      isStructType(nameArr[i], lineschemaItems),
		Name:            subStructName,
		Type:            typ,
		_parent:         parentStruct,
	}
	structs.AddIngore(subStruct)
}

// jsonschemaline 生成go 结构体工具
type Struct struct {
	IsRoot          bool
	Name            string // 这个名称已经增加了命名空间如 in,out 等
	Lineschema      string
	Attrs           StructAttrs
	IsTypeName      bool   // 当前结构体的名称是否为其它结构体字段类型,标记为是时，其名称可以使用originBaseName替换
	_originBaseName string // lineschema 上原始fullname字段最后一个
	Type            string
	_parent         *Struct
}

// AddAttrIgnore 已经存在则跳过
func (s *Struct) AddAttrIgnore(attrs ...StructAttr) {
	if len(s.Attrs) == 0 {
		s.Attrs = make(StructAttrs, 0)
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
		s.Attrs = make(StructAttrs, 0)
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

type StructAttrs []*StructAttr

func (attrs *StructAttrs) Add(ats ...*StructAttr) {
	if *attrs == nil {
		*attrs = make(StructAttrs, 0)
	}
	*attrs = append(*attrs, ats...)
}

// RemoveByType 移除指定类型的属性
func (attrs *StructAttrs) RemoveByType(typeName string) {
	tmp := make(StructAttrs, 0)
	for _, attr := range *attrs {
		if attr.Type == typeName || attr.Type == fmt.Sprintf("[]%s", typeName) {
			continue
		}
		tmp = append(tmp, attr)
	}
	*attrs = tmp
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
		newStruct.Attrs = make(StructAttrs, 0)
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
	allAttrs := make(StructAttrs, 0)
	for _, struc := range *s {
		allAttrs.Add(struc.Attrs...)
	}
	for _, struc := range *s {
		if struc.IsTypeName {
			continue // 当前结构体是值类型，并且已经作为其他结构体属性时，不修改名称
		}
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
