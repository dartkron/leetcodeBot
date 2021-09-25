package common

import (
	"testing"
	"time"
)

func TestReplaceImgTagWithA(t *testing.T) {
	tests := map[string]string{
		"test<img src=\"http://mytest.com/test.png\"/>test<img src=\"http://anothermytest.org/pic.jpg\"/>test": "test\n<a href=\"http://mytest.com/test.png\">Picture 0</a>test\n<a href=\"http://anothermytest.org/pic.jpg\">Picture 1</a>test",
	}
	for source, result := range tests {

		calculatedResult := ReplaceImgTagWithA(source)
		if calculatedResult != result {
			t.Fatalf("Wrong transform, got %s, but %s awaited", calculatedResult, result)
		}

	}
}

type DateTestCase struct {
	year  int
	month int
	day   int
}

func TestGetDateId(t *testing.T) {
	loc, _ := time.LoadLocation("US/Pacific")
	testData := map[DateTestCase]uint64{
		{1970, 1, 1}:   19700101,
		{2021, 12, 31}: 20211231,
		{2020, 2, 29}:  20200229,
	}
	for dataPart, result := range testData {
		calculatedResult := GetDateID(time.Date(dataPart.year, time.Month(dataPart.month), dataPart.day, 0, 0, 0, 0, loc))
		if calculatedResult != result {
			t.Errorf("%d awaited, but got %d", result, calculatedResult)
		}
	}
}

func TestRemoveUnsuppotedTags(t *testing.T) {
	cases := map[string]string{
		"test":                   "test",
		"te<p>st":                "test",
		"te</p>st":               "test",
		"t<p>e</p>st":            "test",
		"te<ul>st":               "test",
		"te</ul>st":              "test",
		"t<ul>e</ul>st":          "test",
		"te<li>st":               "te — st",
		"te</li>st":              "test",
		"t<li>e</li>st":          "t — est",
		"te&nbsp;st":             "te st",
		"t&nbsp;e&nbsp;s&nbsp;t": "t e s t",
		"te<sup>st":              "te**st",
		"te</sup>st":             "test",
		"t<sup>e</sup>st":        "t**est",
		"te<sub>st":              "te(st",
		"te</sub>st":             "te)st",
		"t<sub>e</sub>st":        "t(e)st",
		"te<em>st":               "test",
		"te</em>st":              "test",
		"t<em>e</em>st":          "test",
		"te\n\nst":               "test",
		"t<p>e</p>s<ul>t</ul>t<li>e</li>s&nbsp;t<sup>t</sup>e<sub>s</sub>tt<em>e</em>s\n\nt<br>t</br>e</strong>": "testt — es t**te(s)ttestte</strong>",
		"<p></p><ul></ul><li></li>&nbsp;<sup></sup><sub></sub><em></em>\n\n<br></br></strong>":                   " —  **()</strong>",
		"te<br>st":      "test",
		"te</br>st":     "test",
		"t<br>e</br>st": "test",
	}
	for testCase, result := range cases {
		calculatedResult := RemoveUnsupportedTags(testCase)
		if calculatedResult != result {
			t.Errorf("%s case proceeded into %s but %s awaited", testCase, calculatedResult, result)
		}
	}
}
