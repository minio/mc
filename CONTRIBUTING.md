### Setup your minioc Github Repository
Fork [minioc upstream](https://github.com/minio/minioc/fork) source repository to your own personal repository.
```sh
$ mkdir -p $GOPATH/src/github.com/minio
$ cd $GOPATH/src/github.com/minio
$ git clone https://github.com/$USER_ID/minioc
$ cd minioc
$ make
$ minioc --help
```

###  Developer Guidelines

``minioc`` welcomes your contribution. To make the process as seamless as possible, we ask for the following:

* Go ahead and fork the project and make your changes. We encourage pull requests to discuss code changes.
    - Fork it
    - Create your feature branch (git checkout -b my-new-feature)
    - Commit your changes (git commit -am 'Add some feature')
    - Push to the branch (git push origin my-new-feature)
    - Create new Pull Request

* If you have additional dependencies for ``minioc``, ``minioc`` manages its dependencies using [govendor](https://github.com/kardianos/govendor)
    - Run `go get foo/bar`
    - Edit your code to import foo/bar
    - Run `make pkg-add PKG=foo/bar` from top-level folder

* If you have dependencies which needs to be removed from ``minioc``
    - Edit your code to not import foo/bar
    - Run `make pkg-remove PKG=foo/bar` from top-level folder

* When you're ready to create a pull request, be sure to:
    - Have test cases for the new code. If you have questions about how to do it, please ask in your pull request.
    - Run `go fmt`
    - Squash your commits into a single commit. `git rebase -i`. It's okay to force update your pull request.
    - Make sure `make build` completes.

* Read [Effective Go](https://github.com/golang/go/wiki/CodeReviewComments) article from Golang project
    - `minioc` project is conformant with Golang style
    - if you happen to observe offending code, please feel free to send a pull request
