### Setup your objectstorage-go Github Repository
Fork [objectstorage-go upstream](https://github.com/minio/objectstorage-go/fork) source repository to your own personal repository.
```sh
$ git clone https://github.com/$USER_ID/objectstorage-go
$ cd objectstorage-go
$ make
$ objectstorage-go --help
```

###  Developer Guidelines

``objectstorage-go`` welcomes your contribution. To make the process as seamless as possible, we ask for the following:

* Go ahead and fork the project and make your changes. We encourage pull requests to discuss code changes.
    - Fork it
    - Create your feature branch (git checkout -b my-new-feature)
    - Commit your changes (git commit -am 'Add some feature')
    - Push to the branch (git push origin my-new-feature)
    - Create new Pull Request

* If you have additional dependencies for ``objectstorage-go``, ``objectstorage-go`` manages its depedencies using [godep](https://github.com/tools/godep)
    - Run `go get foo/bar`
    - Edit your code to import foo/bar
    - Run `make save` from top-level directory (or `godep restore && godep save ./...`).

* When you're ready to create a pull request, be sure to:
    - Have test cases for the new code. If you have questions about how to do it, please ask in your pull request.
    - Run `go fmt`
    - Squash your commits into a single commit. `git rebase -i`. It's okay to force update your pull request.
    - Make sure `go test -race ./...` and `go build` completes.

* Read [Effective Go](https://github.com/golang/go/wiki/CodeReviewComments) article from Golang project
    - `objectstorage-go` project is strictly conformant with Golang style
    - if you happen to observe offending code, please feel free to send a pull request
