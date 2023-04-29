## HTML-Go -- Packages for Generating HTML from Go

HTML-Go defines a basic interface for appending HTML content to a byte slice, and then provides a number of types that
implement this interface that can efficiently generate HTML in a Go web application.

### A Simple Login Form Example

The following uses the `github.com/swdunlop/html-go/tag` packge to create a simple login form:

```go
tag.New(`main#main`).Add(
    tag.New(`h1`).Text(`Welcome to Example Co!`),
    p.Text(
        `Example Co is in private beta, you have been invited to participate in the beta by "`, inviter, `".`,
    ),
    p.Text(`Since this is your first here, we will need to create an account, but you must read and agree to the`).
        Add(tag.New(`a[href='/terms']`).Text(`terms of service`)).
        Text(`before we can do that.`),
    tag.New(`form`).Set(`action`, `/invite/`+token).Add(
        tag.New(`input[type=hidden][name=csrf]`).Set(`value`, requestCSRF(w, r)),
        tag.New(`input[type=hidden][name=token]`).Set(`value`, token),
        tag.New(`input[type=text][name=username][placeholder="Your-User-Name"]`),
        tag.New(`input[type=checkbox][name=accept][required]`),
        tag.New(`input[type=submit][value="Create Account"]`),
    ),
)
```

This form implements the `html-go.Content` interface, which is simply defined as:

```go
type Content interface {
    AppendHTML(dst []byte) []byte
}
```

The `html-go` package also includes a number of lower level types that implement this interface, such as:

- `html.Text` -- A text string which escapes HTML special characters when used as content.
- `html.HTML` -- A byte slice which is assumed to already be valid HTML and simply appends itself as content.
- `html.Group` -- A slice of other HTML content that can be used as content.

### Usage Tips

The [tag](./tag) package was derived from the [`m(selector, attributes, children)`](https://mithril.js.org/hyperscript.html)
function in the [Mithril](https://mithril.js.org/) JavaScript framework and shares some usage patterns:

- Avoid dynamically generating the selector -- instead, use the `Set`, `Add`, and `Text` methods to add dynamic content
  to the selector.
- Tags can be reused, each method returns a copy of the tag with the necessary changes applied.  This is meant to make
  it easy to build up a library of common tags.
- Be careful around HTML tags with really strange rules about their content -- specifically `style` and `script` that
  do not support the use of entities or comments, and `textarea`.  HTML5 is not as uniform as you may expect.

In addition, `html.HTML` is very literal about its contents, it is common and expected that you might concatenate
a number of HTML elements into a single static `html.HTML` value.  You can use the higher level `tag` package to build
a series of complex HTML elements and then use `html.Static` to convert that element to a static `html.HTML` value.

### Generating CDN Tags with Unpkg

This repository also includes [cmd/unpkg](./cmd/unpkg), a simple command line utility for generating script and link tags
for the unpkg.com CDN with SRIs.  Unpkg is great, but it can be a little dicey to figure out the correct request path,
which is important if you also use SRI hashes to ensure the integrity of your dependencies.

```shell
go run ./cmd/unpkg htmx.org@1.9.2 htmx.org@1.9.1/dist/ext/sse.js hyperscript.org@0.9.8 chota
```
```html
<script defer src="https://unpkg.com/htmx.org@1.9.2/dist/htmx.min.js" integrity="sha384-L6OqL9pRWyyFU3+/bjdSri+iIphTN/bvYyM37tICVyOJkWZLpP2vGn6VUEXgzg6h" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
<script defer src="https://unpkg.com/htmx.org@1.9.1/dist/ext/sse.js" integrity="sha384-wQMrQ8lhjmPC6O2HZmiTsqEHeO4hD9lX2A4Q46YGtlaagNrRYVcuf9aJ3y/VN2hs" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
<script defer src="https://unpkg.com/hyperscript.org@0.9.8/dist/_hyperscript.min.js" integrity="sha384-1u4t3o4KScBpVyJ8r7E1vifF4H/GMUeZjN7CYA3v2xMXifSTac20oOseU3Irrup2" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
<link rel="stylesheet" href="https://unpkg.com/chota@0.9.2/dist/chota.min.css" integrity="sha384-A2UBIkgVTcNWgv+snhw7PKvU/L9N0JqHwgwDwyNcbsLiVhGG5KAuR64N4wuDYd99" referrerpolicy="no-referrer" />
```

### Why?

I find the `html/template` package frustrating outside of simple use cases and prefer to generate HTML directly in 
view functions rather than deal with the awkwardness of Go templates.
