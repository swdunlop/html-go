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

### Simplified HTTP Logging With Chi and Zerolog

This module includes the [hog](./hog/hog.go) package as an alternative to the 
[httplog](https://github.com/go-chi/httplog) package.  Hog is less verbose than httplog and adds a per-request logger to 
the request context for use by request handlers using `http.Request.WithContext` with `zerolog.WithContext` and 
`zerolog.Ctx`.

```go
func main() {
    // Zerolog's default logger uses JSON output, but this makes it more readable (and slower, since zerolog must
    // generate JSON logs as normal and then parse them into text).
    log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
        With().Timestamp().Logger()

    // For some reason, zerolog.DefaultContextLogger is nil out of the box, and
    // this is the default logger used by zerolog.Ctx (and therefore hog.For) --
    // you must bind it to a logger if you want to see log output at all.
    //
    // Alternately, you can specify the logger in the request context before using
    // hog, but it is generally easier to fix the problem at the source.
    zerolog.DefaultContextLogger = &log.Logger

    r := chi.NewRouter()
    r.Use(hog.Middleware())
    r.Get("/lazy", func(w http.ResponseWriter, r *http.Request) {
        hog.For(r).Info().Msg("taking a nap..")
        time.Sleep(1 * time.Second)
        http.Error(w, "I'm awake!", http.StatusOK)
    })
    http.ListenAndServe(":8080", r)
}
```
```
~/swdunlop/html-go> go run ./examples/lazy
10:58PM INF taking a nap.. method=GET remote_addr=127.0.0.1:60899 path=/lazy
10:58PM INF method=GET remote_addr=127.0.0.1:60899 path=/lazy status=200 took=1001 wrote=11
```

You can access the injected logger with `hog.For(r)` from a request or `hog.From(ctx)` from a context.  The `For`, 
`From`, and `Middleware` functions all accept a series of options that can be used to customize the logger.

**WARNING**: The `hog` package will include the URL request path (but not the query) in the log output by default.  This
may be a security concern for handlers like invite links that include sensitive information in the URL path.  You will 
want to avoid using `hog.For`, `hog.From` and `hog.Middleware` for these handlers.

### Why?

I find the `html/template` package frustrating outside of simple use cases and prefer to generate HTML directly in 
view functions rather than deal with the awkwardness of Go templates.  There are a lot of other interesting template
languages for Go but they all have their own quirks and I find myself dropping down to writing Go functions anyway.

Everything else in this package is just a collection of utilities for making life easier once you have decided to 
write your UI in Go.