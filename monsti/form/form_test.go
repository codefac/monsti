package form

import (
	"net/url"
	"reflect"
	"testing"
)

type TestData struct {
	Name string
	Age  int
}

func TestRender(t *testing.T) {
	data := TestData{}
	form := NewForm(&data, Fields{
		"Name": Field{"Your name", "Your full name", Required(), nil},
		"Age":  Field{"Your age", "Years since your birth.", Required(), nil}})
	vals := url.Values{
		"Name": []string{""},
		"Age":  []string{"14"}}
	form.Fill(vals)
	renderData := form.RenderData()
	fieldTests := []struct {
		Field    string
		Expected FieldRenderData
	}{
		{
			Field: "Name",
			Expected: FieldRenderData{
				Label:    "Your name",
				LabelTag: `<label for="name">Your name</label>`,
				Help:     "Your full name",
				Errors:   []string{"Required."},
				Input:    `<input id="name" type="text" name="Name" value=""/>`}},
		{
			Field: "Age",
			Expected: FieldRenderData{
				Label:    "Your age",
				LabelTag: `<label for="age">Your age</label>`,
				Help:     "Years since your birth.",
				Errors:   nil,
				Input:    `<input id="age" type="text" name="Age" value="14"/>`}}}
	for i, test := range fieldTests {
		if !reflect.DeepEqual(renderData.Fields[i], test.Expected) {
			t.Errorf("RenderData for Field '%v' =\n%v,\nexpected\n%v",
				test.Field, renderData.Fields[i], test.Expected)
		}
	}
}

func TestRequire(t *testing.T) {
	invalid, valid := "", "foo"
	validator := Required()
	err := validator(valid)
	if err != nil {
		t.Errorf("require(%v) = %v, want %v", valid, err, nil)
	}
	err = validator(invalid)
	if err == nil {
		t.Errorf("require(%v) = %v, want %v", invalid, err, "'Required.'")
	}
}

func TestRegex(t *testing.T) {
	tests := []struct {
		Exp    string
		String string
		Valid  bool
	}{
		{"^[\\w]+$", "", false},
		{"^[\\w]+$", "foobar", true},
		{"", "", true},
		{"", "foobar", true},
		{"^[^!]+$", "foobar", true},
		{"^[^!]+$", "foo!bar", false}}

	for _, v := range tests {
		ret := Regex(v.Exp, "damn!")(v.String)
		if (ret == nil && !v.Valid) || (ret != nil && v.Valid) {
			t.Errorf(`Regex("%v")("%v") = %v, this is wrong!`, v.Exp, v.String,
				ret)
		}
	}
}

func TestAnd(t *testing.T) {
	tests := []struct {
		String     string
		Validators []Validator
		Valid      bool
	}{
		{"Hey! 1", []Validator{Required()}, true},
		{"", []Validator{Required()}, false},
		{"Hey! 2", []Validator{Required(), Regex("Oink", "No way!")}, false},
		{"Hey! 3", []Validator{Required(), Regex("Hey", "No way!")}, true}}
	for _, v := range tests {
		ret := And(v.Validators...)(v.String)
		if (ret == nil && !v.Valid) || (ret != nil && v.Valid) {
			t.Errorf(`And(...)("%v") = %v, this is wrong!`, v.String, ret)
		}
	}
}
