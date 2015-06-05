// This file is part of Monsti, a web content management system.
// Copyright 2012-2015 Christian Neumann
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/chrneumann/htmlwidgets"
	"pkg.monsti.org/gettext"
	"pkg.monsti.org/monsti/api/util/i18n"
)

func init() {
	gob.Register(new(TextFieldType))
	gob.Register(new(HTMLFieldType))
	gob.Register(new(BoolFieldType))
	gob.Register(new(DateTimeFieldType))
	gob.Register(new(FileFieldType))
	gob.Register(new(RefFieldType))
	gob.Register(new(ListFieldType))
	gob.Register(new(MapFieldType))
	gob.Register(new(CombinedFieldType))
	gob.Register(new(IntegerFieldType))
	gob.Register(new(DynamicTypeFieldType))
	gob.Register(new(DummyFieldType))
}

type NestedMap map[string]interface{}

func (n NestedMap) Get(id string) interface{} {
	parts := strings.Split(id, ".")
	field := interface{}(map[string]interface{}(n))
	for _, part := range parts {
		var ok bool
		field, ok = field.(map[string]interface{})[part]
		if !ok {
			return nil
		}
	}
	return field
}

func (n NestedMap) Set(id string, value interface{}) {
	parts := strings.Split(id, ".")
	field := interface{}(map[string]interface{}(n))
	for _, part := range parts[:len(parts)-1] {
		next := field.(map[string]interface{})[part]
		if next == nil {
			next = make(map[string]interface{})
			field.(map[string]interface{})[part] = next
		}
		field = next
	}
	field.(map[string]interface{})[parts[len(parts)-1]] = value
}

type Field interface {
	// Init initializes the field.
	Init(*MonstiClient, string) error
	// RenderHTML returns a string or template.HTML to be used in a html
	// template.
	RenderHTML() interface{}
	// Value returns the value of the field, e.g. a boolean value for
	// Bool fields.
	Value() interface{}
	// Load loads the field data using the given function (see also `Dump`).
	//
	// The passed function unmarshals the raw value (as returned by an
	// earlier `Dump`) into the given value.
	Load(func(interface{}) error) error
	// Dump dumps the field data.
	//
	// The dumped value must be something that can be marshalled into
	// JSON by encoding/json.
	Dump() interface{}
	// Adds a form field to the given form.
	//
	// The nested map stores the field values used by the form. Locale
	// is used for translations.
	ToFormField(form *htmlwidgets.Form, values NestedMap, field *FieldConfig,
		locale string)
	// Load values from the form submission
	FromFormField(NestedMap, *FieldConfig)

	// TODO Replace ToFormField and FromFormField using the Form* methods.
	// Needed for rendering of nested fields (see ListField).
	FormWidget() htmlwidgets.Widget
	FormData() interface{}
	FromFormData(data interface{})
}

type BoolFieldType int

func (_ BoolFieldType) Field() Field {
	return new(BoolField)
}

// BoolField is a basic boolean field rendered as checkbox.
type BoolField bool

func (t BoolField) Init(*MonstiClient, string) error {
	return nil
}

func (t BoolField) Value() interface{} {
	return bool(t)
}

func (t BoolField) RenderHTML() interface{} {
	return t.Value()
}

func (t *BoolField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t BoolField) Dump() interface{} {
	return t
}

func (t BoolField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, t)
	form.AddWidget(new(htmlwidgets.BoolWidget), "Fields."+field.Id,
		field.Name.Get(locale), "")
}

func (t *BoolField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = BoolField(data.Get(field.Id).(bool))
}

func (t *BoolField) Bool() bool {
	return bool(*t)
}

func (f BoolField) FormData() interface{} {
	panic("Not implemented")
}

