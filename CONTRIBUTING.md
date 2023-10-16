### Setup your mc Github Repository
Fork [mc upstream](https://github.com/minio/mc/fork) source repository to your own personal repository.
```
$ mkdir -p $GOPATH/src/github.com/minio
$ cd $GOPATH/src/github.com/minio
$ git clone https://github.com/$USER_ID/mc
$ cd mc
$ make
$ ./mc --help
```

###  Developer Guidelines

``mc`` welcomes your contribution. To make the process as seamless as possible, we ask for the following:

* Go ahead and fork the project and make your changes. We encourage pull requests to discuss code changes.
    - Fork it
    - Create your feature branch (git checkout -b my-new-feature)
    - Commit your changes (git commit -am 'Add some feature')
    - Push to the branch (git push origin my-new-feature)
    - Create new Pull Request

* If you have additional dependencies for ``mc``, ``mc`` manages its dependencies using `go mod`
    - Run `go get foo/bar`
    - Edit your code to import foo/bar
    - Run `GO111MODULE=on go mod tidy` from top-level folder

* When you're ready to create a pull request, be sure to:
    - Have test cases for the new code. If you have questions about how to do it, please ask in your pull request.
    - Run `go fmt`
    - Squash your commits into a single commit. `git rebase -i`. It's okay to force update your pull request.
    - Make sure `make install` completes.

* Read [Effective Go](https://github.com/golang/go/wiki/CodeReviewComments) article from Golang project
    - `mc` project is conformant with Golang style
    - if you happen to observe offending code, please feel free to send a pull request
