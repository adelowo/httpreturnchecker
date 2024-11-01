# httpreturnchecker

`httpreturnchecker` is a static analysis tool that checks if you forget to
correctly use a `return` after writing to the `w` variable - `http.ResponseWriter`

## Install

```sh

go install github.com/adelowo/httpreturnchecker

```

## How to use

```sh
go vet -vettool=$(which httpreturnchecker) path/to/file.go

```

### Notices

It only handles scenarios like this:

- `fmt.Fprint` family of functions
- `io.Copy`
- `w.Write()`
- `render.Render` using github.com/go-chi/render

### Example

```go

// OK - return here
func handler1(w http.ResponseWriter, r *http.Request) {
 w.Write([]byte("hello"))
 return
}

// OK - last statement
func handler2(w http.ResponseWriter, r *http.Request) {
 w.Write([]byte("hello"))
}

// Bad - write with no return and not last statement
func handler3(w http.ResponseWriter, r *http.Request) {
 w.Write([]byte("hello"))
 someOtherOperation()
}

```
