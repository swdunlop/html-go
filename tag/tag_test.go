package tag

import "testing"

func Test(t *testing.T) {
	test(t, `Empty`, `<div></div>`, func() Interface {
		return New(``)
	})
	test(t, `Div`, `<div></div>`, func() Interface {
		return New(`div`)
	})
	test(t, `Br`, `<br>`, func() Interface {
		return New(`br`)
	})
	test(t, `DivId`, `<div id='one'></div>`, func() Interface {
		return New(`div#one`)
	})
	test(t, `DivStaticId`, `<div id='one'></div>`, func() Interface {
		return New(`div[id=one]`)
	})
	test(t, `DivDynamicId`, `<div id='one'></div>`, func() Interface {
		return New(`div`).Set(`id`, `one`)
	})
	test(t, `DivOverrideId`, `<div id='one'></div>`, func() Interface {
		return New(`div#zero`).Set(`id`, `one`)
	})
	test(t, `DivClass`, `<div class='one'></div>`, func() Interface {
		return New(`div.one`)
	})
	test(t, `DivClasses`, `<div class='one two three'></div>`, func() Interface {
		return New(`div.one.two.three`)
	})
	test(t, `DivStaticClass`, `<div class='one'></div>`, func() Interface {
		return New(`div[class=one]`)
	})
	test(t, `DivDynamicClass`, `<div class='one'></div>`, func() Interface {
		return New(`div`).Set(`class`, `one`)
	})
	test(t, `DivAppendClass`, `<div class='zero one two three'></div>`, func() Interface {
		return New(`div.zero`).Class(`one`, `two`, `three`)
	})
	test(t, `DivOverrideClass`, `<div class='one'></div>`, func() Interface {
		return New(`div.zero`).Set(`class`, `one`)
	})
	test(t, `AStaticHref`, `<a href='http://example.com'>example</a>`, func() Interface {
		return New(`a[href=http://example.com]`).Text(`example`)
	})
	test(t, `ADynamicHref`, `<a href='http://example.com'>example</a>`, func() Interface {
		return New(`a`).Set(`href`, `http://example.com`).Text(`example`)
	})
}

func test(t *testing.T, name string, expect string, do func() Interface) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		got := string(do().AppendHTML(nil))
		t.Log(`generated:`, got)
		if got != expect {
			t.Error(` expected:`, expect)
		}
	})
}
