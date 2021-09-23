# go-accelerator

Multithreaded download accelerator made in Go.

![Demo](demo.gif)

## Usage

Build an executable with:

```console
go build
```

Run the executable:

```console
$ ./go-accelerator [-t] url

-t int
        number of threads (default 12)
```

By default, the numbers of threads is set according to your CPU.