func (f *BoolField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f BoolField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type CombinedFieldType struct {
	Fields map[string]FieldConfig
}

func (t CombinedFieldType) Field() Field {
	return &CombinedField{fieldType: &t}
}

type CombinedField struct {
	Fields    map[string]Field
	fieldType *CombinedFieldType
	monsti    *MonstiClient
	site      string
}

func (f *CombinedField) Init(m *MonstiClient, site string) error {
	f.Fields = make(map[string]Field)
	f.monsti = m
	f.site = site
	return nil
}

func (f CombinedField) RenderHTML() interface{} {
	var out []interface{}
	for k, field := range f.Fields {
		out = append(out, fmt.Sprintf("%v:", k), field.RenderHTML())
	}
	return out
}

func (f CombinedField) Value() interface{} {
	return f.Fields
}

func (f *CombinedField) Load(dataFnc func(interface{}) error) error {
	var data map[string]json.RawMessage
	if err := dataFnc(&data); err != nil {
		return err
	}
	for k, msg := range data {
		fieldDataFnc := func(in interface{}) error {
			return json.Unmarshal(msg, in)
		}
		field := f.fieldType.Fields[k].Type.Field()
		field.Init(f.monsti, f.site)
		if err := field.Load(fieldDataFnc); err != nil {
			return fmt.Errorf("Could not parse combined field data: %v", err)
		}
		f.Fields[k] = field
	}
	return nil
}

func (f CombinedField) Dump() interface{} {
	out := make(map[string]interface{})
	for k, field := range f.Fields {
		out[k] = field.Dump()
	}
	return out
}

func (f CombinedField) FormData() interface{} {
	panic("Not implemented")
	return nil
}

func (f *CombinedField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f CombinedField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

func (t CombinedField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
}

func (t *CombinedField) FromFormField(data NestedMap, field *FieldConfig) {
}

type DummyFieldType int

func (_ DummyFieldType) Field() Field {
	return new(DummyField)
}

// DummyField is a placeholder field that does nothing.
type DummyField int

func (t DummyField) Init(*MonstiClient, string) error {
	return nil
}

func (t DummyField) Value() interface{} {
	return nil
}

func (t DummyField) RenderHTML() interface{} {
	return nil
}

func (t *DummyField) Load(f func(interface{}) error) error {
	return nil
}

func (t DummyField) Dump() interface{} {
	return nil
}

func (t DummyField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
}

func (t *DummyField) FromFormField(data NestedMap, field *FieldConfig) {
}

func (f DummyField) FormData() interface{} {
	return nil
}

func (f *DummyField) FromFormData(data interface{}) {
}

func (f DummyField) FormWidget() htmlwidgets.Widget {
	return nil
}

// DynamicTypeFieldType describes a field that can hold a field of one
// of the available types.
type DynamicTypeFieldType struct {
	// Holds the available types.
	Fields []FieldConfig
}

func (t DynamicTypeFieldType) Field() Field {
	return &DynamicTypeField{fieldType: &t}
}

// DynamicTypeField holds a field of arbitrary type.
type DynamicTypeField struct {
	// Stores the actual field.
	Field Field
	// The id of the field's config as defined in fieldType.Fields.
	DynamicType string
	fieldType   *DynamicTypeFieldType
	monsti      *MonstiClient
	site        string
}

func (f *DynamicTypeField) Init(m *MonstiClient, site string) error {
	f.monsti = m
	f.site = site
	return nil
}

func (f DynamicTypeField) RenderHTML() interface{} {
	return f.Field.RenderHTML()
}

func (f DynamicTypeField) Value() interface{} {
	return f.Field
}

type dynamicTypeFieldJSON struct {
	Type string
	Data interface{}
}

func (f *DynamicTypeField) Load(dataFnc func(interface{}) error) error {
	var data dynamicTypeFieldJSON
	data.Data = json.RawMessage{}
	if err := dataFnc(&data); err != nil {
		return err
	}
	fieldDataFnc := func(in interface{}) error {
		return json.Unmarshal(data.Data.([]byte), in)
	}
	var field Field
	for _, v := range f.fieldType.Fields {
		if v.Id == data.Type {
			field = v.Type.Field()
			break
		}
	}
	if field == nil {
		return fmt.Errorf("Unknown field type in DynamicField: %v", data.Type)
	}
	f.DynamicType = data.Type
	field.Init(f.monsti, f.site)
	if err := field.Load(fieldDataFnc); err != nil {
		return fmt.Errorf("Could not parse combined field data: %v", err)
	}
	f.Field = field
	return nil
}

func (f DynamicTypeField) Dump() interface{} {
	var out dynamicTypeFieldJSON
	out.Type = f.DynamicType
	out.Data = f.Field.Dump()
	return out
}

func (f DynamicTypeField) FormData() interface{} {
	panic("Not implemented")
	return nil
}

func (f *DynamicTypeField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f DynamicTypeField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

func (t DynamicTypeField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
}

func (t *DynamicTypeField) FromFormField(data NestedMap, field *FieldConfig) {
}

type RefFieldType int

func (_ RefFieldType) Field() Field {
	return new(RefField)
}

// RefField contains a reference to another node.
type RefField string

func (t RefField) Init(*MonstiClient, string) error {
	return nil
}

func (t RefField) Value() interface{} {
	return string(t)
}

func (t RefField) RenderHTML() interface{} {
	return t
}

func (t *RefField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t RefField) Dump() interface{} {
	return string(t)
}

func (t RefField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, string(t))
	G, _, _, _ := gettext.DefaultLocales.Use("", locale)
	widget := new(htmlwidgets.TextWidget)
	if field.Required {
		widget.MinLength = 1
		widget.ValidationError = G("Required.")
	}
	form.AddWidget(widget, "Fields."+field.Id, field.Name.Get(locale), "")
}

func (t *RefField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = RefField(data.Get(field.Id).(string))
}

func (f RefField) FormData() interface{} {
	panic("Not implemented")
}

func (f *RefField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f RefField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type IntegerFieldType int

func (_ IntegerFieldType) Field() Field {
	return new(IntegerField)
}

// IntegerField is a basic integer field.
type IntegerField int

func (t IntegerField) Init(*MonstiClient, string) error {
	return nil
}

func (t IntegerField) Value() interface{} {
	return int(t)
}

func (t IntegerField) RenderHTML() interface{} {
	return strconv.Itoa(int(t))
}

func (t *IntegerField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t IntegerField) Dump() interface{} {
	return int(t)
}

func (t IntegerField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, int(t))
	G, _, _, _ := gettext.DefaultLocales.Use("", locale)
	widget := new(htmlwidgets.TextWidget)
	if field.Required {
		widget.MinLength = 1
		widget.ValidationError = G("Required.")
	}
	form.AddWidget(widget, "Fields."+field.Id, field.Name.Get(locale), "")
}

func (t *IntegerField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = IntegerField(data.Get(field.Id).(int))
}

func (f IntegerField) FormData() interface{} {
	panic("Not implemented")
}

func (f *IntegerField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f IntegerField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type TextFieldType int

func (_ TextFieldType) Field() Field {
	return new(TextField)
}

// TextField is a basic unicode text field
type TextField string

func (t TextField) Init(*MonstiClient, string) error {
	return nil
}

func (t TextField) Value() interface{} {
	return string(t)
}

func (t TextField) RenderHTML() interface{} {
	return t
}

func (t *TextField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t TextField) Dump() interface{} {
	return string(t)
}

func (t TextField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, string(t))
	G, _, _, _ := gettext.DefaultLocales.Use("", locale)
	widget := new(htmlwidgets.TextWidget)
	if field.Required {
		widget.MinLength = 1
		widget.ValidationError = G("Required.")
	}
	form.AddWidget(widget, "Fields."+field.Id, field.Name.Get(locale), "")
}

func (t *TextField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = TextField(data.Get(field.Id).(string))
}

func (f TextField) FormData() interface{} {
	panic("Not implemented")
}

func (f *TextField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f TextField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type HTMLFieldType int

func (_ HTMLFieldType) Field() Field {
	return new(HTMLField)
}

// HTMLField is a text area containing HTML code
type HTMLField string

func (t HTMLField) Init(*MonstiClient, string) error {
	return nil
}

func (t HTMLField) Value() interface{} {
	return string(t)
}

func (t HTMLField) RenderHTML() interface{} {
	return template.HTML(t)
}

func (t *HTMLField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t HTMLField) Dump() interface{} {
	return string(t)
}

func (t HTMLField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	//G, _, _, _ := gettext.DefaultLocales.Use("", locale)
	data.Set(field.Id, string(t))
	widget := form.AddWidget(new(htmlwidgets.TextAreaWidget), "Fields."+field.Id,
		field.Name.Get(locale), "")
	widget.Base().Classes = []string{"html-field"}
}

func (t *HTMLField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = HTMLField(data.Get(field.Id).(string))
}

func (f HTMLField) FormData() interface{} {
	panic("Not implemented")
}

func (f *HTMLField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f HTMLField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type FileFieldType int

func (_ FileFieldType) Field() Field {
	return new(FileField)
}

type FileField string

func (t FileField) Init(*MonstiClient, string) error {
	return nil
}

func (t FileField) Value() interface{} {
	return string(t)
}

func (t FileField) RenderHTML() interface{} {
	return template.HTML(t)
}

func (t *FileField) Load(f func(interface{}) error) error {
	return f(t)
}

func (t FileField) Dump() interface{} {
	return ""
}

func (t FileField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, "")
	form.AddWidget(new(htmlwidgets.FileWidget), "Fields."+field.Id,
		field.Name.Get(locale), "")
}

func (t *FileField) FromFormField(data NestedMap, field *FieldConfig) {
	*t = FileField(data.Get(field.Id).(string))
}

func (f FileField) FormData() interface{} {
	panic("Not implemented")
}

func (f *FileField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f FileField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

type DateTimeFieldType int

func (_ DateTimeFieldType) Field() Field {
	return &DateTimeField{}
}

type DateTimeField struct {
	Time     time.Time
	Location *time.Location
}

func (t *DateTimeField) Init(m *MonstiClient, site string) error {
	settings, err := m.LoadSiteSettings(site)
	if err != nil {
		return fmt.Errorf("Could not get timezone: %v", err)
	}
	t.Location, err = time.LoadLocation(settings.StringValue("core.Timezone"))
	if err != nil {
		t.Location = time.UTC
	}
	return nil
}

func (t DateTimeField) RenderHTML() interface{} {
	return t.Time.String()
}

func (t DateTimeField) Value() interface{} {
	return t.Time
}

func (t *DateTimeField) Load(f func(interface{}) error) error {
	var date string
	if err := f(&date); err != nil {
		return err
	}
	val, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return fmt.Errorf("Could not parse the date value: %v", err)
	}
	t.Time = val.In(t.Location)
	return nil
}

func (t DateTimeField) Dump() interface{} {
	return t.Time.UTC().Format(time.RFC3339)
}

func (t DateTimeField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	data.Set(field.Id, t.Time)
	form.AddWidget(&htmlwidgets.TimeWidget{Location: t.Location},
		"Fields."+field.Id, field.Name.Get(locale), "")
}

func (t *DateTimeField) FromFormField(data NestedMap, field *FieldConfig) {
	time := data.Get(field.Id).(time.Time)
	*t = DateTimeField{Time: time}
}

func (f DateTimeField) FormData() interface{} {
	panic("Not implemented")
}

func (f *DateTimeField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f DateTimeField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

func initFields(fields map[string]Field, configs []*FieldConfig,
	m *MonstiClient, site string) error {
	for _, config := range configs {
		val := config.Type.Field()
		err := val.Init(m, site)
		if err != nil {
			return fmt.Errorf("Could not init field %q: %v", config.Id, err)
		}
		fields[config.Id] = val
	}
	return nil
}

type ListFieldType struct {
	ElementType FieldType
}

func (t ListFieldType) Field() Field {
	return &ListField{fieldType: &t}
}

type ListField struct {
	Fields    []Field
	fieldType FieldType
	monsti    *MonstiClient
	site      string
}

func (f *ListField) Init(m *MonstiClient, site string) error {
	f.monsti = m
	f.site = site
	return nil
}

func (f ListField) RenderHTML() interface{} {
	var out []interface{}
	for _, field := range f.Fields {
		out = append(out, field.RenderHTML())
	}
	return out
}

func (f ListField) Value() interface{} {
	return f.Fields
}

func (f *ListField) Load(dataFnc func(interface{}) error) error {
	var data []json.RawMessage
	if err := dataFnc(&data); err != nil {
		return err
	}
	elementType := f.fieldType.(*ListFieldType).ElementType
	for _, msg := range data {
		fieldDataFnc := func(in interface{}) error {
			return json.Unmarshal(msg, in)
		}
		field := elementType.Field()
		field.Init(f.monsti, f.site)
		if err := field.Load(fieldDataFnc); err != nil {
			return fmt.Errorf("Could not parse the date value: %v", err)
		}
		f.Fields = append(f.Fields, field)
	}
	return nil
}

func (f ListField) Dump() interface{} {
	var out []interface{}
	for _, field := range f.Fields {
		out = append(out, field.Dump())
	}
	return out
}

func (f ListField) FormData() interface{} {
	var out []interface{}
	for _, field := range f.Fields {
		out = append(out, field.FormData())
	}
	return out
}

func (f *ListField) FromFormData(data interface{}) {
	dataList := data.([]interface{})
	if len(dataList) != len(f.Fields) {
		panic("Implement me!")
	}
	for idx, field := range f.Fields {
		field.FromFormData(dataList[idx])
	}
}

func (f ListField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
	/*
		return &htmlwidgets.ListWidget{
			InnerWidget: &htmlwidgets.TextWidget{},
		}
	*/
}

func (t ListField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
	/*
		data.Set(field.Id, t.Time)
		form.AddWidget(&htmlwidgets.TimeWidget{Location: t.Location},
			"Fields."+field.Id, field.Name.Get(locale), "")
	*/
}

func (t *ListField) FromFormField(data NestedMap, field *FieldConfig) {
	/*
		time := data.Get(field.Id).(time.Time)
		*t = DateTimeField{Time: time}
	*/
}

type MapFieldType struct {
	ElementType FieldType
}

func (t MapFieldType) Field() Field {
	return &MapField{fieldType: &t}
}

type MapField struct {
	Fields    map[string]Field
	fieldType FieldType
	monsti    *MonstiClient
	site      string
}

func (f *MapField) Init(m *MonstiClient, site string) error {
	f.Fields = make(map[string]Field)
	f.monsti = m
	f.site = site
	return nil
}

func (f MapField) RenderHTML() interface{} {
	var out []interface{}
	for k, field := range f.Fields {
		out = append(out, fmt.Sprintf("%v:", k), field.RenderHTML())
	}
	return out
}

func (f MapField) Value() interface{} {
	return f.Fields
}

func (f *MapField) Load(dataFnc func(interface{}) error) error {
	var data map[string]json.RawMessage
	if err := dataFnc(&data); err != nil {
		return err
	}
	elementType := f.fieldType.(*MapFieldType).ElementType
	for k, msg := range data {
		fieldDataFnc := func(in interface{}) error {
			return json.Unmarshal(msg, in)
		}
		field := elementType.Field()
		field.Init(f.monsti, f.site)
		if err := field.Load(fieldDataFnc); err != nil {
			return fmt.Errorf("Could not parse map data: %v", err)
		}
		f.Fields[k] = field
	}
	return nil
}

func (f MapField) Dump() interface{} {
	out := make(map[string]interface{})
	for k, field := range f.Fields {
		out[k] = field.Dump()
	}
	return out
}

func (f MapField) FormData() interface{} {
	panic("Not implemented")
	return nil
}

func (f *MapField) FromFormData(data interface{}) {
	panic("Not implemented")
}

func (f MapField) FormWidget() htmlwidgets.Widget {
	panic("Not implemented")
	return nil
}

func (t MapField) ToFormField(form *htmlwidgets.Form, data NestedMap,
	field *FieldConfig, locale string) {
}

func (t *MapField) FromFormField(data NestedMap, field *FieldConfig) {
}

type FieldType interface {
	// Field returns a new field for the type.
	Field() Field
}

// FieldConfig is the configuration of a field.
type FieldConfig struct {
	// The id of the field, e.g. `core.Title`.
	Id string
	// The type of the field.
	Type FieldType
	// The name of the field as shown in the web interface.
	Name i18n.LanguageMap
	// True if the user has to set this field (if applicable).
	Required bool
	// Hidden fields won't show up in the web interface.
	Hidden bool
}
